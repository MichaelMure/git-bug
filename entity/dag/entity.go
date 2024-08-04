// Package dag contains the base common code to define an entity stored
// in a chain of git objects, supporting actions like Push, Pull and Merge.
package dag

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	bootstrap "github.com/MichaelMure/git-bug/entity/boostrap"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
)

const refsPattern = "refs/%s/%s"
const creationClockPattern = "%s-create"
const editClockPattern = "%s-edit"

type OperationUnmarshaler func(raw json.RawMessage, resolver entity.Resolvers) (Operation, error)

// Definition hold the details defining one specialization of an Entity.
type Definition struct {
	// the name of the entity (bug, pull-request, ...), for human consumption
	Typename string
	// the Namespace in git references (bugs, prs, ...)
	Namespace string
	// a function decoding a JSON message into an Operation
	OperationUnmarshaler OperationUnmarshaler
	// the expected format version number, that can be used for data migration/upgrade
	FormatVersion uint
}

// Entity is a data structure stored in a chain of git objects, supporting actions like Push, Pull and Merge.
type Entity struct {
	// A Lamport clock is a logical clock that allow to order event
	// inside a distributed system.
	// It must be the first field in this struct due to https://github.com/golang/go/issues/36606
	createTime lamport.Time
	editTime   lamport.Time

	Definition

	// operations that are already stored in the repository
	ops []Operation
	// operations not yet stored in the repository
	staging []Operation

	lastCommit repository.Hash
}

// New create an empty Entity
func New(definition Definition) *Entity {
	return &Entity{
		Definition: definition,
	}
}

// Read will read and decode a stored local Entity from a repository
func Read[EntityT entity.Bare](def Definition, wrapper func(e *Entity) EntityT, repo repository.ClockedRepo, resolvers entity.Resolvers, id entity.Id) (EntityT, error) {
	if err := id.Validate(); err != nil {
		return *new(EntityT), errors.Wrap(err, "invalid id")
	}

	ref := fmt.Sprintf("refs/%s/%s", def.Namespace, id.String())

	return read[EntityT](def, wrapper, repo, resolvers, ref)
}

// readRemote will read and decode a stored remote Entity from a repository
func readRemote[EntityT entity.Bare](def Definition, wrapper func(e *Entity) EntityT, repo repository.ClockedRepo, resolvers entity.Resolvers, remote string, id entity.Id) (EntityT, error) {
	if err := id.Validate(); err != nil {
		return *new(EntityT), errors.Wrap(err, "invalid id")
	}

	ref := fmt.Sprintf("refs/remotes/%s/%s/%s", def.Namespace, remote, id.String())

	return read[EntityT](def, wrapper, repo, resolvers, ref)
}

