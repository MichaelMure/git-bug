package cache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

type Excerpt interface {
	Id() entity.Id
	setId(id entity.Id)
}

type CacheEntity interface {
	Id() entity.Id
	NeedCommit() bool
	Lock()
}

type getUserIdentityFunc func() (*IdentityCache, error)

// Actions expose a number of action functions on Entities, to give upper layers (cache) a way to normalize interactions.
// Note: ideally this wouldn't exist, the cache layer would assume that everything is an entity/dag, and directly use the
// functions from this package, but right now identities are not using that framework.
type Actions[EntityT entity.Interface] struct {
	ReadWithResolver    func(repo repository.ClockedRepo, resolvers entity.Resolvers, id entity.Id) (EntityT, error)
	ReadAllWithResolver func(repo repository.ClockedRepo, resolvers entity.Resolvers) <-chan entity.StreamedEntity[EntityT]
	Remove              func(repo repository.ClockedRepo, id entity.Id) error
	RemoveAll           func(repo repository.ClockedRepo) error
	MergeAll            func(repo repository.ClockedRepo, resolvers entity.Resolvers, remote string, mergeAuthor entity.Interface) <-chan entity.MergeResult
}

var _ cacheMgmt = &SubCache[entity.Interface, Excerpt, CacheEntity]{}

type SubCache[EntityT entity.Interface, ExcerptT Excerpt, CacheT CacheEntity] struct {
	repo      repository.ClockedRepo
	resolvers func() entity.Resolvers

	getUserIdentity getUserIdentityFunc
	makeCached      func(entity EntityT, entityUpdated func(id entity.Id) error) CacheT
	makeExcerpt     func(CacheT) ExcerptT
	makeIndexData   func(CacheT) []string
	actions         Actions[EntityT]

	typename  string
	namespace string
	version   uint
	maxLoaded int

	mu       sync.RWMutex
	excerpts map[entity.Id]ExcerptT
	cached   map[entity.Id]CacheT
	lru      lruIdCache
}

func NewSubCache[EntityT entity.Interface, ExcerptT Excerpt, CacheT CacheEntity](
	repo repository.ClockedRepo,
	resolvers func() entity.Resolvers, getUserIdentity getUserIdentityFunc,
	makeCached func(entity EntityT, entityUpdated func(id entity.Id) error) CacheT,
	makeExcerpt func(CacheT) ExcerptT,
	makeIndexData func(CacheT) []string,
	actions Actions[EntityT],
	typename, namespace string,
	version uint, maxLoaded int) *SubCache[EntityT, ExcerptT, CacheT] {
	return &SubCache[EntityT, ExcerptT, CacheT]{
		repo:            repo,
		resolvers:       resolvers,
		getUserIdentity: getUserIdentity,
		makeCached:      makeCached,
		makeExcerpt:     makeExcerpt,
		makeIndexData:   makeIndexData,
		actions:         actions,
		typename:        typename,
		namespace:       namespace,
		version:         version,
		maxLoaded:       maxLoaded,
		excerpts:        make(map[entity.Id]ExcerptT),
		cached:          make(map[entity.Id]CacheT),
		lru:             newLRUIdCache(),
	}
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) Typename() string {
	return sc.typename
}

// Load will try to read from the disk the entity cache file
func (sc *SubCache[EntityT, ExcerptT, CacheT]) Load() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	f, err := sc.repo.LocalStorage().Open(filepath.Join("cache", sc.namespace))
	if err != nil {
		return err
	}

	aux := struct {
		Version  uint
		Excerpts map[entity.Id]ExcerptT
	}{}

	decoder := gob.NewDecoder(f)
	err = decoder.Decode(&aux)
	if err != nil {
		_ = f.Close()
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	if aux.Version != sc.version {
		return fmt.Errorf("unknown %s cache format version %v", sc.namespace, aux.Version)
	}

	// the id is not serialized in the excerpt itself (non-exported field in go, long story ...),
	// so we fix it here, which doubles as enforcing coherency.
	for id, excerpt := range aux.Excerpts {
		excerpt.setId(id)
	}

	sc.excerpts = aux.Excerpts

	index, err := sc.repo.GetIndex(sc.namespace)
	if err != nil {
		return err
	}

	// simple heuristic to detect a mismatch between the index and the entities
	count, err := index.DocCount()
	if err != nil {
		return err
	}
	if count != uint64(len(sc.excerpts)) {
		return fmt.Errorf("count mismatch between bleve and %s excerpts", sc.namespace)
	}

	// TODO: find a way to check lamport clocks

	return nil
}

