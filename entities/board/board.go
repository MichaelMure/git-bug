package board

import (
	"fmt"

	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/entities/identity"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/repository"
)

var _ Interface = &Board{}

// 1: original format
const formatVersion = 1

const Typename = "board"
const Namespace = "boards"

var def = dag.Definition{
	Typename:             Typename,
	Namespace:            Namespace,
	OperationUnmarshaler: operationUnmarshaler,
	FormatVersion:        formatVersion,
}

var ClockLoader = dag.ClockLoader(def)

type Interface interface {
	dag.Interface[*Snapshot, Operation]
}

// Board holds the data of a project board.
type Board struct {
	*dag.Entity
}

// NewBoard create a new Board
func NewBoard() *Board {
	return &Board{
		Entity: dag.New(def),
	}
}

func wrapper(e *dag.Entity) *Board {
	return &Board{Entity: e}
}

func simpleResolvers(repo repository.ClockedRepo) entity.Resolvers {
	return entity.Resolvers{
		&identity.Identity{}: identity.NewSimpleResolver(repo),
		&bug.Bug{}:           bug.NewSimpleResolver(repo),
	}
}

// Read will read a board from a repository
func Read(repo repository.ClockedRepo, id entity.Id) (*Board, error) {
	return ReadWithResolver(repo, simpleResolvers(repo), id)
}

// ReadWithResolver will read a board from its Id, with a custom identity.Resolver
func ReadWithResolver(repo repository.ClockedRepo, resolvers entity.Resolvers, id entity.Id) (*Board, error) {
	return dag.Read(def, wrapper, repo, resolvers, id)
}

// ReadAll read and parse all local boards
func ReadAll(repo repository.ClockedRepo) <-chan entity.StreamedEntity[*Board] {
	return dag.ReadAll(def, wrapper, repo, simpleResolvers(repo))
}

// ReadAllWithResolver read and parse all local boards
func ReadAllWithResolver(repo repository.ClockedRepo, resolvers entity.Resolvers) <-chan entity.StreamedEntity[*Board] {
	return dag.ReadAll(def, wrapper, repo, resolvers)
}

// Validate check if the Board data is valid
func (board *Board) Validate() error {
	if err := board.Entity.Validate(); err != nil {
		return err
	}

	// The very first Op should be a CreateOp
	firstOp := board.FirstOp()
	if firstOp == nil || firstOp.Type() != CreateOp {
		return fmt.Errorf("first operation should be a Create op")
	}

	// Check that there is no more CreateOp op
	for i, op := range board.Entity.Operations() {
		if i == 0 {
			continue
		}
		if op.Type() == CreateOp {
			return fmt.Errorf("only one Create op allowed")
		}
	}

	return nil
}

// Append add a new Operation to the Board
func (board *Board) Append(op Operation) {
	board.Entity.Append(op)
}

// Operations return the ordered operations
func (board *Board) Operations() []Operation {
	source := board.Entity.Operations()
	result := make([]Operation, len(source))
	for i, op := range source {
		result[i] = op.(Operation)
	}
	return result
}

// Compile a board in an easily usable snapshot
func (board *Board) Compile() *Snapshot {
	snap := &Snapshot{
		id: board.Id(),
	}

	for _, op := range board.Operations() {
		op.Apply(snap)
		snap.Operations = append(snap.Operations, op)
	}

	return snap
}

// FirstOp lookup for the very first operation of the board.
// For a valid Board, this operation should be a CreateOp
func (board *Board) FirstOp() Operation {
	if fo := board.Entity.FirstOp(); fo != nil {
		return fo.(Operation)
	}
	return nil
}

// LastOp lookup for the very last operation of the board.
// For a valid Board, should never be nil
func (board *Board) LastOp() Operation {
	if lo := board.Entity.LastOp(); lo != nil {
		return lo.(Operation)
	}
	return nil
}
