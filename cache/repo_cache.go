package cache

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"

	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/multierr"
	"github.com/MichaelMure/git-bug/util/process"
)

// 1: original format
// 2: added cache for identities with a reference in the bug cache
// 3: no more legacy identity
// 4: entities make their IDs from data, not git commit
const formatVersion = 4

// The maximum number of bugs loaded in memory. After that, eviction will be done.
const defaultMaxLoadedBugs = 1000

var _ repository.RepoCommon = &RepoCache{}
var _ repository.RepoConfig = &RepoCache{}
var _ repository.RepoKeyring = &RepoCache{}

// cacheMgmt is the expected interface for a sub-cache.
type cacheMgmt interface {
	Typename() string
	Load() error
	Build() <-chan BuildEvent
	SetCacheSize(size int)
	RemoveAll() error
	MergeAll(remote string) <-chan entity.MergeResult
	GetNamespace() string
	Close() error
}

// RepoCache is a cache for a Repository. This cache has multiple functions:
//
//  1. After being loaded, a Bug is kept in memory in the cache, allowing for fast
//     access later.
//  2. The cache maintain in memory and on disk a pre-digested excerpt for each bug,
//     allowing for fast querying the whole set of bugs without having to load
//     them individually.
//  3. The cache guarantee that a single instance of a Bug is loaded at once, avoiding
//     loss of data that we could have with multiple copies in the same process.
//  4. The same way, the cache maintain in memory a single copy of the loaded identities.
//
// The cache also protect the on-disk data by locking the git repository for its
// own usage, by writing a lock file. Of course, normal git operations are not
// affected, only git-bug related one.
type RepoCache struct {
	// the underlying repo
	repo repository.ClockedRepo

	// the name of the repository, as defined in the MultiRepoCache
	name string

	// resolvers for all known entities and excerpts
	resolvers entity.Resolvers

	bugs       *RepoCacheBug
	identities *RepoCacheIdentity

	subcaches []cacheMgmt

	// the user identity's id, if known
	muUserIdentity sync.RWMutex
	userIdentityId entity.Id
}

// NewRepoCache create or open a cache on top of a raw repository.
// The caller is expected to read all returned events before the cache is considered
// ready to use.
func NewRepoCache(r repository.ClockedRepo) (*RepoCache, chan BuildEvent) {
	return NewNamedRepoCache(r, defaultRepoName)
}

// NewNamedRepoCache create or open a named cache on top of a raw repository.
// The caller is expected to read all returned events before the cache is considered
// ready to use.
func NewNamedRepoCache(r repository.ClockedRepo, name string) (*RepoCache, chan BuildEvent) {
	c := &RepoCache{
		repo: r,
		name: name,
	}

	c.identities = NewRepoCacheIdentity(r, c.getResolvers, c.GetUserIdentity)
	c.subcaches = append(c.subcaches, c.identities)

	c.bugs = NewRepoCacheBug(r, c.getResolvers, c.GetUserIdentity)
	c.subcaches = append(c.subcaches, c.bugs)

	c.resolvers = entity.Resolvers{
		&IdentityCache{}:   entity.ResolverFunc[*IdentityCache](c.identities.Resolve),
		&IdentityExcerpt{}: entity.ResolverFunc[*IdentityExcerpt](c.identities.ResolveExcerpt),

		bug.Interface(nil): entity.ResolverFunc[*BugCache](c.bugs.Resolve),
		&BugCache{}:        entity.ResolverFunc[*BugCache](c.bugs.Resolve),
		&BugExcerpt{}:      entity.ResolverFunc[*BugExcerpt](c.bugs.ResolveExcerpt),
	}

	// small buffer so that below functions can emit an event without blocking
	events := make(chan BuildEvent)

	go func() {
		defer close(events)

		err := c.lock(events)
		if err != nil {
			events <- BuildEvent{Err: err}
			return
		}

		err = c.load()
		if err == nil {
			return
		}

		// Cache is either missing, broken or outdated. Rebuilding.
		c.buildCache(events)
	}()

	return c, events
}

func NewRepoCacheNoEvents(r repository.ClockedRepo) (*RepoCache, error) {
	cache, events := NewRepoCache(r)
	for event := range events {
		if event.Err != nil {
			for range events {
			}
			return nil, event.Err
		}
	}
	return cache, nil
}

// Bugs gives access to the Bug entities
func (c *RepoCache) Bugs() *RepoCacheBug {
	return c.bugs
}

// Identities gives access to the Identity entities
func (c *RepoCache) Identities() *RepoCacheIdentity {
	return c.identities
}

