package github

import (
	"context"
	"fmt"
	"time"

	"github.com/shurcooL/githubv4"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/util/text"
)

const EmptyTitlePlaceholder = "<empty string>"

// githubImporter implement the Importer interface
type githubImporter struct {
	conf core.Configuration

	// default client
	client *client

	// mediator to access the Github API
	mediator *importMediator

	// send only channel
	out chan<- core.ImportResult
}

func (gi *githubImporter) Init(_ context.Context, repo *cache.RepoCache, conf core.Configuration) error {
	gi.conf = conf
	creds, err := auth.List(repo,
		auth.WithTarget(target),
		auth.WithKind(auth.KindToken),
		auth.WithMeta(auth.MetaKeyLogin, conf[confKeyDefaultLogin]),
	)
	if err != nil {
		return err
	}
	if len(creds) <= 0 {
		return ErrMissingIdentityToken
	}
	gi.client = buildClient(creds[0].(*auth.Token))

	return nil
}

// ImportAll iterate over all the configured repository issues and ensure the creation of the
// missing issues / timeline items / edits / label events ...
func (gi *githubImporter) ImportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ImportResult, error) {
	gi.mediator = NewImportMediator(ctx, gi.client, gi.conf[confKeyOwner], gi.conf[confKeyProject], since)
	out := make(chan core.ImportResult)
	gi.out = out

	go func() {
		defer close(gi.out)
		var currBug *cache.BugCache
		var currEvent ImportEvent
		var nextEvent ImportEvent
		var err error
		for {
			// An IssueEvent contains the issue in its most recent state. If an issue
			// has at least one issue edit, then the history of the issue edits is
			// represented by IssueEditEvents. That is, the unedited (original) issue
			// might be saved only in the IssueEditEvent following the IssueEvent.
			// Since we replicate the edit history we need to either use the IssueEvent
			// (if there are no edits) or the IssueEvent together with its first
			// IssueEditEvent (if there are edits).
			// Exactly the same is true for comments and comment edits.
			// As a consequence we need to look at the current event and one look ahead
			// event.
			currEvent = nextEvent
			if currEvent == nil {
				currEvent = gi.getEventHandleMsgs()
			}
			if currEvent == nil {
				break
			}
			nextEvent = gi.getEventHandleMsgs()

			switch event := currEvent.(type) {
			case RateLimitingEvent:
				out <- core.NewImportRateLimiting(event.msg)
			case IssueEvent:
				// first: commit what is being held in currBug
				if err = gi.commit(currBug, out); err != nil {
					out <- core.NewImportError(err, "")
					return
				}
				// second: create new issue
				switch next := nextEvent.(type) {
				case IssueEditEvent:
					// consuming and using next event
					nextEvent = nil
					currBug, err = gi.ensureIssue(ctx, repo, &event.issue, &next.userContentEdit)
				default:
					currBug, err = gi.ensureIssue(ctx, repo, &event.issue, nil)
				}
				if err != nil {
					err := fmt.Errorf("issue creation: %v", err)
					out <- core.NewImportError(err, "")
					return
				}
			case IssueEditEvent:
				err = gi.ensureIssueEdit(ctx, repo, currBug, event.issueId, &event.userContentEdit)
				if err != nil {
					err = fmt.Errorf("issue edit: %v", err)
					out <- core.NewImportError(err, "")
					return
				}
			case TimelineEvent:
				if next, ok := nextEvent.(CommentEditEvent); ok && event.Typename == "IssueComment" {
					// consuming and using next event
					nextEvent = nil
					err = gi.ensureComment(ctx, repo, currBug, &event.timelineItem.IssueComment, &next.userContentEdit)
				} else {
					err = gi.ensureTimelineItem(ctx, repo, currBug, &event.timelineItem)
				}
				if err != nil {
					err = fmt.Errorf("timeline item creation: %v", err)
					out <- core.NewImportError(err, "")
					return
				}
			case CommentEditEvent:
				err = gi.ensureCommentEdit(ctx, repo, currBug, event.commentId, &event.userContentEdit)
				if err != nil {
					err = fmt.Errorf("comment edit: %v", err)
					out <- core.NewImportError(err, "")
					return
				}
			default:
				panic("Unknown event type")
			}
		}
		// commit what is being held in currBug before returning
		if err = gi.commit(currBug, out); err != nil {
			out <- core.NewImportError(err, "")
		}
		if err = gi.mediator.Error(); err != nil {
			gi.out <- core.NewImportError(err, "")
		}
	}()

	return out, nil
}

