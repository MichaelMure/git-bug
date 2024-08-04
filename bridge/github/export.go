package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
	"golang.org/x/sync/errgroup"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/entities/common"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
)

var (
	ErrMissingIdentityToken = errors.New("missing identity token")
)

// githubExporter implement the Exporter interface
type githubExporter struct {
	conf core.Configuration

	// cache identities clients
	identityClient map[entity.Id]*rateLimitHandlerClient

	// the client to use for non user-specific queries
	// it's the client associated to the "default-login" config
	// used for the github V4 API (graphql)
	defaultClient *rateLimitHandlerClient

	// the token of the default user
	// it's the token associated to the "default-login" config
	// used for the github V3 API (REST)
	defaultToken *auth.Token

	// github repository ID
	repositoryID string

	// cache identifiers used to speed up exporting operations
	// cleared for each bug
	cachedOperationIDs map[entity.Id]string

	// cache labels used to speed up exporting labels events
	cachedLabels map[string]string

	// channel to send export results
	out chan<- core.ExportResult
}

// Init .
func (ge *githubExporter) Init(_ context.Context, repo *cache.RepoCache, conf core.Configuration) error {
	ge.conf = conf
	ge.identityClient = make(map[entity.Id]*rateLimitHandlerClient)
	ge.cachedOperationIDs = make(map[entity.Id]string)
	ge.cachedLabels = make(map[string]string)

	// preload all clients
	err := ge.cacheAllClient(repo)
	if err != nil {
		return err
	}

	return nil
}

func (ge *githubExporter) cacheAllClient(repo *cache.RepoCache) error {
	creds, err := auth.List(repo, auth.WithTarget(target), auth.WithKind(auth.KindToken))
	if err != nil {
		return err
	}

	for _, cred := range creds {
		login, ok := cred.GetMetadata(auth.MetaKeyLogin)
		if !ok {
			_, _ = fmt.Fprintf(os.Stderr, "credential %s is not tagged with a Github login\n", cred.ID().Human())
			continue
		}

		user, err := repo.Identities().ResolveIdentityImmutableMetadata(metaKeyGithubLogin, login)
		if entity.IsErrNotFound(err) {
			continue
		}
		if err != nil {
			return nil
		}

		if _, ok := ge.identityClient[user.Id()]; ok {
			continue
		}

		client := buildClient(creds[0].(*auth.Token))
		ge.identityClient[user.Id()] = client

		// assign the default client and token as well
		if ge.defaultClient == nil && login == ge.conf[confKeyDefaultLogin] {
			ge.defaultClient = client
			ge.defaultToken = creds[0].(*auth.Token)
		}
	}

	if ge.defaultClient == nil {
		return fmt.Errorf("no token found for the default login \"%s\"", ge.conf[confKeyDefaultLogin])
	}

	return nil
}

// getClientForIdentity return a githubv4 API client configured with the access token of the given identity.
func (ge *githubExporter) getClientForIdentity(userId entity.Id) (*rateLimitHandlerClient, error) {
	client, ok := ge.identityClient[userId]
	if ok {
		return client, nil
	}

	return nil, ErrMissingIdentityToken
}

// ExportAll export all event made by the current user to Github
func (ge *githubExporter) ExportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ExportResult, error) {
	out := make(chan core.ExportResult)
	ge.out = out

	var err error
	// get repository node id
	ge.repositoryID, err = getRepositoryNodeID(
		ctx,
		ge.defaultToken,
		ge.conf[confKeyOwner],
		ge.conf[confKeyProject],
	)
	if err != nil {
		return nil, err
	}

	go func() {
		defer close(out)

		// query all labels
		err = ge.cacheGithubLabels(ctx, ge.defaultClient)
		if err != nil {
			out <- core.NewExportError(errors.Wrap(err, "can't obtain Github labels"), "")
			return
		}

		allIdentitiesIds := make([]entity.Id, 0, len(ge.identityClient))
		for id := range ge.identityClient {
			allIdentitiesIds = append(allIdentitiesIds, id)
		}

		allBugsIds := repo.Bugs().AllIds()

		for _, id := range allBugsIds {
			b, err := repo.Bugs().Resolve(id)
			if err != nil {
				out <- core.NewExportError(errors.Wrap(err, "can't load bug"), id)
				return
			}

			select {

			case <-ctx.Done():
				// stop iterating if context cancel function is called
				return

			default:
				snapshot := b.Compile()

				// ignore issues created before since date
				// TODO: compare the Lamport time instead of using the unix time
				if snapshot.CreateTime.Before(since) {
					out <- core.NewExportNothing(b.Id(), "bug created before the since date")
					continue
				}

				if snapshot.HasAnyActor(allIdentitiesIds...) {
					// try to export the bug and it associated events
					ge.exportBug(ctx, b, out)
				}
			}
		}
	}()

	return out, nil
}

