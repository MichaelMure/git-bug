package dag

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

// ListLocalIds list all the available local Entity's Id
func ListLocalIds(def Definition, repo repository.RepoData) ([]entity.Id, error) {
	refs, err := repo.ListRefs(fmt.Sprintf("refs/%s/", def.Namespace))
	if err != nil {
		return nil, err
	}
	return entity.RefsToIds(refs), nil
}

// Fetch retrieve updates from a remote
// This does not change the local entity state
func Fetch(def Definition, repo repository.Repo, remote string) (string, error) {
	return repo.FetchRefs(remote, def.Namespace)
}

// Push update a remote with the local changes
func Push(def Definition, repo repository.Repo, remote string) (string, error) {
	return repo.PushRefs(remote, def.Namespace)
}

// Pull will do a Fetch + MergeAll
// Contrary to MergeAll, this function will return an error if a merge fail.
func Pull[EntityT entity.Bare](def Definition, wrapper func(e *Entity) EntityT, repo repository.ClockedRepo, resolvers entity.Resolvers, remote string, author identity.Interface) error {
	_, err := Fetch(def, repo, remote)
	if err != nil {
		return err
	}

	for merge := range MergeAll(def, wrapper, repo, resolvers, remote, author) {
		if merge.Err != nil {
			return merge.Err
		}
		if merge.Status == entity.MergeStatusInvalid {
			return errors.Errorf("merge failure: %s", merge.Reason)
		}
	}

	return nil
}

// MergeAll will merge all the available remote Entity:
//
// Multiple scenario exist:
//  1. if the remote Entity doesn't exist locally, it's created
//     --> emit entity.MergeStatusNew
//  2. if the remote and local Entity have the same state, nothing is changed
//     --> emit entity.MergeStatusNothing
//  3. if the local Entity has new commits but the remote don't, nothing is changed
//     --> emit entity.MergeStatusNothing
//  4. if the remote has new commit, the local bug is updated to match the same history
//     (fast-forward update)
//     --> emit entity.MergeStatusUpdated
//  5. if both local and remote Entity have new commits (that is, we have a concurrent edition),
//     a merge commit with an empty operationPack is created to join both branch and form a DAG.
//     --> emit entity.MergeStatusUpdated
//
// Note: an author is necessary for the case where a merge commit is created, as this commit will
// have an author and may be signed if a signing key is available.
func MergeAll[EntityT entity.Bare](def Definition, wrapper func(e *Entity) EntityT, repo repository.ClockedRepo, resolvers entity.Resolvers, remote string, author identity.Interface) <-chan entity.MergeResult {
	out := make(chan entity.MergeResult)

	go func() {
		defer close(out)

		remoteRefSpec := fmt.Sprintf("refs/remotes/%s/%s/", remote, def.Namespace)
		remoteRefs, err := repo.ListRefs(remoteRefSpec)
		if err != nil {
			out <- entity.MergeResult{Err: err}
			return
		}

		for _, remoteRef := range remoteRefs {
			out <- merge[EntityT](def, wrapper, repo, resolvers, remoteRef, author)
		}
	}()

	return out
}

