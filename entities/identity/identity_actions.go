package identity

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	bootstrap "github.com/MichaelMure/git-bug/entity/boostrap"
	"github.com/MichaelMure/git-bug/repository"
)

// Fetch retrieve updates from a remote
// This does not change the local identities state
func Fetch(repo repository.Repo, remote string) (string, error) {
	return repo.FetchRefs(remote, Namespace)
}

// Push update a remote with the local changes
func Push(repo repository.Repo, remote string) (string, error) {
	return repo.PushRefs(remote, Namespace)
}

// Pull will do a Fetch + MergeAll
// This function will return an error if a merge fail
func Pull(repo repository.ClockedRepo, remote string) error {
	_, err := Fetch(repo, remote)
	if err != nil {
		return err
	}

	for merge := range MergeAll(repo, remote) {
		if merge.Err != nil {
			return merge.Err
		}
		if merge.Status == bootstrap.MergeStatusInvalid {
			return errors.Errorf("merge failure: %s", merge.Reason)
		}
	}

	return nil
}

// MergeAll will merge all the available remote identity
func MergeAll(repo repository.ClockedRepo, remote string) <-chan bootstrap.MergeResult {
	out := make(chan bootstrap.MergeResult)

	go func() {
		defer close(out)

		remoteRefSpec := fmt.Sprintf(identityRemoteRefPattern, remote)
		remoteRefs, err := repo.ListRefs(remoteRefSpec)

		if err != nil {
			out <- bootstrap.MergeResult{Err: err}
			return
		}

		for _, remoteRef := range remoteRefs {
			refSplit := strings.Split(remoteRef, "/")
			id := bootstrap.Id(refSplit[len(refSplit)-1])

			if err := id.Validate(); err != nil {
				out <- bootstrap.NewMergeInvalidStatus(id, errors.Wrap(err, "invalid ref").Error())
				continue
			}

			remoteIdentity, err := read(repo, remoteRef)

			if err != nil {
				out <- bootstrap.NewMergeInvalidStatus(id, errors.Wrap(err, "remote identity is not readable").Error())
				continue
			}

			// Check for error in remote data
			if err := remoteIdentity.Validate(); err != nil {
				out <- bootstrap.NewMergeInvalidStatus(id, errors.Wrap(err, "remote identity is invalid").Error())
				continue
			}

			localRef := identityRefPattern + remoteIdentity.Id().String()
			localExist, err := repo.RefExist(localRef)

			if err != nil {
				out <- bootstrap.NewMergeError(err, id)
				continue
			}

			// the identity is not local yet, simply create the reference
			if !localExist {
				err := repo.CopyRef(remoteRef, localRef)

				if err != nil {
					out <- bootstrap.NewMergeError(err, id)
					return
				}

				out <- bootstrap.NewMergeNewStatus(id, remoteIdentity)
				continue
			}

			localIdentity, err := read(repo, localRef)

			if err != nil {
				out <- bootstrap.NewMergeError(errors.Wrap(err, "local identity is not readable"), id)
				return
			}

			updated, err := localIdentity.Merge(repo, remoteIdentity)

			if err != nil {
				out <- bootstrap.NewMergeInvalidStatus(id, errors.Wrap(err, "merge failed").Error())
				return
			}

			if updated {
				out <- bootstrap.NewMergeUpdatedStatus(id, localIdentity)
			} else {
				out <- bootstrap.NewMergeNothingStatus(id)
			}
		}
	}()

	return out
}

// Remove will remove a local identity from its entity.Id.
// It is left as a responsibility to the caller to make sure that this identities is not
// linked from another entity, otherwise it would break it.
// Remove is idempotent.
func Remove(repo repository.ClockedRepo, id bootstrap.Id) error {
	var fullMatches []string

	refs, err := repo.ListRefs(identityRefPattern + id.String())
	if err != nil {
		return err
	}
	if len(refs) > 1 {
		return bootstrap.NewErrMultipleMatch(Typename, bootstrap.RefsToIds(refs))
	}
	if len(refs) == 1 {
		// we have the identity locally
		fullMatches = append(fullMatches, refs[0])
	}

	remotes, err := repo.GetRemotes()
	if err != nil {
		return err
	}

	for remote := range remotes {
		remotePrefix := fmt.Sprintf(identityRemoteRefPattern+id.String(), remote)
		remoteRefs, err := repo.ListRefs(remotePrefix)
		if err != nil {
			return err
		}
		if len(remoteRefs) > 1 {
			return bootstrap.NewErrMultipleMatch(Typename, bootstrap.RefsToIds(refs))
		}
		if len(remoteRefs) == 1 {
			// found the identity in a remote
			fullMatches = append(fullMatches, remoteRefs[0])
		}
	}

	if len(fullMatches) == 0 {
		return bootstrap.NewErrNotFound(Typename)
	}

	for _, ref := range fullMatches {
		err = repo.RemoveRef(ref)
		if err != nil {
			return err
		}
	}

	return nil
}

// RemoveAll will remove all local identities.
// It is left as a responsibility to the caller to make sure that those identities are not
// linked from another entity, otherwise it would break them.
// RemoveAll is idempotent.
func RemoveAll(repo repository.ClockedRepo) error {
	localIds, err := ListLocalIds(repo)
	if err != nil {
		return err
	}
	for _, id := range localIds {
		err = Remove(repo, id)
		if err != nil {
			return err
		}
	}
	return nil
}