// exportBug publish bugs and related events
func (ge *githubExporter) exportBug(ctx context.Context, b *cache.BugCache, out chan<- core.ExportResult) {
	snapshot := b.Compile()
	var bugUpdated bool

	var bugGithubID string
	var bugGithubURL string

	// Special case:
	// if a user try to export a bug that is not already exported to Github (or imported
	// from Github) and we do not have the token of the bug author, there is nothing we can do.

	// first operation is always createOp
	createOp := snapshot.Operations[0].(*bug.CreateOperation)
	author := snapshot.Author

	// skip bug if origin is not allowed
	origin, ok := snapshot.GetCreateMetadata(core.MetaKeyOrigin)
	if ok && origin != target {
		out <- core.NewExportNothing(b.Id(), fmt.Sprintf("issue tagged with origin: %s", origin))
		return
	}

	// get github bug ID
	githubID, ok := snapshot.GetCreateMetadata(metaKeyGithubId)
	if ok {
		githubURL, ok := snapshot.GetCreateMetadata(metaKeyGithubUrl)
		if !ok {
			// if we find github ID, github URL must be found too
			err := fmt.Errorf("incomplete Github metadata: expected to find issue URL")
			out <- core.NewExportError(err, b.Id())
		}

		// extract owner and project
		owner, project, err := splitURL(githubURL)
		if err != nil {
			err := fmt.Errorf("bad project url: %v", err)
			out <- core.NewExportError(err, b.Id())
			return
		}

		// ignore issue coming from other repositories
		if owner != ge.conf[confKeyOwner] && project != ge.conf[confKeyProject] {
			out <- core.NewExportNothing(b.Id(), fmt.Sprintf("skipping issue from url:%s", githubURL))
			return
		}

		// will be used to mark operation related to a bug as exported
		bugGithubID = githubID
		bugGithubURL = githubURL

	} else {
		// check that we have a token for operation author
		client, err := ge.getClientForIdentity(author.Id())
		if err != nil {
			// if bug is still not exported and we do not have the author stop the execution
			out <- core.NewExportNothing(b.Id(), fmt.Sprintf("missing author token"))
			return
		}

		// create bug
		id, url, err := ge.createGithubIssue(ctx, client, ge.repositoryID, createOp.Title, createOp.Message)
		if err != nil {
			err := errors.Wrap(err, "exporting github issue")
			out <- core.NewExportError(err, b.Id())
			return
		}

		out <- core.NewExportBug(b.Id())

		// mark bug creation operation as exported
		if err := markOperationAsExported(b, createOp.Id(), id, url); err != nil {
			err := errors.Wrap(err, "marking operation as exported")
			out <- core.NewExportError(err, b.Id())
			return
		}

		// commit operation to avoid creating multiple issues with multiple pushes
		if err := b.CommitAsNeeded(); err != nil {
			err := errors.Wrap(err, "bug commit")
			out <- core.NewExportError(err, b.Id())
			return
		}

		// cache bug github ID and URL
		bugGithubID = id
		bugGithubURL = url
	}

	// cache operation github id
	ge.cachedOperationIDs[createOp.Id()] = bugGithubID

	for _, op := range snapshot.Operations[1:] {
		// ignore SetMetadata operations
		if _, ok := op.(dag.OperationDoesntChangeSnapshot); ok {
			continue
		}

		// ignore operations already existing in github (due to import or export)
		// cache the ID of already exported or imported issues and events from Github
		if id, ok := op.GetMetadata(metaKeyGithubId); ok {
			ge.cachedOperationIDs[op.Id()] = id
			continue
		}

		opAuthor := op.Author()
		client, err := ge.getClientForIdentity(opAuthor.Id())
		if err != nil {
			continue
		}

		var id, url string
		switch op := op.(type) {
		case *bug.AddCommentOperation:
			// send operation to github
			id, url, err = ge.addCommentGithubIssue(ctx, client, bugGithubID, op.Message)
			if err != nil {
				err := errors.Wrap(err, "adding comment")
				out <- core.NewExportError(err, b.Id())
				return
			}

			out <- core.NewExportComment(b.Id())

			// cache comment id
			ge.cachedOperationIDs[op.Id()] = id

		case *bug.EditCommentOperation:
			// Since github doesn't consider the issue body as a comment
			if op.Target == createOp.Id() {

				// case bug creation operation: we need to edit the Github issue
				if err := ge.updateGithubIssueBody(ctx, client, bugGithubID, op.Message); err != nil {
					err := errors.Wrap(err, "editing issue")
					out <- core.NewExportError(err, b.Id())
					return
				}

				out <- core.NewExportCommentEdition(b.Id())

				id = bugGithubID
				url = bugGithubURL

			} else {

				// case comment edition operation: we need to edit the Github comment
				commentID, ok := ge.cachedOperationIDs[op.Target]
				if !ok {
					panic("unexpected error: comment id not found")
				}

				eid, eurl, err := ge.editCommentGithubIssue(ctx, client, commentID, op.Message)
				if err != nil {
					err := errors.Wrap(err, "editing comment")
					out <- core.NewExportError(err, b.Id())
					return
				}

				out <- core.NewExportCommentEdition(b.Id())

				// use comment id/url instead of issue id/url
				id = eid
				url = eurl
			}

		case *bug.SetStatusOperation:
			if err := ge.updateGithubIssueStatus(ctx, client, bugGithubID, op.Status); err != nil {
				err := errors.Wrap(err, "editing status")
				out <- core.NewExportError(err, b.Id())
				return
			}

			out <- core.NewExportStatusChange(b.Id())

			id = bugGithubID
			url = bugGithubURL

		case *bug.SetTitleOperation:
			if err := ge.updateGithubIssueTitle(ctx, client, bugGithubID, op.Title); err != nil {
				err := errors.Wrap(err, "editing title")
				out <- core.NewExportError(err, b.Id())
				return
			}

			out <- core.NewExportTitleEdition(b.Id())

			id = bugGithubID
			url = bugGithubURL

		case *bug.LabelChangeOperation:
			if err := ge.updateGithubIssueLabels(ctx, client, bugGithubID, op.Added, op.Removed); err != nil {
				err := errors.Wrap(err, "updating labels")
				out <- core.NewExportError(err, b.Id())
				return
			}

			out <- core.NewExportLabelChange(b.Id())

			id = bugGithubID
			url = bugGithubURL

		default:
			panic("unhandled operation type case")
		}

		// mark operation as exported
		if err := markOperationAsExported(b, op.Id(), id, url); err != nil {
			err := errors.Wrap(err, "marking operation as exported")
			out <- core.NewExportError(err, b.Id())
			return
		}

		// commit at each operation export to avoid exporting same events multiple times
		if err := b.CommitAsNeeded(); err != nil {
			err := errors.Wrap(err, "bug commit")
			out <- core.NewExportError(err, b.Id())
			return
		}

		bugUpdated = true
	}

	if !bugUpdated {
		out <- core.NewExportNothing(b.Id(), "nothing has been exported")
	}
}

