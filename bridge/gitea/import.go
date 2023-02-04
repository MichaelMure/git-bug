package gitea

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"code.gitea.io/sdk/gitea"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/bridge/gitea/iterator"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/util/text"
)

// giteaImporter implement the Importer interface
type giteaImporter struct {
	conf core.Configuration

	// default client
	client *gitea.Client

	// iterator
	iterator *iterator.Iterator

	// send only channel
	out chan<- core.ImportResult
}

func (gi *giteaImporter) Init(_ context.Context, repo *cache.RepoCache, conf core.Configuration) error {
	gi.conf = conf

	creds, err := auth.List(repo,
		auth.WithTarget(target),
		auth.WithKind(auth.KindToken),
		auth.WithMeta(auth.MetaKeyBaseURL, conf[confKeyBaseURL]),
		auth.WithMeta(auth.MetaKeyLogin, conf[confKeyDefaultLogin]),
	)
	if err != nil {
		return err
	}

	if len(creds) == 0 {
		return ErrMissingIdentityToken
	}

	gi.client, err = buildClient(conf[confKeyBaseURL], creds[0].(*auth.Token))
	if err != nil {
		return err
	}

	return nil
}

// ImportAll iterate over all the configured repository issues (comments) and ensure the creation
// of the missing issues / comments / label events / title changes ...
func (gi *giteaImporter) ImportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ImportResult, error) {
	gi.iterator = iterator.NewIterator(ctx, gi.client, 10, gi.conf[confKeyOwner], gi.conf[confKeyProject], defaultTimeout, since)
	out := make(chan core.ImportResult)
	gi.out = out

	go func() {
		defer close(gi.out)

		// Loop over all matching issues
		for gi.iterator.NextIssue() {
			issue := gi.iterator.IssueValue()

			// create issue
			b, err := gi.ensureIssue(repo, issue)
			if err != nil {
				err := fmt.Errorf("issue creation: %v", err)
				out <- core.NewImportError(err, "")
				return
			}

			// Loop over all comments

			// Loop over all label events

			if !b.NeedCommit() {
				out <- core.NewImportNothing(b.Id(), "no imported operation")
			} else if err := b.Commit(); err != nil {
				// commit bug state
				err := fmt.Errorf("bug commit: %v", err)
				out <- core.NewImportError(err, "")
				return
			}
		}

		if err := gi.iterator.Error(); err != nil {
			out <- core.NewImportError(err, "")
		}
	}()

	return out, nil
}

func (gi *giteaImporter) ensureIssue(repo *cache.RepoCache, issue *gitea.Issue) (*cache.BugCache, error) {
	// ensure issue author
	author, err := gi.ensurePerson(repo, issue.Poster.UserName)
	if err != nil {
		return nil, err
	}

	giteaID := strconv.FormatInt(issue.Index, 10)

	// resolve bug
	b, err := repo.ResolveBugMatcher(func(excerpt *cache.BugExcerpt) bool {
		return excerpt.CreateMetadata[core.MetaKeyOrigin] == target &&
			excerpt.CreateMetadata[metaKeyGiteaID] == giteaID &&
			excerpt.CreateMetadata[metaKeyGiteaBaseURL] == gi.conf[confKeyBaseURL] &&
			excerpt.CreateMetadata[metaKeyGiteaOwner] == gi.conf[confKeyOwner] &&
			excerpt.CreateMetadata[metaKeyGiteaProject] == gi.conf[confKeyProject]
	})
	if err == nil {
		return b, nil
	}
	if err != bug.ErrBugNotExist {
		return nil, err
	}

	// if bug was never imported, create bug
	b, _, err = repo.NewBugRaw(
		author,
		issue.Created.Unix(),
		text.CleanupOneLine(issue.Title),
		text.Cleanup(issue.Body),
		nil,
		map[string]string{
			core.MetaKeyOrigin:  target,
			metaKeyGiteaID:      giteaID,
			metaKeyGiteaOwner:   gi.conf[confKeyOwner],
			metaKeyGiteaProject: gi.conf[confKeyProject],
			metaKeyGiteaBaseURL: gi.conf[confKeyBaseURL],
		},
	)

	if err != nil {
		return nil, err
	}

	// importing a new bug
	gi.out <- core.NewImportBug(b.Id())

	return b, nil
}

func (gi *giteaImporter) ensurePerson(repo *cache.RepoCache, loginName string) (*cache.IdentityCache, error) {
	// Look first in the cache
	i, err := repo.ResolveIdentityImmutableMetadata(metaKeyGiteaLogin, loginName)
	if err == nil {
		return i, nil
	}
	if entity.IsErrMultipleMatch(err) {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	gi.client.SetContext(ctx)

	user, _, err := gi.client.GetUserInfo(loginName)
	if err != nil {
		if err.Error() == "404 Not Found" {
			user.FullName = loginName
			user.UserName = loginName
			user.Email = loginName + "@fake-email.com"
			user.AvatarURL = ""
		} else {
			return nil, err
		}
	}

	i, err = repo.NewIdentityRaw(
		user.FullName,
		user.Email,
		user.UserName,
		user.AvatarURL,
		nil,
		map[string]string{
			// because Gitea
			metaKeyGiteaLogin: user.UserName,
		},
	)
	if err != nil {
		return nil, err
	}

	gi.out <- core.NewImportIdentity(i.Id())
	return i, nil
}
