package gitlab

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/xanzy/go-gitlab"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/util/text"
)

// gitlabImporter implement the Importer interface
type gitlabImporter struct {
	conf core.Configuration

	// default client
	client *gitlab.Client

	// send only channel
	out chan<- core.ImportResult
}

func (gi *gitlabImporter) Init(_ context.Context, repo *cache.RepoCache, conf core.Configuration) error {
	gi.conf = conf

	creds, err := auth.List(repo,
		auth.WithTarget(target),
		auth.WithKind(auth.KindToken),
		auth.WithMeta(auth.MetaKeyBaseURL, conf[confKeyGitlabBaseUrl]),
		auth.WithMeta(auth.MetaKeyLogin, conf[confKeyDefaultLogin]),
	)
	if err != nil {
		return err
	}

	if len(creds) == 0 {
		return ErrMissingIdentityToken
	}

	gi.client, err = buildClient(conf[confKeyGitlabBaseUrl], creds[0].(*auth.Token))
	if err != nil {
		return err
	}

	return nil
}

// ImportAll iterate over all the configured repository issues (notes) and ensure the creation
// of the missing issues / comments / label events / title changes ...
func (gi *gitlabImporter) ImportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ImportResult, error) {
	out := make(chan core.ImportResult)
	gi.out = out

	go func() {
		defer close(out)

		for issue := range Issues(ctx, gi.client, gi.conf[confKeyProjectID], since) {

			b, err := gi.ensureIssue(repo, issue)
			if err != nil {
				err := fmt.Errorf("issue creation: %v", err)
				out <- core.NewImportError(err, "")
				return
			}

			issueEvents := SortedEvents(
				Notes(ctx, gi.client, issue),
				LabelEvents(ctx, gi.client, issue),
				StateEvents(ctx, gi.client, issue),
			)

			for e := range issueEvents {
				if e, ok := e.(ErrorEvent); ok {
					out <- core.NewImportError(e.Err, "")
					continue
				}
				if err := gi.ensureIssueEvent(repo, b, issue, e); err != nil {
					err := fmt.Errorf("issue event creation: %v", err)
					out <- core.NewImportError(err, entity.Id(e.ID()))
				}
			}

			if !b.NeedCommit() {
				out <- core.NewImportNothing(b.Id(), "no imported operation")
			} else if err := b.Commit(); err != nil {
				// commit bug state
				err := fmt.Errorf("bug commit: %v", err)
				out <- core.NewImportError(err, "")
				return
			}
		}
	}()

	return out, nil
}

func (gi *gitlabImporter) ensureIssue(repo *cache.RepoCache, issue *gitlab.Issue) (*cache.BugCache, error) {
	// ensure issue author
	author, err := gi.ensurePerson(repo, issue.Author.ID)
	if err != nil {
		return nil, err
	}

	// resolve bug
	b, err := repo.Bugs().ResolveMatcher(func(excerpt *cache.BugExcerpt) bool {
		return excerpt.CreateMetadata[core.MetaKeyOrigin] == target &&
			excerpt.CreateMetadata[metaKeyGitlabId] == fmt.Sprintf("%d", issue.IID) &&
			excerpt.CreateMetadata[metaKeyGitlabBaseUrl] == gi.conf[confKeyGitlabBaseUrl] &&
			excerpt.CreateMetadata[metaKeyGitlabProject] == gi.conf[confKeyProjectID]
	})
	if err == nil {
		return b, nil
	}
	if !entity.IsErrNotFound(err) {
		return nil, err
	}

	// if bug was never imported, create bug
	b, _, err = repo.Bugs().NewRaw(
		author,
		issue.CreatedAt.Unix(),
		text.CleanupOneLine(issue.Title),
		text.Cleanup(issue.Description),
		nil,
		map[string]string{
			core.MetaKeyOrigin:   target,
			metaKeyGitlabId:      fmt.Sprintf("%d", issue.IID),
			metaKeyGitlabUrl:     issue.WebURL,
			metaKeyGitlabProject: gi.conf[confKeyProjectID],
			metaKeyGitlabBaseUrl: gi.conf[confKeyGitlabBaseUrl],
		},
	)

	if err != nil {
		return nil, err
	}

	// importing a new bug
	gi.out <- core.NewImportBug(b.Id())

	return b, nil
}

