package board

import (
	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/repository"
)

// Fetch retrieve updates from a remote
// This does not change the local board state
func Fetch(repo repository.Repo, remote string) (string, error) {
	return dag.Fetch(def, repo, remote)
}

// Push update a remote with the local changes
func Push(repo repository.Repo, remote string) (string, error) {
	return dag.Push(def, repo, remote)
}

// Pull will do a Fetch + MergeAll
// This function will return an error if a merge fail
// Note: an author is necessary for the case where a merge commit is created, as this commit will
// have an author and may be signed if a signing key is available.
func Pull(repo repository.ClockedRepo, resolvers entity.Resolvers, remote string, mergeAuthor identity.Interface) error {
	return dag.Pull(def, wrapper, repo, resolvers, remote, mergeAuthor)
}

// MergeAll will merge all the available remote board
// Note: an author is necessary for the case where a merge commit is created, as this commit will
// have an author and may be signed if a signing key is available.
func MergeAll(repo repository.ClockedRepo, resolvers entity.Resolvers, remote string, mergeAuthor identity.Interface) <-chan entity.MergeResult {
	return dag.MergeAll(def, wrapper, repo, resolvers, remote, mergeAuthor)
}

// Remove will remove a local bug from its entity.Id
func Remove(repo repository.ClockedRepo, id entity.Id) error {
	return dag.Remove(def, repo, id)
}