// getRepositoryNodeID request github api v3 to get repository node id
func getRepositoryNodeID(ctx context.Context, token *auth.Token, owner, project string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s", githubV3Url, owner, project)
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// need the token for private repositories
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token.Value))

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error %v retrieving repository node id", resp.StatusCode)
	}

	aux := struct {
		NodeID string `json:"node_id"`
	}{}

	data, _ := io.ReadAll(resp.Body)
	err = resp.Body.Close()
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return "", err
	}

	return aux.NodeID, nil
}

func markOperationAsExported(b *cache.BugCache, target entity.Id, githubID, githubURL string) error {
	_, err := b.SetMetadata(
		target,
		map[string]string{
			metaKeyGithubId:  githubID,
			metaKeyGithubUrl: githubURL,
		},
	)

	return err
}

func (ge *githubExporter) cacheGithubLabels(ctx context.Context, gc *rateLimitHandlerClient) error {
	variables := map[string]interface{}{
		"owner": githubv4.String(ge.conf[confKeyOwner]),
		"name":  githubv4.String(ge.conf[confKeyProject]),
		"first": githubv4.Int(10),
		"after": (*githubv4.String)(nil),
	}

	q := labelsQuery{}

	hasNextPage := true
	for hasNextPage {
		if err := gc.queryExport(ctx, &q, variables, ge.out); err != nil {
			return err
		}

		for _, label := range q.Repository.Labels.Nodes {
			ge.cachedLabels[label.Name] = label.ID
		}

		hasNextPage = q.Repository.Labels.PageInfo.HasNextPage
		variables["after"] = q.Repository.Labels.PageInfo.EndCursor
	}

	return nil
}

