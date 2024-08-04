package identity

import (
	bootstrap "github.com/MichaelMure/git-bug/entity/boostrap"
	"github.com/MichaelMure/git-bug/repository"
)

var _ bootstrap.Resolver = &SimpleResolver{}

// SimpleResolver is a Resolver loading Identities directly from a Repo
type SimpleResolver struct {
	repo repository.Repo
}

func NewSimpleResolver(repo repository.Repo) *SimpleResolver {
	return &SimpleResolver{repo: repo}
}

func (r *SimpleResolver) Resolve(id bootstrap.Id) (bootstrap.Resolved, error) {
	return ReadLocal(r.repo, id)
}
