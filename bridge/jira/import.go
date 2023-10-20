package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/entities/common"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/util/text"
)

const (
	defaultPageSize = 10
)

// jiraImporter implement the Importer interface
type jiraImporter struct {
	conf core.Configuration

	client *Client

	// send only channel
	out chan<- core.ImportResult
}

// Init .
func (ji *jiraImporter) Init(ctx context.Context, repo *cache.RepoCache, conf core.Configuration) error {
	ji.conf = conf

	var cred auth.Credential

	// Prioritize LoginPassword credentials to avoid a prompt
	creds, err := auth.List(repo,
		auth.WithTarget(target),
		auth.WithKind(auth.KindLoginPassword),
		auth.WithMeta(auth.MetaKeyBaseURL, conf[confKeyBaseUrl]),
		auth.WithMeta(auth.MetaKeyLogin, conf[confKeyDefaultLogin]),
	)
	if err != nil {
		return err
	}
	if len(creds) > 0 {
		cred = creds[0]
		goto end
	}

	creds, err = auth.List(repo,
		auth.WithTarget(target),
		auth.WithKind(auth.KindLogin),
		auth.WithMeta(auth.MetaKeyBaseURL, conf[confKeyBaseUrl]),
		auth.WithMeta(auth.MetaKeyLogin, conf[confKeyDefaultLogin]),
	)
	if err != nil {
		return err
	}
	if len(creds) > 0 {
		cred = creds[0]
	}

end:
	if cred == nil {
		return fmt.Errorf("no credential for this bridge")
	}

	// TODO(josh)[da52062]: Validate token and if it is expired then prompt for
	// credentials and generate a new one
	ji.client, err = buildClient(ctx, conf[confKeyBaseUrl], conf[confKeyCredentialType], cred)
	return err
}

// ImportAll iterate over all the configured repository issues and ensure the
// creation of the missing issues / timeline items / edits / label events ...
func (ji *jiraImporter) ImportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ImportResult, error) {
	sinceStr := since.Format("2006-01-02 15:04")
	project := ji.conf[confKeyProject]

	out := make(chan core.ImportResult)
	ji.out = out

	go func() {
		defer close(ji.out)

		message, err := ji.client.Search(
			fmt.Sprintf("project=%s AND updatedDate>\"%s\"", project, sinceStr), 0, 0)
		if err != nil {
			out <- core.NewImportError(err, "")
			return
		}

		fmt.Printf("So far so good. Have %d issues to import\n", message.Total)

		jql := fmt.Sprintf("project=%s AND updatedDate>\"%s\"", project, sinceStr)
		var searchIter *SearchIterator
		for searchIter =
			ji.client.IterSearch(jql, defaultPageSize); searchIter.HasNext(); {
			issue := searchIter.Next()
			b, err := ji.ensureIssue(repo, *issue)
			if err != nil {
				err := fmt.Errorf("issue creation: %v", err)
				out <- core.NewImportError(err, "")
				return
			}

			var commentIter *CommentIterator
			for commentIter =
				ji.client.IterComments(issue.ID, defaultPageSize); commentIter.HasNext(); {
				comment := commentIter.Next()
				err := ji.ensureComment(repo, b, *comment)
				if err != nil {
					out <- core.NewImportError(err, "")
				}
			}
			if commentIter.HasError() {
				out <- core.NewImportError(commentIter.Err, "")
			}

			snapshot := b.Compile()
			opIdx := 0

			var changelogIter *ChangeLogIterator
			for changelogIter =
				ji.client.IterChangeLog(issue.ID, defaultPageSize); changelogIter.HasNext(); {
				changelogEntry := changelogIter.Next()

				// Advance the operation iterator up to the first operation which has
				// an export date not before the changelog entry date. If the changelog
				// entry was created in response to an exported operation, then this
				// will be that operation.
				var exportTime time.Time
				for ; opIdx < len(snapshot.Operations); opIdx++ {
					exportTimeStr, hasTime := snapshot.Operations[opIdx].GetMetadata(
						metaKeyJiraExportTime)
					if !hasTime {
						continue
					}
					exportTime, err = http.ParseTime(exportTimeStr)
					if err != nil {
						continue
					}
					if !exportTime.Before(changelogEntry.Created.Time) {
						break
					}
				}
				if opIdx < len(snapshot.Operations) {
					err = ji.ensureChange(repo, b, *changelogEntry, snapshot.Operations[opIdx])
				} else {
					err = ji.ensureChange(repo, b, *changelogEntry, nil)
				}
				if err != nil {
					out <- core.NewImportError(err, "")
				}

			}
			if changelogIter.HasError() {
				out <- core.NewImportError(changelogIter.Err, "")
			}

			if !b.NeedCommit() {
				out <- core.NewImportNothing(b.Id(), "no imported operation")
			} else if err := b.Commit(); err != nil {
				err = fmt.Errorf("bug commit: %v", err)
				out <- core.NewImportError(err, "")
				return
			}
		}
		if searchIter.HasError() {
			out <- core.NewImportError(searchIter.Err, "")
		}
	}()

	return out, nil
}