func (gi *githubImporter) getEventHandleMsgs() ImportEvent {
	for {
		// read event from import mediator
		event := gi.mediator.NextImportEvent()
		// consume (and use) all rate limiting events
		if e, ok := event.(RateLimitingEvent); ok {
			gi.out <- core.NewImportRateLimiting(e.msg)
			continue
		}
		return event
	}
}

func (gi *githubImporter) commit(b *cache.BugCache, out chan<- core.ImportResult) error {
	if b == nil {
		return nil
	}
	if !b.NeedCommit() {
		out <- core.NewImportNothing(b.Id(), "no imported operation")
		return nil
	} else if err := b.Commit(); err != nil {
		// commit bug state
		return fmt.Errorf("bug commit: %v", err)
	}
	return nil
}

func (gi *githubImporter) ensureIssue(ctx context.Context, repo *cache.RepoCache, issue *issue, issueEdit *userContentEdit) (*cache.BugCache, error) {
	author, err := gi.ensurePerson(ctx, repo, issue.Author)
	if err != nil {
		return nil, err
	}

	// resolve bug
	b, err := repo.ResolveBugMatcher(func(excerpt *cache.BugExcerpt) bool {
		return excerpt.CreateMetadata[core.MetaKeyOrigin] == target &&
			excerpt.CreateMetadata[metaKeyGithubId] == parseId(issue.Id)
	})
	if err == nil {
		return b, nil
	}
	if err != bug.ErrBugNotExist {
		return nil, err
	}

	// At Github there exist issues with seemingly empty titles. An example is
	// https://github.com/NixOS/nixpkgs/issues/72730 .
	// The title provided by the GraphQL API actually consists of a space followed by a
	// zero width space (U+200B). This title would cause the NewBugRaw() function to
	// return an error: empty title.
	title := string(issue.Title)
	if title == " \u200b" { // U+200B == zero width space
		title = EmptyTitlePlaceholder
	}

	var textInput string
	if issueEdit != nil {
		// use the first issue edit: it represents the bug creation itself
		textInput = string(*issueEdit.Diff)
	} else {
		// if there are no issue edits then the issue struct holds the bug creation
		textInput = string(issue.Body)
	}

	// create bug
	b, _, err = repo.NewBugRaw(
		author,
		issue.CreatedAt.Unix(),
		text.CleanupOneLine(title), // TODO: this is the *current* title, not the original one
		text.Cleanup(textInput),
		nil,
		map[string]string{
			core.MetaKeyOrigin: target,
			metaKeyGithubId:    parseId(issue.Id),
			metaKeyGithubUrl:   issue.Url.String(),
		})
	if err != nil {
		return nil, err
	}
	// importing a new bug
	gi.out <- core.NewImportBug(b.Id())

	return b, nil
}

func (gi *githubImporter) ensureIssueEdit(ctx context.Context, repo *cache.RepoCache, bug *cache.BugCache, ghIssueId githubv4.ID, edit *userContentEdit) error {
	return gi.ensureCommentEdit(ctx, repo, bug, ghIssueId, edit)
}