// Write will serialize on disk the entity cache file
func (sc *SubCache[EntityT, ExcerptT, CacheT]) write() error {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	var data bytes.Buffer

	aux := struct {
		Version  uint
		Excerpts map[entity.Id]ExcerptT
	}{
		Version:  sc.version,
		Excerpts: sc.excerpts,
	}

	encoder := gob.NewEncoder(&data)

	err := encoder.Encode(aux)
	if err != nil {
		return err
	}

	f, err := sc.repo.LocalStorage().Create(filepath.Join("cache", sc.namespace))
	if err != nil {
		return err
	}

	_, err = f.Write(data.Bytes())
	if err != nil {
		_ = f.Close()
		return err
	}

	return f.Close()
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) Build() <-chan BuildEvent {
	// value chosen experimentally as giving the fasted indexing, while
	// not driving the cache size on disk too high.
	//
	// | batchCount | bugIndex (MB) | idIndex (kB) | time (s) |
	// |:----------:|:-------------:|:------------:|:--------:|
	// |     10     |      24       |      84      |   1,59   |
	// |     30     |      26       |      84      |  1,388   |
	// |     50     |      26       |      84      |   1,44   |
	// |     60     |      26       |      80      |  1,377   |
	// |     68     |      27       |      80      |  1,385   |
	// |     75     |      26       |      84      |   1,32   |
	// |     80     |      26       |      80      |   1,37   |
	// |     85     |      27       |      80      |  1,317   |
	// |    100     |      26       |      80      |  1,455   |
	// |    150     |      26       |      80      |  2,066   |
	// |    200     |      28       |      80      |  2,885   |
	// |    250     |      30       |      72      |  3,555   |
	// |    300     |      31       |      72      |  4,787   |
	// |    500     |      23       |      72      |   5,4    |
	const maxBatchCount = 75

	out := make(chan BuildEvent)

	go func() {
		defer close(out)

		out <- BuildEvent{
			Typename: sc.typename,
			Event:    BuildEventStarted,
		}

		sc.excerpts = make(map[entity.Id]ExcerptT)

		allEntities := sc.actions.ReadAllWithResolver(sc.repo, sc.resolvers())

		index, err := sc.repo.GetIndex(sc.namespace)
		if err != nil {
			out <- BuildEvent{
				Typename: sc.typename,
				Err:      err,
			}
			return
		}

		// wipe the index just to be sure
		err = index.Clear()
		if err != nil {
			out <- BuildEvent{
				Typename: sc.typename,
				Err:      err,
			}
			return
		}

		indexer, indexEnd := index.IndexBatch()
		var batchCount int

		for e := range allEntities {
			if e.Err != nil {
				out <- BuildEvent{
					Typename: sc.typename,
					Err:      e.Err,
				}
				return
			}

			cached := sc.makeCached(e.Entity, sc.entityUpdated)
			sc.excerpts[e.Entity.Id()] = sc.makeExcerpt(cached)
			// might as well keep them in memory
			sc.cached[e.Entity.Id()] = cached

			indexData := sc.makeIndexData(cached)
			if err := indexer(e.Entity.Id().String(), indexData); err != nil {
				out <- BuildEvent{
					Typename: sc.typename,
					Err:      err,
				}
				return
			}

			batchCount++
			if batchCount >= maxBatchCount {
				err = indexEnd()
				if err != nil {
					out <- BuildEvent{
						Typename: sc.typename,
						Err:      err,
					}
					return
				}

				indexer, indexEnd = index.IndexBatch()
				batchCount = 0
			}

			out <- BuildEvent{
				Typename: sc.typename,
				Event:    BuildEventProgress,
				Progress: e.CurrentEntity,
				Total:    e.TotalEntities,
			}
		}

		if batchCount > 0 {
			err = indexEnd()
			if err != nil {
				out <- BuildEvent{
					Typename: sc.typename,
					Err:      err,
				}
				return
			}
		}

		err = sc.write()
		if err != nil {
			out <- BuildEvent{
				Typename: sc.typename,
				Err:      err,
			}
			return
		}

		out <- BuildEvent{
			Typename: sc.typename,
			Event:    BuildEventFinished,
		}
	}()

	return out
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) SetCacheSize(size int) {
	sc.maxLoaded = size
	sc.evictIfNeeded()
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) Close() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.excerpts = nil
	sc.cached = make(map[entity.Id]CacheT)
	return nil
}