func (ge *githubExporter) getLabelID(label string) (string, error) {
	label = strings.ToLower(label)
	for cachedLabel, ID := range ge.cachedLabels {
		if label == strings.ToLower(cachedLabel) {
			return ID, nil
		}
	}

	return "", fmt.Errorf("didn't find label id in cache")
}

// create a new label and return it github id
// NOTE: since createLabel mutation is still in preview mode we use github api v3 to create labels
// see https://developer.github.com/v4/mutation/createlabel/ and https://developer.github.com/v4/previews/#labels-preview
func (ge *githubExporter) createGithubLabel(ctx context.Context, label, color string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/labels", githubV3Url, ge.conf[confKeyOwner], ge.conf[confKeyProject])
	client := &http.Client{}

	params := struct {
		Name        string `json:"name"`
		Color       string `json:"color"`
		Description string `json:"description"`
	}{
		Name:  label,
		Color: color,
	}

	data, err := json.Marshal(params)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	req = req.WithContext(ctx)

	// need the token for private repositories
	req.Header.Set("Authorization", fmt.Sprintf("token %s", ge.defaultToken.Value))

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("error creating label: response status %v", resp.StatusCode)
	}

	aux := struct {
		ID     int    `json:"id"`
		NodeID string `json:"node_id"`
		Color  string `json:"color"`
	}{}

	data, _ = io.ReadAll(resp.Body)
	defer resp.Body.Close()

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return "", err
	}

	return aux.NodeID, nil
}

/**
// create github label using api v4
func (ge *githubExporter) createGithubLabelV4(gc *githubv4.Client, label, labelColor string) (string, error) {
	m := createLabelMutation{}
	input := createLabelInput{
		RepositoryID: ge.repositoryID,
		Name:         githubv4.String(label),
		Color:        githubv4.String(labelColor),
	}

	ctx := context.Background()

	if err := gc.mutate(ctx, &m, input, nil); err != nil {
		return "", err
	}

	return m.CreateLabel.Label.ID, nil
}
*/

func (ge *githubExporter) getOrCreateGithubLabelID(ctx context.Context, gc *rateLimitHandlerClient, repositoryID string, label bug.Label) (string, error) {
	// try to get label id from cache
	labelID, err := ge.getLabelID(string(label))
	if err == nil {
		return labelID, nil
	}

	// RGBA to hex color
	rgba := label.Color().RGBA()
	hexColor := fmt.Sprintf("%.2x%.2x%.2x", rgba.R, rgba.G, rgba.B)

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	labelID, err = ge.createGithubLabel(ctx, string(label), hexColor)
	if err != nil {
		return "", err
	}

	return labelID, nil
}

func (ge *githubExporter) getLabelsIDs(ctx context.Context, gc *rateLimitHandlerClient, repositoryID string, labels []bug.Label) ([]githubv4.ID, error) {
	ids := make([]githubv4.ID, 0, len(labels))
	var err error

	// check labels ids
	for _, label := range labels {
		id, ok := ge.cachedLabels[string(label)]
		if !ok {
			// try to query label id
			id, err = ge.getOrCreateGithubLabelID(ctx, gc, repositoryID, label)
			if err != nil {
				return nil, errors.Wrap(err, "get or create github label")
			}

			// cache label id
			ge.cachedLabels[string(label)] = id
		}

		ids = append(ids, githubv4.ID(id))
	}

	return ids, nil
}

