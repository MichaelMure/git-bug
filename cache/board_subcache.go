package cache

import (
	"time"

	"github.com/MichaelMure/git-bug/entities/board"
	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

type RepoCacheBoard struct {
	*SubCache[*board.Board, *BoardExcerpt, *BoardCache]
}

func NewRepoCacheBoard(repo repository.ClockedRepo,
	resolvers func() entity.Resolvers,
	getUserIdentity getUserIdentityFunc) *RepoCacheBoard {

	makeCached := func(b *board.Board, entityUpdated func(id entity.Id) error) *BoardCache {
		return NewBoardCache(b, repo, getUserIdentity, entityUpdated)
	}

	makeIndexData := func(b *BoardCache) []string {
		// no indexing
		return nil
	}

	actions := Actions[*board.Board]{
		ReadWithResolver:    board.ReadWithResolver,
		ReadAllWithResolver: board.ReadAllWithResolver,
		Remove:              board.Remove,
		MergeAll:            board.MergeAll,
	}

	sc := NewSubCache[*board.Board, *BoardExcerpt, *BoardCache](
		repo, resolvers, getUserIdentity,
		makeCached, NewBoardExcerpt, makeIndexData, actions,
		board.Typename, board.Namespace,
		formatVersion, defaultMaxLoadedBugs,
	)

	return &RepoCacheBoard{SubCache: sc}
}

func (c *RepoCacheBoard) New(title, description string, columns []string) (*BoardCache, *board.CreateOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, nil, err
	}

	return c.NewRaw(author, time.Now().Unix(), title, description, columns, nil)
}

func (c *RepoCacheBoard) NewDefaultColumns(title, description string) (*BoardCache, *board.CreateOperation, error) {
	return c.New(title, description, board.DefaultColumns)
}

// NewRaw create a new board with the given title, description and columns.
// The new board is written in the repository (commit).
func (c *RepoCacheBoard) NewRaw(author identity.Interface, unixTime int64, title, description string, columns []string, metadata map[string]string) (*BoardCache, *board.CreateOperation, error) {
	b, op, err := board.Create(author, unixTime, title, description, columns, metadata)
	if err != nil {
		return nil, nil, err
	}

	err = b.Commit(c.repo)
	if err != nil {
		return nil, nil, err
	}

	cached, err := c.add(b)
	if err != nil {
		return nil, nil, err
	}

	return cached, op, nil
}