// Create a bug.Person from a JIRA user
func (ji *jiraImporter) ensurePerson(repo *cache.RepoCache, user User) (*cache.IdentityCache, error) {
	// Look first in the cache
	i, err := repo.Identities().ResolveIdentityImmutableMetadata(
		metaKeyJiraUser, string(user.Key))
	if err == nil {
		return i, nil
	}
	if _, ok := err.(entity.ErrMultipleMatch); ok {
		return nil, err
	}

	i, err = repo.Identities().NewRaw(
		user.DisplayName,
		user.EmailAddress,
		user.Key,
		"",
		nil,
		map[string]string{
			metaKeyJiraUser: user.Key,
		},
	)

	if err != nil {
		return nil, err
	}

	ji.out <- core.NewImportIdentity(i.Id())
	return i, nil
}

// Create a bug.Bug based from a JIRA issue
func (ji *jiraImporter) ensureIssue(repo *cache.RepoCache, issue Issue) (*cache.BugCache, error) {
	author, err := ji.ensurePerson(repo, issue.Fields.Creator)
	if err != nil {
		return nil, err
	}

	b, err := repo.Bugs().ResolveMatcher(func(excerpt *cache.BugExcerpt) bool {
		if _, ok := excerpt.CreateMetadata[metaKeyJiraBaseUrl]; ok &&
			excerpt.CreateMetadata[metaKeyJiraBaseUrl] != ji.conf[confKeyBaseUrl] {
			return false
		}

		return excerpt.CreateMetadata[core.MetaKeyOrigin] == target &&
			excerpt.CreateMetadata[metaKeyJiraId] == issue.ID &&
			excerpt.CreateMetadata[metaKeyJiraProject] == ji.conf[confKeyProject]
	})
	if err != nil && !entity.IsErrNotFound(err) {
		return nil, err
	}

	if entity.IsErrNotFound(err) {
		b, _, err = repo.Bugs().NewRaw(
			author,
			issue.Fields.Created.Unix(),
			text.CleanupOneLine(issue.Fields.Summary),
			text.Cleanup(issue.Fields.Description),
			nil,
			map[string]string{
				core.MetaKeyOrigin: target,
				metaKeyJiraId:      issue.ID,
				metaKeyJiraKey:     issue.Key,
				metaKeyJiraProject: ji.conf[confKeyProject],
				metaKeyJiraBaseUrl: ji.conf[confKeyBaseUrl],
			})
		if err != nil {
			return nil, err
		}

		ji.out <- core.NewImportBug(b.Id())
	}

	return b, nil
}

// Return a unique string derived from a unique jira id and a timestamp
func getTimeDerivedID(jiraID string, timestamp Time) string {
	return fmt.Sprintf("%s-%d", jiraID, timestamp.Unix())
}

// Create a bug.Comment from a JIRA comment
func (ji *jiraImporter) ensureComment(repo *cache.RepoCache, b *cache.BugCache, item Comment) error {
	// ensure person
	author, err := ji.ensurePerson(repo, item.Author)
	if err != nil {
		return err
	}

	targetOpID, err := b.ResolveOperationWithMetadata(metaKeyJiraId, item.ID)
	if err != nil && err != cache.ErrNoMatchingOp {
		return err
	}

	// If the comment is a new comment then create it
	if targetOpID == "" && err == cache.ErrNoMatchingOp {
		var cleanText string
		if item.Updated != item.Created {
			// We don't know the original text... we only have the updated text.
			cleanText = ""
		} else {
			cleanText = text.Cleanup(item.Body)
		}

		// add comment operation
		commentId, op, err := b.AddCommentRaw(
			author,
			item.Created.Unix(),
			cleanText,
			nil,
			map[string]string{
				metaKeyJiraId: item.ID,
			},
		)
		if err != nil {
			return err
		}

		ji.out <- core.NewImportComment(b.Id(), commentId)
		targetOpID = op.Id()
	}

	// If there are no updates to this comment, then we are done
	if item.Updated == item.Created {
		return nil
	}

	// If there has been an update to this comment, we try to find it in the
	// database. We need a unique id so we'll concat the issue id with the update
	// timestamp. Note that this must be consistent with the exporter during
	// export of an EditCommentOperation
	derivedID := getTimeDerivedID(item.ID, item.Updated)
	_, err = b.ResolveOperationWithMetadata(metaKeyJiraId, derivedID)
	if err == nil {
		// Already imported this edition
		return nil
	}

	if err != cache.ErrNoMatchingOp {
		return err
	}

	// ensure editor identity
	editor, err := ji.ensurePerson(repo, item.UpdateAuthor)
	if err != nil {
		return err
	}

	commentId := entity.CombineIds(b.Id(), targetOpID)

	// comment edition
	_, err = b.EditCommentRaw(
		editor,
		item.Updated.Unix(),
		commentId,
		text.Cleanup(item.Body),
		map[string]string{
			metaKeyJiraId: derivedID,
		},
	)

	if err != nil {
		return err
	}

	ji.out <- core.NewImportCommentEdition(b.Id(), commentId)

	return nil
}

