// Package bug contains the bug data model and low-level related functions
package bug

import (
	"fmt"

	"github.com/MichaelMure/git-bug/entities/common"
	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	bootstrap "github.com/MichaelMure/git-bug/entity/boostrap"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/repository"
)

var _ Interface = &Bug{}
var _ entity.Interface[Operation, *Snapshot] = &Bug{}

// 1: original format
// 2: no more legacy identities
// 3: Ids are generated from the create operation serialized data instead of from the first git commit
// 4: with DAG entity framework
const formatVersion = 4

const Typename = "bug"
const Namespace = "bugs"

var def = dag.Definition{
	Typename:             Typename,
	Namespace:            Namespace,
	OperationUnmarshaler: operationUnmarshaler,
	FormatVersion:        formatVersion,
}

var ClockLoader = dag.ClockLoader(def)

type Interface interface {
	entity.Interface[Operation, *Snapshot]
}

// Bug holds the data of a bug thread, organized in a way close to
// how it will be persisted inside Git. This is the data structure
// used to merge two different version of the same Bug.
type Bug struct {
	*dag.Entity
}

// NewBug create a new Bug
func NewBug() *Bug {
	return wrapper(dag.New(def))
}

func wrapper(e *dag.Entity) *Bug {
	return &Bug{Entity: e}
}

func simpleResolvers(repo repository.ClockedRepo) entity.Resolvers {
	return entity.Resolvers{
		&identity.Identity{}: identity.NewSimpleResolver(repo),
	}
}

// Read will read a bug from a repository
func Read(repo repository.ClockedRepo, id entity.Id) (*Bug, error) {
	return ReadWithResolver(repo, simpleResolvers(repo), id)
}

// ReadWithResolver will read a bug from its Id, with custom resolvers
func ReadWithResolver(repo repository.ClockedRepo, resolvers entity.Resolvers, id entity.Id) (*Bug, error) {
	return dag.Read(def, wrapper, repo, resolvers, id)
}

// ReadAll read and parse all local bugs
func ReadAll(repo repository.ClockedRepo) <-chan bootstrap.StreamedEntity[*Bug] {
	return dag.ReadAll(def, wrapper, repo, simpleResolvers(repo))
}

// ReadAllWithResolver read and parse all local bugs
func ReadAllWithResolver(repo repository.ClockedRepo, resolvers entity.Resolvers) <-chan bootstrap.StreamedEntity[*Bug] {
	return dag.ReadAll(def, wrapper, repo, resolvers)
}

// ListLocalIds list all the available local bug ids
func ListLocalIds(repo repository.Repo) ([]entity.Id, error) {
	return dag.ListLocalIds(def, repo)
}

// Validate check if the Bug data is valid
func (bug *Bug) Validate() error {
	if err := bug.Entity.Validate(); err != nil {
		return err
	}

	// The very first Op should be a CreateOp
	firstOp := bug.FirstOp()
	if firstOp == nil || firstOp.Type() != CreateOp {
		return fmt.Errorf("first operation should be a Create op")
	}

	// Check that there is no more CreateOp op
	for i, op := range bug.Entity.Operations() {
		if i == 0 {
			continue
		}
		if op.Type() == CreateOp {
			return fmt.Errorf("only one Create op allowed")
		}
	}

	return nil
}

// Append add a new Operation to the Bug
func (bug *Bug) Append(op Operation) {
	bug.Entity.Append(op)
}

// Operations return the ordered operations
func (bug *Bug) Operations() []Operation {
	source := bug.Entity.Operations()
	result := make([]Operation, len(source))
	for i, op := range source {
		result[i] = op.(Operation)
	}
	return result
}

// Compile a bug in an easily usable snapshot
func (bug *Bug) Compile() *Snapshot {
	snap := &Snapshot{
		id:     bug.Id(),
		Status: common.OpenStatus,
	}

	for _, op := range bug.Operations() {
		op.Apply(snap)
		snap.Operations = append(snap.Operations, op)
	}

	return snap
}

// FirstOp lookup for the very first operation of the bug.
// For a valid Bug, this operation should be a CreateOp
func (bug *Bug) FirstOp() Operation {
	if fo := bug.Entity.FirstOp(); fo != nil {
		return fo.(Operation)
	}
	return nil
}

// LastOp lookup for the very last operation of the bug.
// For a valid Bug, should never be nil
func (bug *Bug) LastOp() Operation {
	if lo := bug.Entity.LastOp(); lo != nil {
		return lo.(Operation)
	}
	return nil
}