func (gi *githubImporter) ensureTimelineItem(ctx context.Context, repo *cache.RepoCache, b *cache.BugCache, item *timelineItem) error {

	switch item.Typename {
	case "IssueComment":
		err := gi.ensureComment(ctx, repo, b, &item.IssueComment, nil)
		if err != nil {
			return fmt.Errorf("timeline comment creation: %v", err)
		}
		return nil

	case "LabeledEvent":
		id := parseId(item.LabeledEvent.Id)
		_, err := b.ResolveOperationWithMetadata(metaKeyGithubId, id)
		if err == nil {
			return nil
		}

		if err != cache.ErrNoMatchingOp {
			return err
		}
		author, err := gi.ensurePerson(ctx, repo, item.LabeledEvent.Actor)
		if err != nil {
			return err
		}
		op, err := b.ForceChangeLabelsRaw(
			author,
			item.LabeledEvent.CreatedAt.Unix(),
			[]string{
				text.CleanupOneLine(string(item.LabeledEvent.Label.Name)),
			},
			nil,
			map[string]string{metaKeyGithubId: id},
		)
		if err != nil {
			return err
		}

		gi.out <- core.NewImportLabelChange(op.Id())
		return nil

	case "UnlabeledEvent":
		id := parseId(item.UnlabeledEvent.Id)
		_, err := b.ResolveOperationWithMetadata(metaKeyGithubId, id)
		if err == nil {
			return nil
		}
		if err != cache.ErrNoMatchingOp {
			return err
		}
		author, err := gi.ensurePerson(ctx, repo, item.UnlabeledEvent.Actor)
		if err != nil {
			return err
		}

		op, err := b.ForceChangeLabelsRaw(
			author,
			item.UnlabeledEvent.CreatedAt.Unix(),
			nil,
			[]string{
				text.CleanupOneLine(string(item.UnlabeledEvent.Label.Name)),
			},
			map[string]string{metaKeyGithubId: id},
		)
		if err != nil {
			return err
		}

		gi.out <- core.NewImportLabelChange(op.Id())
		return nil

	case "ClosedEvent":
		id := parseId(item.ClosedEvent.Id)
		_, err := b.ResolveOperationWithMetadata(metaKeyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		if err == nil {
			return nil
		}
		author, err := gi.ensurePerson(ctx, repo, item.ClosedEvent.Actor)
		if err != nil {
			return err
		}
		op, err := b.CloseRaw(
			author,
			item.ClosedEvent.CreatedAt.Unix(),
			map[string]string{metaKeyGithubId: id},
		)

		if err != nil {
			return err
		}

		gi.out <- core.NewImportStatusChange(op.Id())
		return nil

	case "ReopenedEvent":
		id := parseId(item.ReopenedEvent.Id)
		_, err := b.ResolveOperationWithMetadata(metaKeyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		if err == nil {
			return nil
		}
		author, err := gi.ensurePerson(ctx, repo, item.ReopenedEvent.Actor)
		if err != nil {
			return err
		}
		op, err := b.OpenRaw(
			author,
			item.ReopenedEvent.CreatedAt.Unix(),
			map[string]string{metaKeyGithubId: id},
		)

		if err != nil {
			return err
		}

		gi.out <- core.NewImportStatusChange(op.Id())
		return nil

	case "RenamedTitleEvent":
		id := parseId(item.RenamedTitleEvent.Id)
		_, err := b.ResolveOperationWithMetadata(metaKeyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		if err == nil {
			return nil
		}
		author, err := gi.ensurePerson(ctx, repo, item.RenamedTitleEvent.Actor)
		if err != nil {
			return err
		}

		// At Github there exist issues with seemingly empty titles. An example is
		// https://github.com/NixOS/nixpkgs/issues/72730 .
		// The title provided by the GraphQL API actually consists of a space followed
		// by a zero width space (U+200B). This title would cause the NewBugRaw()
		// function to return an error: empty title.
		title := text.CleanupOneLine(string(item.RenamedTitleEvent.CurrentTitle))
		if title == " \u200b" { // U+200B == zero width space
			title = EmptyTitlePlaceholder
		}

		op, err := b.SetTitleRaw(
			author,
			item.RenamedTitleEvent.CreatedAt.Unix(),
			title,
			map[string]string{metaKeyGithubId: id},
		)
		if err != nil {
			return err
		}

		gi.out <- core.NewImportTitleEdition(op.Id())
		return nil
	}

	return nil
}

func (gi *githubImporter) ensureCommentEdit(ctx context.Context, repo *cache.RepoCache, b *cache.BugCache, ghTargetId githubv4.ID, edit *userContentEdit) error {
	// find comment
	target, err := b.ResolveOperationWithMetadata(metaKeyGithubId, parseId(ghTargetId))
	if err != nil {
		return err
	}
	_, err = b.ResolveOperationWithMetadata(metaKeyGithubId, parseId(edit.Id))
	if err == nil {
		return nil
	}
	if err != cache.ErrNoMatchingOp {
		// real error
		return err
	}

	editor, err := gi.ensurePerson(ctx, repo, edit.Editor)
	if err != nil {
		return err
	}

	if edit.DeletedAt != nil {
		// comment deletion, not supported yet
		return nil
	}

	// comment edition
	op, err := b.EditCommentRaw(
		editor,
		edit.CreatedAt.Unix(),
		target,
		text.Cleanup(string(*edit.Diff)),
		map[string]string{
			metaKeyGithubId: parseId(edit.Id),
		},
	)

	if err != nil {
		return err
	}

	gi.out <- core.NewImportCommentEdition(op.Id())
	return nil
}

func (gi *githubImporter) ensureComment(ctx context.Context, repo *cache.RepoCache, b *cache.BugCache, comment *issueComment, firstEdit *userContentEdit) error {
	author, err := gi.ensurePerson(ctx, repo, comment.Author)
	if err != nil {
		return err
	}

	_, err = b.ResolveOperationWithMetadata(metaKeyGithubId, parseId(comment.Id))
	if err == nil {
		return nil
	}
	if err != cache.ErrNoMatchingOp {
		// real error
		return err
	}

	var textInput string
	if firstEdit != nil {
		// use the first comment edit: it represents the comment creation itself
		textInput = string(*firstEdit.Diff)
	} else {
		// if there are not comment edits, then the comment struct holds the comment creation
		textInput = string(comment.Body)
	}

	// add comment operation
	op, err := b.AddCommentRaw(
		author,
		comment.CreatedAt.Unix(),
		text.Cleanup(textInput),
		nil,
		map[string]string{
			metaKeyGithubId:  parseId(comment.Id),
			metaKeyGithubUrl: comment.Url.String(),
		},
	)
	if err != nil {
		return err
	}

	gi.out <- core.NewImportComment(op.Id())
	return nil
}

// ensurePerson create a bug.Person from the Github data
func (gi *githubImporter) ensurePerson(ctx context.Context, repo *cache.RepoCache, actor *actor) (*cache.IdentityCache, error) {
	// When a user has been deleted, Github return a null actor, while displaying a profile named "ghost"
	// in it's UI. So we need a special case to get it.
	if actor == nil {
		return gi.getGhost(ctx, repo)
	}

	// Look first in the cache
	i, err := repo.ResolveIdentityImmutableMetadata(metaKeyGithubLogin, string(actor.Login))
	if err == nil {
		return i, nil
	}
	if entity.IsErrMultipleMatch(err) {
		return nil, err
	}

	// importing a new identity
	var name string
	var email string

	switch actor.Typename {
	case "User":
		if actor.User.Name != nil {
			name = string(*(actor.User.Name))
		}
		email = string(actor.User.Email)
	case "Organization":
		if actor.Organization.Name != nil {
			name = string(*(actor.Organization.Name))
		}
		if actor.Organization.Email != nil {
			email = string(*(actor.Organization.Email))
		}
	case "Bot":
	}

	// Name is not necessarily set, fallback to login as a name is required in the identity
	if name == "" {
		name = string(actor.Login)
	}

	i, err = repo.NewIdentityRaw(
		name,
		email,
		string(actor.Login),
		string(actor.AvatarUrl),
		nil,
		map[string]string{
			metaKeyGithubLogin: string(actor.Login),
		},
	)

	if err != nil {
		return nil, err
	}

	gi.out <- core.NewImportIdentity(i.Id())
	return i, nil
}

func (gi *githubImporter) getGhost(ctx context.Context, repo *cache.RepoCache) (*cache.IdentityCache, error) {
	loginName := "ghost"
	// Look first in the cache
	i, err := repo.ResolveIdentityImmutableMetadata(metaKeyGithubLogin, loginName)
	if err == nil {
		return i, nil
	}
	if entity.IsErrMultipleMatch(err) {
		return nil, err
	}
	user, err := gi.mediator.User(ctx, loginName)
	if err != nil {
		return nil, err
	}
	userName := ""
	if user.Name != nil {
		userName = string(*user.Name)
	}
	return repo.NewIdentityRaw(
		userName,
		"",
		string(user.Login),
		string(user.AvatarUrl),
		nil,
		map[string]string{
			metaKeyGithubLogin: string(user.Login),
		},
	)
}

// parseId converts the unusable githubv4.ID (an interface{}) into a string
func parseId(id githubv4.ID) string {
	return fmt.Sprintf("%v", id)
}