func (gi *gitlabImporter) ensureIssueEvent(repo *cache.RepoCache, b *cache.BugCache, issue *gitlab.Issue, event Event) error {
	id, errResolve := b.ResolveOperationWithMetadata(metaKeyGitlabId, event.ID())
	if errResolve != nil && errResolve != cache.ErrNoMatchingOp {
		return errResolve
	}

	// ensure issue author
	author, err := gi.ensurePerson(repo, event.UserID())
	if err != nil {
		return err
	}

	switch event.Kind() {
	case EventClosed:
		if errResolve == nil {
			return nil
		}

		op, err := b.CloseRaw(
			author,
			event.CreatedAt().Unix(),
			map[string]string{
				metaKeyGitlabId: event.ID(),
			},
		)

		if err != nil {
			return err
		}

		gi.out <- core.NewImportStatusChange(b.Id(), op.Id())

	case EventReopened:
		if errResolve == nil {
			return nil
		}

		op, err := b.OpenRaw(
			author,
			event.CreatedAt().Unix(),
			map[string]string{
				metaKeyGitlabId: event.ID(),
			},
		)
		if err != nil {
			return err
		}

		gi.out <- core.NewImportStatusChange(b.Id(), op.Id())

	case EventDescriptionChanged:
		firstComment := b.Compile().Comments[0]
		// since gitlab doesn't provide the issue history
		// we should check for "changed the description" notes and compare issue texts
		// TODO: Check only one time and ignore next 'description change' within one issue
		cleanedDesc := text.Cleanup(issue.Description)
		if errResolve == cache.ErrNoMatchingOp && cleanedDesc != firstComment.Message {
			// comment edition
			op, err := b.EditCommentRaw(
				author,
				event.(NoteEvent).UpdatedAt.Unix(),
				firstComment.CombinedId(),
				cleanedDesc,
				map[string]string{
					metaKeyGitlabId: event.ID(),
				},
			)
			if err != nil {
				return err
			}

			gi.out <- core.NewImportTitleEdition(b.Id(), op.Id())
		}

	case EventComment:
		cleanText := text.Cleanup(event.(NoteEvent).Body)

		// if we didn't import the comment
		if errResolve == cache.ErrNoMatchingOp {

			// add comment operation
			commentId, _, err := b.AddCommentRaw(
				author,
				event.CreatedAt().Unix(),
				cleanText,
				nil,
				map[string]string{
					metaKeyGitlabId: event.ID(),
				},
			)
			if err != nil {
				return err
			}
			gi.out <- core.NewImportComment(b.Id(), commentId)
			return nil
		}

		// if comment was already exported

		// search for last comment update
		comment, err := b.Compile().SearchCommentByOpId(id)
		if err != nil {
			return err
		}

		// compare local bug comment with the new event body
		if comment.Message != cleanText {
			// comment edition
			_, err := b.EditCommentRaw(
				author,
				event.(NoteEvent).UpdatedAt.Unix(),
				comment.CombinedId(),
				cleanText,
				nil,
			)

			if err != nil {
				return err
			}
			gi.out <- core.NewImportCommentEdition(b.Id(), comment.CombinedId())
		}

		return nil

	case EventTitleChanged:
		// title change events are given new notes
		if errResolve == nil {
			return nil
		}

		op, err := b.SetTitleRaw(
			author,
			event.CreatedAt().Unix(),
			event.(NoteEvent).Title(),
			map[string]string{
				metaKeyGitlabId: event.ID(),
			},
		)
		if err != nil {
			return err
		}

		gi.out <- core.NewImportTitleEdition(b.Id(), op.Id())

	case EventAddLabel:
		_, err = b.ForceChangeLabelsRaw(
			author,
			event.CreatedAt().Unix(),
			[]string{event.(LabelEvent).Label.Name},
			nil,
			map[string]string{
				metaKeyGitlabId: event.ID(),
			},
		)
		return err

	case EventRemoveLabel:
		_, err = b.ForceChangeLabelsRaw(
			author,
			event.CreatedAt().Unix(),
			nil,
			[]string{event.(LabelEvent).Label.Name},
			map[string]string{
				metaKeyGitlabId: event.ID(),
			},
		)
		return err

	case EventAssigned,
		EventUnassigned,
		EventChangedMilestone,
		EventRemovedMilestone,
		EventChangedDuedate,
		EventRemovedDuedate,
		EventLocked,
		EventUnlocked,
		EventMentionedInIssue,
		EventMentionedInMergeRequest,
		EventMentionedInCommit:

		return nil

	default:
		return fmt.Errorf("unexpected event")
	}

	return nil
}

func (gi *gitlabImporter) ensurePerson(repo *cache.RepoCache, id int) (*cache.IdentityCache, error) {
	// Look first in the cache
	i, err := repo.Identities().ResolveIdentityImmutableMetadata(metaKeyGitlabId, strconv.Itoa(id))
	if err == nil {
		return i, nil
	}
	if entity.IsErrMultipleMatch(err) {
		return nil, err
	}

	user, _, err := gi.client.Users.GetUser(id, gitlab.GetUsersOptions{})
	if err != nil {
		return nil, err
	}

	i, err = repo.Identities().NewRaw(
		user.Name,
		user.PublicEmail,
		user.Username,
		user.AvatarURL,
		nil,
		map[string]string{
			// because Gitlab
			metaKeyGitlabId:    strconv.Itoa(id),
			metaKeyGitlabLogin: user.Username,
		},
	)
	if err != nil {
		return nil, err
	}

	gi.out <- core.NewImportIdentity(i.Id())
	return i, nil
}