func (c *RepoCache) getResolvers() entity.Resolvers {
	return c.resolvers
}

// setCacheSize change the maximum number of loaded bugs
func (c *RepoCache) setCacheSize(size int) {
	for _, subcache := range c.subcaches {
		subcache.SetCacheSize(size)
	}
}

// load will try to read from the disk all the cache files
func (c *RepoCache) load() error {
	var errWait multierr.ErrWaitGroup
	for _, mgmt := range c.subcaches {
		errWait.Go(mgmt.Load)
	}
	return errWait.Wait()
}

func (c *RepoCache) lock(events chan BuildEvent) error {
	err := repoIsAvailable(c.repo, events)
	if err != nil {
		return err
	}

	f, err := c.repo.LocalStorage().Create(lockfile)
	if err != nil {
		return err
	}

	pid := fmt.Sprintf("%d", os.Getpid())
	_, err = f.Write([]byte(pid))
	if err != nil {
		_ = f.Close()
		return err
	}

	return f.Close()
}

func (c *RepoCache) Close() error {
	var errWait multierr.ErrWaitGroup
	for _, mgmt := range c.subcaches {
		errWait.Go(mgmt.Close)
	}
	err := errWait.Wait()
	if err != nil {
		return err
	}

	err = c.repo.Close()
	if err != nil {
		return err
	}

	return c.repo.LocalStorage().Remove(lockfile)
}

type BuildEventType int

const (
	_ BuildEventType = iota
	// BuildEventCacheIsBuilt signal that the cache is being built (aka, not skipped)
	BuildEventCacheIsBuilt
	// BuildEventRemoveLock signal that an old repo lock has been cleaned
	BuildEventRemoveLock
	// BuildEventStarted signal the beginning of a cache build for an entity
	BuildEventStarted
	// BuildEventProgress signal progress in the cache building for an entity
	BuildEventProgress
	// BuildEventFinished signal the end of a cache build for an entity
	BuildEventFinished
)

// BuildEvent carry an event happening during the cache build process.
type BuildEvent struct {
	// Err carry an error if the build process failed. If set, no other field matter.
	Err error
	// Typename is the name of the entity of which the event relate to. Can be empty if not particular entity is involved.
	Typename string
	// Event is the type of the event.
	Event BuildEventType
	// Total is the total number of element being built. Set if Event is BuildEventStarted.
	Total int64
	// Progress is the current count of processed element. Set if Event is BuildEventProgress.
	Progress int64
}

func (c *RepoCache) buildCache(events chan BuildEvent) {
	events <- BuildEvent{Event: BuildEventCacheIsBuilt}

	var wg sync.WaitGroup
	for _, subcache := range c.subcaches {
		wg.Add(1)
		go func(subcache cacheMgmt) {
			defer wg.Done()

			buildEvents := subcache.Build()
			for buildEvent := range buildEvents {
				events <- buildEvent
				if buildEvent.Err != nil {
					return
				}
			}
		}(subcache)
	}
	wg.Wait()
}

// repoIsAvailable check is the given repository is locked by a Cache.
// Note: this is a smart function that will clean the lock file if the
// corresponding process is not there anymore.
// If no error is returned, the repo is free to edit.
func repoIsAvailable(repo repository.RepoStorage, events chan BuildEvent) error {
	// Todo: this leave way for a racey access to the repo between the test
	// if the file exist and the actual write. It's probably not a problem in
	// practice because using a repository will be done from user interaction
	// or in a context where a single instance of git-bug is already guaranteed
	// (say, a server with the web UI running). But still, that might be nice to
	// have a mutex or something to guard that.

	// Todo: this will fail if somehow the filesystem is shared with another
	// computer. Should add a configuration that prevent the cleaning of the
	// lock file

	f, err := repo.LocalStorage().Open(lockfile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if err == nil {
		// lock file already exist
		buf, err := io.ReadAll(io.LimitReader(f, 10))
		if err != nil {
			_ = f.Close()
			return err
		}

		err = f.Close()
		if err != nil {
			return err
		}

		if len(buf) >= 10 {
			return fmt.Errorf("the lock file should be < 10 bytes")
		}

		pid, err := strconv.Atoi(string(buf))
		if err != nil {
			return err
		}

		if process.IsRunning(pid) {
			return fmt.Errorf("the repository you want to access is already locked by the process pid %d", pid)
		}

		// The lock file is just laying there after a crash, clean it

		events <- BuildEvent{Event: BuildEventRemoveLock}

		err = repo.LocalStorage().Remove(lockfile)
		if err != nil {
			return err
		}
	}

	return nil
}
