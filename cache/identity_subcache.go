package cache

import (
	"fmt"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

type RepoCacheIdentity struct {
	*SubCache[*identity.Identity, *IdentityExcerpt, *IdentityCache]
}

func NewRepoCacheIdentity(repo repository.ClockedRepo,
	resolvers func() entity.Resolvers,
	getUserIdentity getUserIdentityFunc) *RepoCacheIdentity {

	makeCached := func(i *identity.Identity, entityUpdated func(id entity.Id) error) *IdentityCache {
		return NewIdentityCache(i, repo, entityUpdated)
	}

	makeIndex := func(i *IdentityCache) []string {
		// no indexing
		return nil
	}

	// TODO: this is terribly ugly, but we are currently stuck with the fact that identities are NOT using the fancy dag framework.
	//   This lead to various complication here and there to handle entities generically, and avoid large code duplication.
	//   TL;DR: something has to give, and this is the less ugly solution I found. This "normalize" identities as just another "dag framework"
	//   entity. Ideally identities would be converted to the dag framework, but right now that could lead to potential attack: if an old
	//   private key is leaked, it would be possible to craft a legal identity update that take over the most recent version. While this is
	//   meaningless in the case of a normal entity, it's really an issues for identities.

	actions := Actions[*identity.Identity]{
		ReadWithResolver: func(repo repository.ClockedRepo, resolvers entity.Resolvers, id entity.Id) (*identity.Identity, error) {
			return identity.ReadLocal(repo, id)
		},
		ReadAllWithResolver: func(repo repository.ClockedRepo, resolvers entity.Resolvers) <-chan entity.StreamedEntity[*identity.Identity] {
			return identity.ReadAllLocal(repo)
		},
		Remove:    identity.Remove,
		RemoveAll: identity.RemoveAll,
		MergeAll: func(repo repository.ClockedRepo, resolvers entity.Resolvers, remote string, mergeAuthor entity.Interface) <-chan entity.MergeResult {
			return identity.MergeAll(repo, remote)
		},
	}

	sc := NewSubCache[*identity.Identity, *IdentityExcerpt, *IdentityCache](
		repo, resolvers, getUserIdentity,
		makeCached, NewIdentityExcerpt, makeIndex, actions,
		identity.Typename, identity.Namespace,
		formatVersion, defaultMaxLoadedBugs,
	)

	return &RepoCacheIdentity{SubCache: sc}
}

// ResolveIdentityImmutableMetadata retrieve an Identity that has the exact given metadata on
// one of its version. If multiple version have the same key, the first defined take precedence.
func (c *RepoCacheIdentity) ResolveIdentityImmutableMetadata(key string, value string) (*IdentityCache, error) {
	return c.ResolveMatcher(func(excerpt *IdentityExcerpt) bool {
		return excerpt.ImmutableMetadata[key] == value
	})
}

// New create a new identity
// The new identity is written in the repository (commit)
func (c *RepoCacheIdentity) New(name string, email string) (*IdentityCache, error) {
	return c.NewRaw(name, email, "", "", nil, nil)
}

// NewFull create a new identity
// The new identity is written in the repository (commit)
func (c *RepoCacheIdentity) NewFull(name string, email string, login string, avatarUrl string, keys []*identity.Key) (*IdentityCache, error) {
	return c.NewRaw(name, email, login, avatarUrl, keys, nil)
}

func (c *RepoCacheIdentity) NewRaw(name string, email string, login string, avatarUrl string, keys []*identity.Key, metadata map[string]string) (*IdentityCache, error) {
	i, err := identity.NewIdentityFull(c.repo, name, email, login, avatarUrl, keys)
	if err != nil {
		return nil, err
	}
	return c.finishIdentity(i, metadata)
}

func (c *RepoCacheIdentity) NewFromGitUser() (*IdentityCache, error) {
	return c.NewFromGitUserRaw(nil)
}

func (c *RepoCacheIdentity) NewFromGitUserRaw(metadata map[string]string) (*IdentityCache, error) {
	i, err := identity.NewFromGitUser(c.repo)
	if err != nil {
		return nil, err
	}
	return c.finishIdentity(i, metadata)
}

func (c *RepoCacheIdentity) finishIdentity(i *identity.Identity, metadata map[string]string) (*IdentityCache, error) {
	for key, value := range metadata {
		i.SetMetadata(key, value)
	}

	err := i.Commit(c.repo)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	if _, has := c.cached[i.Id()]; has {
		return nil, fmt.Errorf("identity %s already exist in the cache", i.Id())
	}

	cached := NewIdentityCache(i, c.repo, c.entityUpdated)
	c.cached[i.Id()] = cached
	c.mu.Unlock()

	// force the write of the excerpt
	err = c.entityUpdated(i.Id())
	if err != nil {
		return nil, err
	}

	return cached, nil
}
