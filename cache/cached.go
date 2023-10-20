package cache

import (
	"sync"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
)

var _ dag.Interface[dag.Snapshot, dag.Operation] = &CachedEntityBase[dag.Snapshot, dag.Operation]{}
var _ CacheEntity = &CachedEntityBase[dag.Snapshot, dag.Operation]{}

// CachedEntityBase provide the base function of an entity managed by the cache.
type CachedEntityBase[SnapT dag.Snapshot, OpT dag.Operation] struct {
	repo            repository.ClockedRepo
	entityUpdated   func(id entity.Id) error
	getUserIdentity getUserIdentityFunc

	mu     sync.RWMutex
	entity dag.Interface[SnapT, OpT]
}

func (e *CachedEntityBase[SnapT, OpT]) Id() entity.Id {
	return e.entity.Id()
}

func (e *CachedEntityBase[SnapT, OpT]) Compile() SnapT {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.entity.Compile()
}

func (e *CachedEntityBase[SnapT, OpT]) notifyUpdated() error {
	return e.entityUpdated(e.entity.Id())
}

// ResolveOperationWithMetadata will find an operation that has the matching metadata
func (e *CachedEntityBase[SnapT, OpT]) ResolveOperationWithMetadata(key string, value string) (entity.Id, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	// preallocate but empty
	matching := make([]entity.Id, 0, 5)

	for _, op := range e.entity.Operations() {
		opValue, ok := op.GetMetadata(key)
		if ok && value == opValue {
			matching = append(matching, op.Id())
		}
	}

	if len(matching) == 0 {
		return "", ErrNoMatchingOp
	}

	if len(matching) > 1 {
		return "", entity.NewErrMultipleMatch("operation", matching)
	}

	return matching[0], nil
}

func (e *CachedEntityBase[SnapT, OpT]) Validate() error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.entity.Validate()
}

func (e *CachedEntityBase[SnapT, OpT]) Append(op OpT) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.entity.Append(op)
}

func (e *CachedEntityBase[SnapT, OpT]) Operations() []OpT {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.entity.Operations()
}

func (e *CachedEntityBase[SnapT, OpT]) Commit() error {
	e.mu.Lock()
	err := e.entity.Commit(e.repo)
	if err != nil {
		e.mu.Unlock()
		return err
	}
	e.mu.Unlock()
	return e.notifyUpdated()
}

func (e *CachedEntityBase[SnapT, OpT]) CommitAsNeeded() error {
	e.mu.Lock()
	err := e.entity.CommitAsNeeded(e.repo)
	if err != nil {
		e.mu.Unlock()
		return err
	}
	e.mu.Unlock()
	return e.notifyUpdated()
}

func (e *CachedEntityBase[SnapT, OpT]) NeedCommit() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.entity.NeedCommit()
}

func (e *CachedEntityBase[SnapT, OpT]) Lock() {
	e.mu.Lock()
}

func (e *CachedEntityBase[SnapT, OpT]) CreateLamportTime() lamport.Time {
	return e.entity.CreateLamportTime()
}

func (e *CachedEntityBase[SnapT, OpT]) EditLamportTime() lamport.Time {
	return e.entity.EditLamportTime()
}

func (e *CachedEntityBase[SnapT, OpT]) FirstOp() OpT {
	return e.entity.FirstOp()
}
