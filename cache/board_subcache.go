package cache

import (
	"errors"
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

func (c *RepoCacheBoard) ResolveColumn(prefix string) (*BoardCache, entity.CombinedId, error) {
	boardPrefix, _ := entity.SeparateIds(prefix)
	boardCandidate := make([]entity.Id, 0, 5)

	// build a list of possible matching boards
	c.mu.RLock()
	for _, excerpt := range c.excerpts {
		if excerpt.Id().HasPrefix(boardPrefix) {
			boardCandidate = append(boardCandidate, excerpt.Id())
		}
	}
	c.mu.RUnlock()

	matchingBoardIds := make([]entity.Id, 0, 5)
	matchingColumnId := entity.UnsetCombinedId
	var matchingBoard *BoardCache

	// search for matching columns
	// searching every board candidate allow for some collision with the board prefix only,
	// before being refined with the full column prefix
	for _, boardId := range boardCandidate {
		b, err := c.Resolve(boardId)
		if err != nil {
			return nil, entity.UnsetCombinedId, err
		}

		for _, column := range b.Snapshot().Columns {
			if column.CombinedId.HasPrefix(prefix) {
				matchingBoardIds = append(matchingBoardIds, boardId)
				matchingBoard = b
				matchingColumnId = column.CombinedId
			}
		}
	}

	if len(matchingBoardIds) > 1 {
		return nil, entity.UnsetCombinedId, entity.NewErrMultipleMatch("board/column", matchingBoardIds)
	} else if len(matchingBoardIds) == 0 {
		return nil, entity.UnsetCombinedId, errors.New("column doesn't exist")
	}

	return matchingBoard, matchingColumnId, nil
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
