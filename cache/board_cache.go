package cache

import (
	"time"

	"github.com/MichaelMure/git-bug/entities/board"
	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

// BoardCache is a wrapper around a Board. It provides multiple functions:
//
// 1. Provide a higher level API to use than the raw API from Board.
// 2. Maintain an up-to-date Snapshot available.
// 3. Deal with concurrency.
type BoardCache struct {
	CachedEntityBase[*board.Snapshot, board.Operation]
}

func NewBoardCache(b *board.Board, repo repository.ClockedRepo, getUserIdentity getUserIdentityFunc, entityUpdated func(id entity.Id) error) *BoardCache {
	return &BoardCache{
		CachedEntityBase: CachedEntityBase[*board.Snapshot, board.Operation]{
			repo:            repo,
			entityUpdated:   entityUpdated,
			getUserIdentity: getUserIdentity,
			entity:          &withSnapshot[*board.Snapshot, board.Operation]{Interface: b},
		},
	}
}

func (c *BoardCache) AddItemDraft(columnId entity.CombinedId, title, message string, files []repository.Hash) (entity.CombinedId, *board.AddItemDraftOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}

	return c.AddItemDraftRaw(author, time.Now().Unix(), columnId, title, message, files, nil)
}

func (c *BoardCache) AddItemDraftRaw(author identity.Interface, unixTime int64, columnId entity.CombinedId, title, message string, files []repository.Hash, metadata map[string]string) (entity.CombinedId, *board.AddItemDraftOperation, error) {
	column, err := c.Snapshot().SearchColumn(columnId)
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}

	c.mu.Lock()
	itemId, op, err := board.AddItemDraft(c.entity, author, unixTime, column.Id, title, message, files, metadata)
	c.mu.Unlock()
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}
	return itemId, op, c.notifyUpdated()
}

func (c *BoardCache) AddItemEntity(columnId entity.CombinedId, e entity.Interface) (entity.CombinedId, *board.AddItemEntityOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}

	return c.AddItemEntityRaw(author, time.Now().Unix(), columnId, e, nil)
}

func (c *BoardCache) AddItemEntityRaw(author identity.Interface, unixTime int64, columnId entity.CombinedId, e entity.Interface, metadata map[string]string) (entity.CombinedId, *board.AddItemEntityOperation, error) {
	column, err := c.Snapshot().SearchColumn(columnId)
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}

	var entityType board.ItemEntityType
	switch e.(type) {
	case *BugCache:
		entityType = board.EntityTypeBug
	default:
		panic("unknown entity type")
	}

	c.mu.Lock()
	itemId, op, err := board.AddItemEntity(c.entity, author, unixTime, column.Id, entityType, e, metadata)
	c.mu.Unlock()
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}
	return itemId, op, c.notifyUpdated()
}

func (c *BoardCache) SetDescription(description string) (*board.SetDescriptionOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.SetDescriptionRaw(author, time.Now().Unix(), description, nil)
}

func (c *BoardCache) SetDescriptionRaw(author identity.Interface, unixTime int64, description string, metadata map[string]string) (*board.SetDescriptionOperation, error) {
	c.mu.Lock()
	op, err := board.SetDescription(c.entity, author, unixTime, description, metadata)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return op, c.notifyUpdated()
}

func (c *BoardCache) SetTitle(title string) (*board.SetTitleOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.SetTitleRaw(author, time.Now().Unix(), title, nil)
}

func (c *BoardCache) SetTitleRaw(author identity.Interface, unixTime int64, title string, metadata map[string]string) (*board.SetTitleOperation, error) {
	c.mu.Lock()
	op, err := board.SetTitle(c.entity, author, unixTime, title, metadata)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return op, c.notifyUpdated()
}