// read fetch from git and decode an Entity at an arbitrary git reference.
func read[EntityT entity.Bare](def Definition, wrapper func(e *Entity) EntityT, repo repository.ClockedRepo, resolvers entity.Resolvers, ref string) (EntityT, error) {
	rootHash, err := repo.ResolveRef(ref)
	if err == repository.ErrNotFound {
		return *new(EntityT), entity.NewErrNotFound(def.Typename)
	}
	if err != nil {
		return *new(EntityT), err
	}

	// Perform a breadth-first search to get a topological order of the DAG where we discover the
	// parents commit and go back in time up to the chronological root

	queue := make([]repository.Hash, 0, 32)
	visited := make(map[repository.Hash]struct{})
	BFSOrder := make([]repository.Commit, 0, 32)

	queue = append(queue, rootHash)
	visited[rootHash] = struct{}{}

	for len(queue) > 0 {
		// pop
		hash := queue[0]
		queue = queue[1:]

		commit, err := repo.ReadCommit(hash)
		if err != nil {
			return *new(EntityT), err
		}

		BFSOrder = append(BFSOrder, commit)

		for _, parent := range commit.Parents {
			if _, ok := visited[parent]; !ok {
				queue = append(queue, parent)
				// mark as visited
				visited[parent] = struct{}{}
			}
		}
	}

	// Now, we can reverse this topological order and read the commits in an order where
	// we are sure to have read all the chronological ancestors when we read a commit.

	// Next step is to:
	// 1) read the operationPacks
	// 2) make sure that clocks causality respect the DAG topology.

	oppMap := make(map[repository.Hash]*operationPack)
	var opsCount int

	for i := len(BFSOrder) - 1; i >= 0; i-- {
		commit := BFSOrder[i]
		isFirstCommit := i == len(BFSOrder)-1
		isMerge := len(commit.Parents) > 1

		// Verify DAG structure: single chronological root, so only the root
		// can have no parents. Said otherwise, the DAG need to have exactly
		// one leaf.
		if !isFirstCommit && len(commit.Parents) == 0 {
			return *new(EntityT), fmt.Errorf("multiple leafs in the entity DAG")
		}

		opp, err := readOperationPack(def, repo, resolvers, commit)
		if err != nil {
			return *new(EntityT), err
		}

		err = opp.Validate()
		if err != nil {
			return *new(EntityT), err
		}

		if isMerge && len(opp.Operations) > 0 {
			return *new(EntityT), fmt.Errorf("merge commit cannot have operations")
		}

		// Check that the create lamport clock is set (not checked in Validate() as it's optional)
		if isFirstCommit && opp.CreateTime <= 0 {
			return *new(EntityT), fmt.Errorf("creation lamport time not set")
		}

		// make sure that the lamport clocks causality match the DAG topology
		for _, parentHash := range commit.Parents {
			parentPack, ok := oppMap[parentHash]
			if !ok {
				panic("DFS failed")
			}

			if parentPack.EditTime >= opp.EditTime {
				return *new(EntityT), fmt.Errorf("lamport clock ordering doesn't match the DAG")
			}

			// to avoid an attack where clocks are pushed toward the uint64 rollover, make sure
			// that the clocks don't jump too far in the future
			// we ignore merge commits here to allow merging after a loooong time without breaking anything,
			// as long as there is one valid chain of small hops, it's fine.
			if !isMerge && opp.EditTime-parentPack.EditTime > 1_000_000 {
				return *new(EntityT), fmt.Errorf("lamport clock jumping too far in the future, likely an attack")
			}
		}

		oppMap[commit.Hash] = opp
		opsCount += len(opp.Operations)
	}

	// The clocks are fine, we witness them
	for _, opp := range oppMap {
		err = repo.Witness(fmt.Sprintf(creationClockPattern, def.Namespace), opp.CreateTime)
		if err != nil {
			return *new(EntityT), err
		}
		err = repo.Witness(fmt.Sprintf(editClockPattern, def.Namespace), opp.EditTime)
		if err != nil {
			return *new(EntityT), err
		}
	}

	// Now that we know that the topological order and clocks are fine, we order the operationPacks
	// based on the logical clocks, entirely ignoring the DAG topology

	oppSlice := make([]*operationPack, 0, len(oppMap))
	for _, pack := range oppMap {
		oppSlice = append(oppSlice, pack)
	}
	sort.Slice(oppSlice, func(i, j int) bool {
		// Primary ordering with the EditTime.
		if oppSlice[i].EditTime != oppSlice[j].EditTime {
			return oppSlice[i].EditTime < oppSlice[j].EditTime
		}
		// We have equal EditTime, which means we have concurrent edition over different machines, and we
		// can't tell which one came first. So, what now? We still need a total ordering and the most stable possible.
		// As a secondary ordering, we can order based on a hash of the serialized Operations in the
		// operationPack. It doesn't carry much meaning but it's unbiased and hard to abuse.
		// This is a lexicographic ordering on the stringified ID.
		return oppSlice[i].Id() < oppSlice[j].Id()
	})

	// Now that we ordered the operationPacks, we have the order of the Operations

	ops := make([]Operation, 0, opsCount)
	var createTime lamport.Time
	var editTime lamport.Time
	for _, pack := range oppSlice {
		for _, operation := range pack.Operations {
			ops = append(ops, operation)
		}
		if pack.CreateTime > createTime {
			createTime = pack.CreateTime
		}
		if pack.EditTime > editTime {
			editTime = pack.EditTime
		}
	}

	return wrapper(&Entity{
		Definition: def,
		ops:        ops,
		lastCommit: rootHash,
		createTime: createTime,
		editTime:   editTime,
	}), nil
}