// AllIds return all known bug ids
func (sc *SubCache[EntityT, ExcerptT, CacheT]) AllIds() []entity.Id {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	result := make([]entity.Id, len(sc.excerpts))

	i := 0
	for _, excerpt := range sc.excerpts {
		result[i] = excerpt.Id()
		i++
	}

	return result
}

// Resolve retrieve an entity matching the exact given id
func (sc *SubCache[EntityT, ExcerptT, CacheT]) Resolve(id entity.Id) (CacheT, error) {
	sc.mu.RLock()
	cached, ok := sc.cached[id]
	if ok {
		sc.lru.Get(id)
		sc.mu.RUnlock()
		return cached, nil
	}
	sc.mu.RUnlock()

	e, err := sc.actions.ReadWithResolver(sc.repo, sc.resolvers(), id)
	if err != nil {
		return *new(CacheT), err
	}

	cached = sc.makeCached(e, sc.entityUpdated)

	sc.mu.Lock()
	sc.cached[id] = cached
	sc.lru.Add(id)
	sc.mu.Unlock()

	sc.evictIfNeeded()

	return cached, nil
}

// ResolvePrefix retrieve an entity matching an id prefix. It fails if multiple
// entities match.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) ResolvePrefix(prefix string) (CacheT, error) {
	return sc.ResolveMatcher(func(excerpt ExcerptT) bool {
		return excerpt.Id().HasPrefix(prefix)
	})
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) ResolveMatcher(f func(ExcerptT) bool) (CacheT, error) {
	id, err := sc.resolveMatcher(f)
	if err != nil {
		return *new(CacheT), err
	}
	return sc.Resolve(id)
}

// ResolveExcerpt retrieve an Excerpt matching the exact given id
func (sc *SubCache[EntityT, ExcerptT, CacheT]) ResolveExcerpt(id entity.Id) (ExcerptT, error) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	excerpt, ok := sc.excerpts[id]
	if !ok {
		return *new(ExcerptT), entity.NewErrNotFound(sc.typename)
	}

	return excerpt, nil
}

