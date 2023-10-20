package cache

import (
	"fmt"
	"time"

	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/repository"
)

var ErrNoMatchingOp = fmt.Errorf("no matching operation found")

var _ bug.Interface = &BugCache{}

// BugCache is a wrapper around a Bug. It provides multiple functions:
//
// 1. Provide a higher level API to use than the raw API from Bug.
// 2. Maintain an up-to-date Snapshot available.
// 3. Deal with concurrency.
type BugCache struct {
	CachedEntityBase[*bug.Snapshot, bug.Operation]
}

func NewBugCache(b *bug.Bug, repo repository.ClockedRepo, getUserIdentity getUserIdentityFunc, entityUpdated func(id entity.Id) error) *BugCache {
	return &BugCache{
		CachedEntityBase: CachedEntityBase[*bug.Snapshot, bug.Operation]{
			repo:            repo,
			entityUpdated:   entityUpdated,
			getUserIdentity: getUserIdentity,
			entity:          &withSnapshot[*bug.Snapshot, bug.Operation]{Interface: b},
		},
	}
}

func (c *BugCache) AddComment(message string) (entity.CombinedId, *bug.AddCommentOperation, error) {
	return c.AddCommentWithFiles(message, nil)
}

func (c *BugCache) AddCommentWithFiles(message string, files []repository.Hash) (entity.CombinedId, *bug.AddCommentOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}

	return c.AddCommentRaw(author, time.Now().Unix(), message, files, nil)
}

func (c *BugCache) AddCommentRaw(author entity.Interface, unixTime int64, message string, files []repository.Hash, metadata map[string]string) (entity.CombinedId, *bug.AddCommentOperation, error) {
	c.mu.Lock()
	commentId, op, err := bug.AddComment(c.entity, author, unixTime, message, files, metadata)
	c.mu.Unlock()
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}
	return commentId, op, c.notifyUpdated()
}

func (c *BugCache) ChangeLabels(added []string, removed []string) ([]bug.LabelChangeResult, *bug.LabelChangeOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, nil, err
	}

	return c.ChangeLabelsRaw(author, time.Now().Unix(), added, removed, nil)
}

func (c *BugCache) ChangeLabelsRaw(author entity.Interface, unixTime int64, added []string, removed []string, metadata map[string]string) ([]bug.LabelChangeResult, *bug.LabelChangeOperation, error) {
	c.mu.Lock()
	changes, op, err := bug.ChangeLabels(c.entity, author, unixTime, added, removed, metadata)
	c.mu.Unlock()
	if err != nil {
		return changes, nil, err
	}
	return changes, op, c.notifyUpdated()
}

func (c *BugCache) ForceChangeLabels(added []string, removed []string) (*bug.LabelChangeOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.ForceChangeLabelsRaw(author, time.Now().Unix(), added, removed, nil)
}

func (c *BugCache) ForceChangeLabelsRaw(author entity.Interface, unixTime int64, added []string, removed []string, metadata map[string]string) (*bug.LabelChangeOperation, error) {
	c.mu.Lock()
	op, err := bug.ForceChangeLabels(c.entity, author, unixTime, added, removed, metadata)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return op, c.notifyUpdated()
}

func (c *BugCache) Open() (*bug.SetStatusOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.OpenRaw(author, time.Now().Unix(), nil)
}

func (c *BugCache) OpenRaw(author entity.Interface, unixTime int64, metadata map[string]string) (*bug.SetStatusOperation, error) {
	c.mu.Lock()
	op, err := bug.Open(c.entity, author, unixTime, metadata)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return op, c.notifyUpdated()
}

func (c *BugCache) Close() (*bug.SetStatusOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.CloseRaw(author, time.Now().Unix(), nil)
}

func (c *BugCache) CloseRaw(author entity.Interface, unixTime int64, metadata map[string]string) (*bug.SetStatusOperation, error) {
	c.mu.Lock()
	op, err := bug.Close(c.entity, author, unixTime, metadata)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return op, c.notifyUpdated()
}

func (c *BugCache) SetTitle(title string) (*bug.SetTitleOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.SetTitleRaw(author, time.Now().Unix(), title, nil)
}

func (c *BugCache) SetTitleRaw(author entity.Interface, unixTime int64, title string, metadata map[string]string) (*bug.SetTitleOperation, error) {
	c.mu.Lock()
	op, err := bug.SetTitle(c.entity, author, unixTime, title, metadata)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return op, c.notifyUpdated()
}

// EditCreateComment is a convenience function to edit the body of a bug (the first comment)
func (c *BugCache) EditCreateComment(body string) (entity.CombinedId, *bug.EditCommentOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}

	return c.EditCreateCommentRaw(author, time.Now().Unix(), body, nil)
}

// EditCreateCommentRaw is a convenience function to edit the body of a bug (the first comment)
func (c *BugCache) EditCreateCommentRaw(author entity.Interface, unixTime int64, body string, metadata map[string]string) (entity.CombinedId, *bug.EditCommentOperation, error) {
	c.mu.Lock()
	commentId, op, err := bug.EditCreateComment(c.entity, author, unixTime, body, nil, metadata)
	c.mu.Unlock()
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}
	return commentId, op, c.notifyUpdated()
}

func (c *BugCache) EditComment(target entity.CombinedId, message string) (*bug.EditCommentOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.EditCommentRaw(author, time.Now().Unix(), target, message, nil)
}

func (c *BugCache) EditCommentRaw(author entity.Interface, unixTime int64, target entity.CombinedId, message string, metadata map[string]string) (*bug.EditCommentOperation, error) {
	comment, err := c.Compile().SearchComment(target)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	commentId, op, err := bug.EditComment(c.entity, author, unixTime, comment.TargetId(), message, nil, metadata)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if commentId != target {
		panic("EditComment returned unexpected comment id")
	}
	return op, c.notifyUpdated()
}

func (c *BugCache) SetMetadata(target entity.Id, newMetadata map[string]string) (*dag.SetMetadataOperation[*bug.Snapshot], error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.SetMetadataRaw(author, time.Now().Unix(), target, newMetadata)
}

func (c *BugCache) SetMetadataRaw(author entity.Interface, unixTime int64, target entity.Id, newMetadata map[string]string) (*dag.SetMetadataOperation[*bug.Snapshot], error) {
	c.mu.Lock()
	op, err := bug.SetMetadata(c.entity, author, unixTime, target, newMetadata)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return op, c.notifyUpdated()
}