// readClockNoCheck fetch from git, read and witness the clocks of an Entity at an arbitrary git reference.
// Note: readClockNoCheck does not verify the integrity of the Entity and could witness incorrect or incomplete
// clocks if so. If data integrity check is a requirement, a flow similar to read without actually reading/decoding
// operation blobs can be implemented instead.
func readClockNoCheck(def Definition, repo repository.ClockedRepo, ref string) error {
	rootHash, err := repo.ResolveRef(ref)
	if err == repository.ErrNotFound {
		return entity.NewErrNotFound(def.Typename)
	}
	if err != nil {
		return err
	}

	commit, err := repo.ReadCommit(rootHash)
	if err != nil {
		return err
	}

	createTime, editTime, err := readOperationPackClock(repo, commit)
	if err != nil {
		return err
	}

	// if we have more than one commit, we need to find the root to have the create time
	if len(commit.Parents) > 0 {
		for len(commit.Parents) > 0 {
			// The path to the root is irrelevant.
			commit, err = repo.ReadCommit(commit.Parents[0])
			if err != nil {
				return err
			}
		}
		createTime, _, err = readOperationPackClock(repo, commit)
		if err != nil {
			return err
		}
	}

	if createTime <= 0 {
		return fmt.Errorf("creation lamport time not set")
	}
	if editTime <= 0 {
		return fmt.Errorf("creation lamport time not set")
	}
	err = repo.Witness(fmt.Sprintf(creationClockPattern, def.Namespace), createTime)
	if err != nil {
		return err
	}
	err = repo.Witness(fmt.Sprintf(editClockPattern, def.Namespace), editTime)
	if err != nil {
		return err
	}
	return nil
}

// ReadAll read and parse all local Entity
func ReadAll[EntityT entity.Bare](def Definition, wrapper func(e *Entity) EntityT, repo repository.ClockedRepo, resolvers entity.Resolvers) <-chan bootstrap.StreamedEntity[EntityT] {
	out := make(chan bootstrap.StreamedEntity[EntityT])

	go func() {
		defer close(out)

		refPrefix := fmt.Sprintf("refs/%s/", def.Namespace)

		refs, err := repo.ListRefs(refPrefix)
		if err != nil {
			out <- bootstrap.StreamedEntity[EntityT]{Err: err}
			return
		}

		total := int64(len(refs))
		current := int64(1)

		for _, ref := range refs {
			e, err := read[EntityT](def, wrapper, repo, resolvers, ref)

			if err != nil {
				out <- bootstrap.StreamedEntity[EntityT]{Err: err}
				return
			}

			out <- bootstrap.StreamedEntity[EntityT]{
				Entity:        e,
				CurrentEntity: current,
				TotalEntities: total,
			}
			current++
		}
	}()

	return out
}

// ReadAllClocksNoCheck goes over all entities matching Definition and read/witness the corresponding clocks so that the
// repo end up with correct clocks for the next write.
func ReadAllClocksNoCheck(def Definition, repo repository.ClockedRepo) error {
	refPrefix := fmt.Sprintf("refs/%s/", def.Namespace)

	refs, err := repo.ListRefs(refPrefix)
	if err != nil {
		return err
	}

	for _, ref := range refs {
		err = readClockNoCheck(def, repo, ref)
		if err != nil {
			return err
		}
	}

	return nil
}

// Id return the Entity identifier
func (e *Entity) Id() entity.Id {
	// id is the id of the first operation
	return e.FirstOp().Id()
}

