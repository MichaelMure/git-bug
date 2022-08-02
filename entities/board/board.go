package board

import (
	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/entities/identity"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/repository"
)

var _ entity.Interface = &Board{}

// 1: original format
const formatVersion = 1

var def = dag.Definition{
	Typename:             "board",
	Namespace:            "boards",
	OperationUnmarshaler: operationUnmarshaller,
	FormatVersion:        formatVersion,
}

var ClockLoader = dag.ClockLoader(def)

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
	e, err := dag.Read(def, repo, resolvers, id)
	if err != nil {
		return nil, err
	}
	return &Board{Entity: e}, nil
}