// Return a unique string derived from a unique jira id and an index into the
// data referred to by that jira id.
func getIndexDerivedID(jiraID string, idx int) string {
	return fmt.Sprintf("%s-%d", jiraID, idx)
}

func labelSetsMatch(jiraSet []string, gitbugSet []bug.Label) bool {
	if len(jiraSet) != len(gitbugSet) {
		return false
	}

	sort.Strings(jiraSet)
	gitbugStrSet := make([]string, len(gitbugSet))
	for idx, label := range gitbugSet {
		gitbugStrSet[idx] = label.String()
	}
	sort.Strings(gitbugStrSet)

	for idx, value := range jiraSet {
		if value != gitbugStrSet[idx] {
			return false
		}
	}

	return true
}

// Create a bug.Operation (or a series of operations) from a JIRA changelog
// entry
func (ji *jiraImporter) ensureChange(repo *cache.RepoCache, b *cache.BugCache, entry ChangeLogEntry, potentialOp dag.Operation) error {

	// If we have an operation which is already mapped to the entire changelog
	// entry then that means this changelog entry was induced by an export
	// operation and we've already done the match, so we skip this one
	_, err := b.ResolveOperationWithMetadata(metaKeyJiraDerivedId, entry.ID)
	if err == nil {
		return nil
	} else if err != cache.ErrNoMatchingOp {
		return err
	}

	// In general, multiple fields may be changed in changelog entry  on
	// JIRA. For example, when an issue is closed both its "status" and its
	// "resolution" are updated within a single changelog entry.
	// I don't thing git-bug has a single operation to modify an arbitrary
	// number of fields in one go, so we break up the single JIRA changelog
	// entry into individual field updates.
	author, err := ji.ensurePerson(repo, entry.Author)
	if err != nil {
		return err
	}

	if len(entry.Items) < 1 {
		return fmt.Errorf("Received changelog entry with no item! (%s)", entry.ID)
	}

	statusMap, err := getStatusMapReverse(ji.conf)
	if err != nil {
		return err
	}

	// NOTE(josh): first do an initial scan and see if any of the changed items
	// matches the current potential operation. If it does, then we know that this
	// entire changelog entry was created in response to that git-bug operation.
	// So we associate the operation with the entire changelog, and not a specific
	// entry.
	for _, item := range entry.Items {
		switch item.Field {
		case "labels":
			fromLabels := removeEmpty(strings.Split(item.FromString, " "))
			toLabels := removeEmpty(strings.Split(item.ToString, " "))
			removedLabels, addedLabels, _ := setSymmetricDifference(fromLabels, toLabels)

			opr, isRightType := potentialOp.(*bug.LabelChangeOperation)
			if isRightType && labelSetsMatch(addedLabels, opr.Added) && labelSetsMatch(removedLabels, opr.Removed) {
				_, err := b.SetMetadata(opr.Id(), map[string]string{
					metaKeyJiraDerivedId: entry.ID,
				})
				if err != nil {
					return err
				}
				return nil
			}

		case "status":
			opr, isRightType := potentialOp.(*bug.SetStatusOperation)
			if isRightType && statusMap[opr.Status.String()] == item.To {
				_, err := b.SetMetadata(opr.Id(), map[string]string{
					metaKeyJiraDerivedId: entry.ID,
				})
				if err != nil {
					return err
				}
				return nil
			}

		case "summary":
			// NOTE(josh): JIRA calls it "summary", which sounds more like the body
			// text, but it's the title
			opr, isRightType := potentialOp.(*bug.SetTitleOperation)
			if isRightType && opr.Title == item.To {
				_, err := b.SetMetadata(opr.Id(), map[string]string{
					metaKeyJiraDerivedId: entry.ID,
				})
				if err != nil {
					return err
				}
				return nil
			}

		case "description":
			// NOTE(josh): JIRA calls it "description", which sounds more like the
			// title but it's actually the body
			opr, isRightType := potentialOp.(*bug.EditCommentOperation)
			if isRightType &&
				opr.Target == b.Compile().Operations[0].Id() &&
				opr.Message == item.ToString {
				_, err := b.SetMetadata(opr.Id(), map[string]string{
					metaKeyJiraDerivedId: entry.ID,
				})
				if err != nil {
					return err
				}
				return nil
			}
		}
	}

	// Since we didn't match the changelog entry to a known export operation,
	// then this is a changelog entry that we should import. We import each
	// changelog entry item as a separate git-bug operation.
	for idx, item := range entry.Items {
		derivedID := getIndexDerivedID(entry.ID, idx)
		_, err := b.ResolveOperationWithMetadata(metaKeyJiraDerivedId, derivedID)
		if err == nil {
			continue
		}
		if err != cache.ErrNoMatchingOp {
			return err
		}

		switch item.Field {
		case "labels":
			fromLabels := removeEmpty(strings.Split(item.FromString, " "))
			toLabels := removeEmpty(strings.Split(item.ToString, " "))
			removedLabels, addedLabels, _ := setSymmetricDifference(fromLabels, toLabels)

			op, err := b.ForceChangeLabelsRaw(
				author,
				entry.Created.Unix(),
				text.CleanupOneLineArray(addedLabels),
				text.CleanupOneLineArray(removedLabels),
				map[string]string{
					metaKeyJiraId:        entry.ID,
					metaKeyJiraDerivedId: derivedID,
				},
			)
			if err != nil {
				return err
			}

			ji.out <- core.NewImportLabelChange(b.Id(), op.Id())

		case "status":
			statusStr, hasMap := statusMap[item.To]
			if hasMap {
				switch statusStr {
				case common.OpenStatus.String():
					op, err := b.OpenRaw(
						author,
						entry.Created.Unix(),
						map[string]string{
							metaKeyJiraId:        entry.ID,
							metaKeyJiraDerivedId: derivedID,
						},
					)
					if err != nil {
						return err
					}
					ji.out <- core.NewImportStatusChange(b.Id(), op.Id())

				case common.ClosedStatus.String():
					op, err := b.CloseRaw(
						author,
						entry.Created.Unix(),
						map[string]string{
							metaKeyJiraId:        entry.ID,
							metaKeyJiraDerivedId: derivedID,
						},
					)
					if err != nil {
						return err
					}
					ji.out <- core.NewImportStatusChange(b.Id(), op.Id())
				}
			} else {
				ji.out <- core.NewImportError(
					fmt.Errorf(
						"No git-bug status mapped for jira status %s (%s)",
						item.ToString, item.To), "")
			}

		case "summary":
			// NOTE(josh): JIRA calls it "summary", which sounds more like the body
			// text, but it's the title
			op, err := b.SetTitleRaw(
				author,
				entry.Created.Unix(),
				text.CleanupOneLine(item.ToString),
				map[string]string{
					metaKeyJiraId:        entry.ID,
					metaKeyJiraDerivedId: derivedID,
				},
			)
			if err != nil {
				return err
			}

			ji.out <- core.NewImportTitleEdition(b.Id(), op.Id())

		case "description":
			// NOTE(josh): JIRA calls it "description", which sounds more like the
			// title but it's actually the body
			commentId, _, err := b.EditCreateCommentRaw(
				author,
				entry.Created.Unix(),
				text.Cleanup(item.ToString),
				map[string]string{
					metaKeyJiraId:        entry.ID,
					metaKeyJiraDerivedId: derivedID,
				},
			)
			if err != nil {
				return err
			}

			ji.out <- core.NewImportCommentEdition(b.Id(), commentId)

		default:
			ji.out <- core.NewImportWarning(
				fmt.Errorf(
					"Unhandled changelog event %s", item.Field), "")
		}

		// Other Examples:
		// "assignee" (jira)
		// "Attachment" (jira)
		// "Epic Link" (custom)
		// "Rank" (custom)
		// "resolution" (jira)
		// "Sprint" (custom)
	}
	return nil
}

func getStatusMap(conf core.Configuration) (map[string]string, error) {
	mapStr, hasConf := conf[confKeyIDMap]
	if !hasConf {
		return map[string]string{
			common.OpenStatus.String():   "1",
			common.ClosedStatus.String(): "6",
		}, nil
	}

	statusMap := make(map[string]string)
	err := json.Unmarshal([]byte(mapStr), &statusMap)
	return statusMap, err
}

func getStatusMapReverse(conf core.Configuration) (map[string]string, error) {
	fwdMap, err := getStatusMap(conf)
	if err != nil {
		return fwdMap, err
	}

	outMap := map[string]string{}
	for key, val := range fwdMap {
		outMap[val] = key
	}

	mapStr, hasConf := conf[confKeyIDRevMap]
	if !hasConf {
		return outMap, nil
	}

	revMap := make(map[string]string)
	err = json.Unmarshal([]byte(mapStr), &revMap)
	for key, val := range revMap {
		outMap[key] = val
	}

	return outMap, err
}

func removeEmpty(values []string) []string {
	output := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			output = append(output, value)
		}
	}
	return output
}