// Validate check if the Entity data is valid
func (e *Entity) Validate() error {
	// non-empty
	if len(e.ops) == 0 && len(e.staging) == 0 {
		return fmt.Errorf("entity has no operations")
	}

	// check if each operation are valid
	for _, op := range e.ops {
		if err := op.Validate(); err != nil {
			return err
		}
	}

	// check if staging is valid if needed
	for _, op := range e.staging {
		if err := op.Validate(); err != nil {
			return err
		}
	}

	// Check that there is no colliding operation's ID
	ids := make(map[entity.Id]struct{})
	for _, op := range e.Operations() {
		if _, ok := ids[op.Id()]; ok {
			return fmt.Errorf("id collision: %s", op.Id())
		}
		ids[op.Id()] = struct{}{}
	}

	return nil
}

// Operations return the ordered operations
func (e *Entity) Operations() []Operation {
	return append(e.ops, e.staging...)
}

// FirstOp lookup for the very first operation of the Entity
func (e *Entity) FirstOp() Operation {
	for _, op := range e.ops {
		return op
	}
	for _, op := range e.staging {
		return op
	}
	return nil
}

// LastOp lookup for the very last operation of the Entity
func (e *Entity) LastOp() Operation {
	if len(e.staging) > 0 {
		return e.staging[len(e.staging)-1]
	}
	if len(e.ops) > 0 {
		return e.ops[len(e.ops)-1]
	}
	return nil
}

// Append add a new Operation to the Entity
func (e *Entity) Append(op Operation) {
	e.staging = append(e.staging, op)
}

// NeedCommit indicate if the in-memory state changed and need to be commit in the repository
func (e *Entity) NeedCommit() bool {
	return len(e.staging) > 0
}

// CommitAsNeeded execute a Commit only if necessary. This function is useful to avoid getting an error if the Entity
// is already in sync with the repository.
func (e *Entity) CommitAsNeeded(repo repository.ClockedRepo) error {
	if e.NeedCommit() {
		return e.Commit(repo)
	}
	return nil
}

// Commit write the appended operations in the repository
func (e *Entity) Commit(repo repository.ClockedRepo) error {
	if !e.NeedCommit() {
		return fmt.Errorf("can't commit an entity with no pending operation")
	}

	err := e.Validate()
	if err != nil {
		return errors.Wrapf(err, "can't commit a %s with invalid data", e.Definition.Typename)
	}

	for len(e.staging) > 0 {
		var author identity.Interface
		var toCommit []Operation

		// Split into chunks with the same author
		for len(e.staging) > 0 {
			op := e.staging[0]
			if author != nil && op.Author().Id() != author.Id() {
				break
			}
			author = e.staging[0].Author()
			toCommit = append(toCommit, op)
			e.staging = e.staging[1:]
		}

		e.editTime, err = repo.Increment(fmt.Sprintf(editClockPattern, e.Namespace))
		if err != nil {
			return err
		}

		opp := &operationPack{
			Author:     author,
			Operations: toCommit,
			EditTime:   e.editTime,
		}

		if e.lastCommit == "" {
			e.createTime, err = repo.Increment(fmt.Sprintf(creationClockPattern, e.Namespace))
			if err != nil {
				return err
			}
			opp.CreateTime = e.createTime
		}

		var parentCommit []repository.Hash
		if e.lastCommit != "" {
			parentCommit = []repository.Hash{e.lastCommit}
		}

		commitHash, err := opp.Write(e.Definition, repo, parentCommit...)
		if err != nil {
			return err
		}

		e.lastCommit = commitHash
		e.ops = append(e.ops, toCommit...)
	}

	// not strictly necessary but make equality testing easier in tests
	e.staging = nil

	// Create or update the Git reference for this entity
	// When pushing later, the remote will ensure that this ref update
	// is fast-forward, that is no data has been overwritten.
	ref := fmt.Sprintf(refsPattern, e.Namespace, e.Id().String())
	return repo.UpdateRef(ref, e.lastCommit)
}

// CreateLamportTime return the Lamport time of creation
func (e *Entity) CreateLamportTime() lamport.Time {
	return e.createTime
}

// EditLamportTime return the Lamport time of the last edition
func (e *Entity) EditLamportTime() lamport.Time {
	return e.editTime
}