// ResolveExcerptPrefix retrieve an Excerpt matching an id prefix. It fails if multiple
// entities match.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) ResolveExcerptPrefix(prefix string) (ExcerptT, error) {
	return sc.ResolveExcerptMatcher(func(excerpt ExcerptT) bool {
		return excerpt.Id().HasPrefix(prefix)
	})
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) ResolveExcerptMatcher(f func(ExcerptT) bool) (ExcerptT, error) {
	id, err := sc.resolveMatcher(f)
	if err != nil {
		return *new(ExcerptT), err
	}
	return sc.ResolveExcerpt(id)
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) resolveMatcher(f func(ExcerptT) bool) (entity.Id, error) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// preallocate but empty
	matching := make([]entity.Id, 0, 5)

	for _, excerpt := range sc.excerpts {
		if f(excerpt) {
			matching = append(matching, excerpt.Id())
		}
	}

	if len(matching) > 1 {
		return entity.UnsetId, entity.NewErrMultipleMatch(sc.typename, matching)
	}

	if len(matching) == 0 {
		return entity.UnsetId, entity.NewErrNotFound(sc.typename)
	}

	return matching[0], nil
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) add(e EntityT) (CacheT, error) {
	sc.mu.Lock()
	if _, has := sc.cached[e.Id()]; has {
		sc.mu.Unlock()
		return *new(CacheT), fmt.Errorf("entity %s already exist in the cache", e.Id())
	}

	cached := sc.makeCached(e, sc.entityUpdated)
	sc.cached[e.Id()] = cached
	sc.lru.Add(e.Id())
	sc.mu.Unlock()

	sc.evictIfNeeded()

	// force the write of the excerpt
	err := sc.entityUpdated(e.Id())
	if err != nil {
		return *new(CacheT), err
	}

	return cached, nil
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) Remove(prefix string) error {
	e, err := sc.ResolvePrefix(prefix)
	if err != nil {
		return err
	}

	sc.mu.Lock()

	err = sc.actions.Remove(sc.repo, e.Id())
	if err != nil {
		sc.mu.Unlock()
		return err
	}

	delete(sc.cached, e.Id())
	delete(sc.excerpts, e.Id())
	sc.lru.Remove(e.Id())

	index, err := sc.repo.GetIndex(sc.namespace)
	if err != nil {
		sc.mu.Unlock()
		return err
	}

	err = index.Remove(e.Id().String())
	sc.mu.Unlock()
	if err != nil {
		return err
	}

	return sc.write()
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) RemoveAll() error {
	sc.mu.Lock()

	err := sc.actions.RemoveAll(sc.repo)
	if err != nil {
		sc.mu.Unlock()
		return err
	}

	for id, _ := range sc.cached {
		delete(sc.cached, id)
		sc.lru.Remove(id)
	}
	for id, _ := range sc.excerpts {
		delete(sc.excerpts, id)
	}

	index, err := sc.repo.GetIndex(sc.namespace)
	if err != nil {
		sc.mu.Unlock()
		return err
	}

	err = index.Clear()
	sc.mu.Unlock()
	if err != nil {
		return err
	}

	return sc.write()
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) MergeAll(remote string) <-chan entity.MergeResult {
	out := make(chan entity.MergeResult)

	// Intercept merge results to update the cache properly
	go func() {
		defer close(out)

		author, err := sc.getUserIdentity()
		if err != nil {
			out <- entity.NewMergeError(err, "")
			return
		}

		results := sc.actions.MergeAll(sc.repo, sc.resolvers(), remote, author)
		for result := range results {
			out <- result

			if result.Err != nil {
				continue
			}

			switch result.Status {
			case entity.MergeStatusNew, entity.MergeStatusUpdated:
				e := result.Entity.(EntityT)
				cached := sc.makeCached(e, sc.entityUpdated)

				sc.mu.Lock()
				sc.excerpts[result.Id] = sc.makeExcerpt(cached)
				// might as well keep them in memory
				sc.cached[result.Id] = cached
				sc.mu.Unlock()
			}
		}

		err = sc.write()
		if err != nil {
			out <- entity.NewMergeError(err, "")
			return
		}
	}()

	return out

}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) GetNamespace() string {
	return sc.namespace
}

// entityUpdated is a callback to trigger when the excerpt of an entity changed
func (sc *SubCache[EntityT, ExcerptT, CacheT]) entityUpdated(id entity.Id) error {
	sc.mu.Lock()
	e, ok := sc.cached[id]
	if !ok {
		sc.mu.Unlock()

		// if the bug is not loaded at this point, it means it was loaded before
		// but got evicted. Which means we potentially have multiple copies in
		// memory and thus concurrent write.
		// Failing immediately here is the simple and safe solution to avoid
		// complicated data loss.
		return errors.New("entity missing from cache")
	}
	sc.lru.Get(id)
	sc.excerpts[id] = sc.makeExcerpt(e)
	sc.mu.Unlock()

	index, err := sc.repo.GetIndex(sc.namespace)
	if err != nil {
		return err
	}

	err = index.IndexOne(e.Id().String(), sc.makeIndexData(e))
	if err != nil {
		return err
	}

	return sc.write()
}

// evictIfNeeded will evict an entity from the cache if needed
func (sc *SubCache[EntityT, ExcerptT, CacheT]) evictIfNeeded() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if sc.lru.Len() <= sc.maxLoaded {
		return
	}

	for _, id := range sc.lru.GetOldestToNewest() {
		b := sc.cached[id]
		if b.NeedCommit() {
			continue
		}

		// as a form of assurance that evicted entities don't get manipulated, we lock them here.
		// if something tries to do it anyway, it will lock the program and make it obvious.
		b.Lock()

		sc.lru.Remove(id)
		delete(sc.cached, id)

		if sc.lru.Len() <= sc.maxLoaded {
			return
		}
	}
}