// merge perform a merge to make sure a local Entity is up-to-date.
// See MergeAll for more details.
func merge[EntityT entity.Bare](def Definition, wrapper func(e *Entity) EntityT, repo repository.ClockedRepo, resolvers entity.Resolvers, remoteRef string, author identity.Interface) entity.MergeResult {
	id := entity.RefToId(remoteRef)

	if err := id.Validate(); err != nil {
		return entity.NewMergeInvalidStatus(id, errors.Wrap(err, "invalid ref").Error())
	}

	remoteEntity, err := read[EntityT](def, wrapper, repo, resolvers, remoteRef)
	if err != nil {
		return entity.NewMergeInvalidStatus(id,
			errors.Wrapf(err, "remote %s is not readable", def.Typename).Error())
	}

	// Check for error in remote data
	if err := remoteEntity.Validate(); err != nil {
		return entity.NewMergeInvalidStatus(id,
			errors.Wrapf(err, "remote %s data is invalid", def.Typename).Error())
	}

	localRef := fmt.Sprintf("refs/%s/%s", def.Namespace, id.String())

	// SCENARIO 1
	// if the remote Entity doesn't exist locally, it's created

	localExist, err := repo.RefExist(localRef)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	if !localExist {
		// the bug is not local yet, simply create the reference
		err := repo.CopyRef(remoteRef, localRef)
		if err != nil {
			return entity.NewMergeError(err, id)
		}

		return entity.NewMergeNewStatus(id, remoteEntity)
	}

	localCommit, err := repo.ResolveRef(localRef)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	remoteCommit, err := repo.ResolveRef(remoteRef)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	// SCENARIO 2
	// if the remote and local Entity have the same state, nothing is changed

	if localCommit == remoteCommit {
		// nothing to merge
		return entity.NewMergeNothingStatus(id)
	}

	// SCENARIO 3
	// if the local Entity has new commits but the remote don't, nothing is changed

	localCommits, err := repo.ListCommits(localRef)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	for _, hash := range localCommits {
		if hash == remoteCommit {
			return entity.NewMergeNothingStatus(id)
		}
	}

	// SCENARIO 4
	// if the remote has new commit, the local bug is updated to match the same history
	// (fast-forward update)

	remoteCommits, err := repo.ListCommits(remoteRef)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	// fast-forward is possible if otherRef include ref
	fastForwardPossible := false
	for _, hash := range remoteCommits {
		if hash == localCommit {
			fastForwardPossible = true
			break
		}
	}

	if fastForwardPossible {
		err = repo.UpdateRef(localRef, remoteCommit)
		if err != nil {
			return entity.NewMergeError(err, id)
		}
		return entity.NewMergeUpdatedStatus(id, remoteEntity)
	}

	// SCENARIO 5
	// if both local and remote Entity have new commits (that is, we have a concurrent edition),
	// a merge commit with an empty operationPack is created to join both branch and form a DAG.

	// fast-forward is not possible, we need to create a merge commit
	// For simplicity when reading and to have clocks that record this change, we store
	// an empty operationPack.
	// First step is to collect those clocks.

	localEntity, err := read[EntityT](def, wrapper, repo, resolvers, localRef)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	editTime, err := repo.Increment(fmt.Sprintf(editClockPattern, def.Namespace))
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	opp := &operationPack{
		Author:     author,
		Operations: nil,
		CreateTime: 0,
		EditTime:   editTime,
	}

	commitHash, err := opp.Write(def, repo, localCommit, remoteCommit)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	// finally update the ref
	err = repo.UpdateRef(localRef, commitHash)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	// Note: we don't need to update localEntity state (lastCommit, operations...) as we
	// discard it entirely anyway.

	return entity.NewMergeUpdatedStatus(id, localEntity)
}

// Remove delete an Entity.
// Remove is idempotent.
func Remove(def Definition, repo repository.ClockedRepo, id entity.Id) error {
	var matches []string

	ref := fmt.Sprintf("refs/%s/%s", def.Namespace, id.String())
	matches = append(matches, ref)

	remotes, err := repo.GetRemotes()
	if err != nil {
		return err
	}

	for remote := range remotes {
		ref = fmt.Sprintf("refs/remotes/%s/%s/%s", remote, def.Namespace, id.String())
		matches = append(matches, ref)
	}

	for _, ref = range matches {
		err = repo.RemoveRef(ref)
		if err != nil {
			return err
		}
	}

	return nil
}

// RemoveAll delete all Entity matching the Definition.
// RemoveAll is idempotent.
func RemoveAll(def Definition, repo repository.ClockedRepo) error {
	localIds, err := ListLocalIds(def, repo)
	if err != nil {
		return err
	}
	for _, id := range localIds {
		err = Remove(def, repo, id)
		if err != nil {
			return err
		}
	}
	return nil
}
