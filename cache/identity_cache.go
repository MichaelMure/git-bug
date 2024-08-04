package cache

import (
	"sync"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

var _ entity.Interface = &IdentityCache{}
var _ CacheEntity = &IdentityCache{}

// IdentityCache is a wrapper around an Identity for caching.
type IdentityCache struct {
	repo          repository.ClockedRepo
	entityUpdated func(id entity.Id) error

	mu sync.Mutex
	*identity.Identity
}

func NewIdentityCache(i *identity.Identity, repo repository.ClockedRepo, entityUpdated func(id entity.Id) error) *IdentityCache {
	return &IdentityCache{
		repo:          repo,
		entityUpdated: entityUpdated,
		Identity:      i,
	}
}

func (i *IdentityCache) notifyUpdated() error {
	return i.entityUpdated(i.Identity.Id())
}

func (i *IdentityCache) Mutate(repo repository.RepoClock, f func(*identity.Mutator)) error {
	i.mu.Lock()
	err := i.Identity.Mutate(repo, f)
	i.mu.Unlock()
	if err != nil {
		return err
	}
	return i.notifyUpdated()
}

func (i *IdentityCache) Commit() error {
	i.mu.Lock()
	err := i.Identity.Commit(i.repo)
	i.mu.Unlock()
	if err != nil {
		return err
	}
	return i.notifyUpdated()
}

func (i *IdentityCache) CommitAsNeeded() error {
	i.mu.Lock()
	err := i.Identity.CommitAsNeeded(i.repo)
	i.mu.Unlock()
	if err != nil {
		return err
	}
	return i.notifyUpdated()
}

func (i *IdentityCache) Lock() {
	i.mu.Lock()
}