// create a github issue and return it ID
func (ge *githubExporter) createGithubIssue(ctx context.Context, gc *rateLimitHandlerClient, repositoryID, title, body string) (string, string, error) {
	m := &createIssueMutation{}
	input := githubv4.CreateIssueInput{
		RepositoryID: repositoryID,
		Title:        githubv4.String(title),
		Body:         (*githubv4.String)(&body),
	}

	if err := gc.mutate(ctx, m, input, nil, ge.out); err != nil {
		return "", "", err
	}

	issue := m.CreateIssue.Issue
	return issue.ID, issue.URL, nil
}

// add a comment to an issue and return its ID
func (ge *githubExporter) addCommentGithubIssue(ctx context.Context, gc *rateLimitHandlerClient, subjectID string, body string) (string, string, error) {
	m := &addCommentToIssueMutation{}
	input := githubv4.AddCommentInput{
		SubjectID: subjectID,
		Body:      githubv4.String(body),
	}

	if err := gc.mutate(ctx, m, input, nil, ge.out); err != nil {
		return "", "", err
	}

	node := m.AddComment.CommentEdge.Node
	return node.ID, node.URL, nil
}

func (ge *githubExporter) editCommentGithubIssue(ctx context.Context, gc *rateLimitHandlerClient, commentID, body string) (string, string, error) {
	m := &updateIssueCommentMutation{}
	input := githubv4.UpdateIssueCommentInput{
		ID:   commentID,
		Body: githubv4.String(body),
	}

	if err := gc.mutate(ctx, m, input, nil, ge.out); err != nil {
		return "", "", err
	}

	return commentID, m.UpdateIssueComment.IssueComment.URL, nil
}

func (ge *githubExporter) updateGithubIssueStatus(ctx context.Context, gc *rateLimitHandlerClient, id string, status common.Status) error {
	m := &updateIssueMutation{}

	// set state
	var state githubv4.IssueState

	switch status {
	case common.OpenStatus:
		state = githubv4.IssueStateOpen
	case common.ClosedStatus:
		state = githubv4.IssueStateClosed
	default:
		panic("unknown bug state")
	}

	input := githubv4.UpdateIssueInput{
		ID:    id,
		State: &state,
	}

	if err := gc.mutate(ctx, m, input, nil, ge.out); err != nil {
		return err
	}

	return nil
}

func (ge *githubExporter) updateGithubIssueBody(ctx context.Context, gc *rateLimitHandlerClient, id string, body string) error {
	m := &updateIssueMutation{}
	input := githubv4.UpdateIssueInput{
		ID:   id,
		Body: (*githubv4.String)(&body),
	}

	if err := gc.mutate(ctx, m, input, nil, ge.out); err != nil {
		return err
	}

	return nil
}

func (ge *githubExporter) updateGithubIssueTitle(ctx context.Context, gc *rateLimitHandlerClient, id, title string) error {
	m := &updateIssueMutation{}
	input := githubv4.UpdateIssueInput{
		ID:    id,
		Title: (*githubv4.String)(&title),
	}

	if err := gc.mutate(ctx, m, input, nil, ge.out); err != nil {
		return err
	}

	return nil
}

// update github issue labels
func (ge *githubExporter) updateGithubIssueLabels(ctx context.Context, gc *rateLimitHandlerClient, labelableID string, added, removed []bug.Label) error {

	wg, ctx := errgroup.WithContext(ctx)
	if len(added) > 0 {
		wg.Go(func() error {
			addedIDs, err := ge.getLabelsIDs(ctx, gc, labelableID, added)
			if err != nil {
				return err
			}

			m := &addLabelsToLabelableMutation{}
			inputAdd := githubv4.AddLabelsToLabelableInput{
				LabelableID: labelableID,
				LabelIDs:    addedIDs,
			}

			// add labels
			if err := gc.mutate(ctx, m, inputAdd, nil, ge.out); err != nil {
				return err
			}
			return nil
		})
	}

	if len(removed) > 0 {
		wg.Go(func() error {
			removedIDs, err := ge.getLabelsIDs(ctx, gc, labelableID, removed)
			if err != nil {
				return err
			}

			m2 := &removeLabelsFromLabelableMutation{}
			inputRemove := githubv4.RemoveLabelsFromLabelableInput{
				LabelableID: labelableID,
				LabelIDs:    removedIDs,
			}

			// remove label labels
			if err := gc.mutate(ctx, m2, inputRemove, nil, ge.out); err != nil {
				return err
			}
			return nil
		})
	}

	return wg.Wait()
}
