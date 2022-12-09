package cache

import (
	"fmt"
	"sync"
	"time"

	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/repository"
)

var ErrNoMatchingOp = fmt.Errorf("no matching operation found")

// BugCache is a wrapper around a Bug. It provides multiple functions:
//
// 1. Provide a higher level API to use than the raw API from Bug.
// 2. Maintain an up-to-date Snapshot available.
// 3. Deal with concurrency.
type BugCache struct {
	repoCache *RepoCache
	mu        sync.RWMutex
	bug       *bug.WithSnapshot
}

func NewBugCache(repoCache *RepoCache, b *bug.Bug) *BugCache {
	return &BugCache{
		repoCache: repoCache,
		bug:       &bug.WithSnapshot{Bug: b},
	}
}

func (c *BugCache) Snapshot() *bug.Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.bug.Compile()
}

func (c *BugCache) Id() entity.Id {
	return c.bug.Id()
}

func (c *BugCache) notifyUpdated() error {
	return c.repoCache.bugUpdated(c.bug.Id())
}

// ResolveOperationWithMetadata will find an operation that has the matching metadata
func (c *BugCache) ResolveOperationWithMetadata(key string, value string) (entity.Id, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// preallocate but empty
	matching := make([]entity.Id, 0, 5)

	for _, op := range c.bug.Operations() {
		opValue, ok := op.GetMetadata(key)
		if ok && value == opValue {
			matching = append(matching, op.Id())
		}
	}

	if len(matching) == 0 {
		return "", ErrNoMatchingOp
	}

	if len(matching) > 1 {
		return "", bug.NewErrMultipleMatchOp(matching)
	}

	return matching[0], nil
}

func (c *BugCache) AddComment(message string) (entity.CombinedId, *bug.AddCommentOperation, error) {
	return c.AddCommentWithFiles(message, nil)
}

func (c *BugCache) AddCommentWithFiles(message string, files []repository.Hash) (entity.CombinedId, *bug.AddCommentOperation, error) {
	author, err := c.repoCache.GetUserIdentity()
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}

	return c.AddCommentRaw(author, time.Now().Unix(), message, files, nil)
}

func (c *BugCache) AddCommentRaw(author *IdentityCache, unixTime int64, message string, files []repository.Hash, metadata map[string]string) (entity.CombinedId, *bug.AddCommentOperation, error) {
	c.mu.Lock()
	commentId, op, err := bug.AddComment(c.bug, author, unixTime, message, files, metadata)
	c.mu.Unlock()
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}
	return commentId, op, c.notifyUpdated()
}

func (c *BugCache) ChangeLabels(added []string, removed []string) ([]bug.LabelChangeResult, *bug.LabelChangeOperation, error) {
	author, err := c.repoCache.GetUserIdentity()
	if err != nil {
		return nil, nil, err
	}

	return c.ChangeLabelsRaw(author, time.Now().Unix(), added, removed, nil)
}

func (c *BugCache) ChangeLabelsRaw(author *IdentityCache, unixTime int64, added []string, removed []string, metadata map[string]string) ([]bug.LabelChangeResult, *bug.LabelChangeOperation, error) {
	c.mu.Lock()
	changes, op, err := bug.ChangeLabels(c.bug, author.Identity, unixTime, added, removed, metadata)
	c.mu.Unlock()
	if err != nil {
		return changes, nil, err
	}
	return changes, op, c.notifyUpdated()
}

func (c *BugCache) ForceChangeLabels(added []string, removed []string) (*bug.LabelChangeOperation, error) {
	author, err := c.repoCache.GetUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.ForceChangeLabelsRaw(author, time.Now().Unix(), added, removed, nil)
}

func (c *BugCache) ForceChangeLabelsRaw(author *IdentityCache, unixTime int64, added []string, removed []string, metadata map[string]string) (*bug.LabelChangeOperation, error) {
	c.mu.Lock()
	op, err := bug.ForceChangeLabels(c.bug, author.Identity, unixTime, added, removed, metadata)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return op, c.notifyUpdated()
}

func (c *BugCache) Open() (*bug.SetStatusOperation, error) {
	author, err := c.repoCache.GetUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.OpenRaw(author, time.Now().Unix(), nil)
}

func (c *BugCache) OpenRaw(author *IdentityCache, unixTime int64, metadata map[string]string) (*bug.SetStatusOperation, error) {
	c.mu.Lock()
	op, err := bug.Open(c.bug, author.Identity, unixTime, metadata)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return op, c.notifyUpdated()
}

func (c *BugCache) Close() (*bug.SetStatusOperation, error) {
	author, err := c.repoCache.GetUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.CloseRaw(author, time.Now().Unix(), nil)
}

func (c *BugCache) CloseRaw(author *IdentityCache, unixTime int64, metadata map[string]string) (*bug.SetStatusOperation, error) {
	c.mu.Lock()
	op, err := bug.Close(c.bug, author.Identity, unixTime, metadata)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return op, c.notifyUpdated()
}

func (c *BugCache) SetTitle(title string) (*bug.SetTitleOperation, error) {
	author, err := c.repoCache.GetUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.SetTitleRaw(author, time.Now().Unix(), title, nil)
}

func (c *BugCache) SetTitleRaw(author *IdentityCache, unixTime int64, title string, metadata map[string]string) (*bug.SetTitleOperation, error) {
	c.mu.Lock()
	op, err := bug.SetTitle(c.bug, author.Identity, unixTime, title, metadata)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return op, c.notifyUpdated()
}

// EditCreateComment is a convenience function to edit the body of a bug (the first comment)
func (c *BugCache) EditCreateComment(body string) (entity.CombinedId, *bug.EditCommentOperation, error) {
	author, err := c.repoCache.GetUserIdentity()
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}

	return c.EditCreateCommentRaw(author, time.Now().Unix(), body, nil)
}

// EditCreateCommentRaw is a convenience function to edit the body of a bug (the first comment)
func (c *BugCache) EditCreateCommentRaw(author *IdentityCache, unixTime int64, body string, metadata map[string]string) (entity.CombinedId, *bug.EditCommentOperation, error) {
	c.mu.Lock()
	commentId, op, err := bug.EditCreateComment(c.bug, author.Identity, unixTime, body, nil, metadata)
	c.mu.Unlock()
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}
	return commentId, op, c.notifyUpdated()
}

func (c *BugCache) EditComment(target entity.CombinedId, message string) (*bug.EditCommentOperation, error) {
	author, err := c.repoCache.GetUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.EditCommentRaw(author, time.Now().Unix(), target, message, nil)
}

func (c *BugCache) EditCommentRaw(author *IdentityCache, unixTime int64, target entity.CombinedId, message string, metadata map[string]string) (*bug.EditCommentOperation, error) {
	comment, err := c.Snapshot().SearchComment(target)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	commentId, op, err := bug.EditComment(c.bug, author.Identity, unixTime, comment.TargetId(), message, nil, metadata)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if commentId != target {
		panic("EditComment returned unexpected comment id")
	}
	return op, c.notifyUpdated()
}

func (c *BugCache) DeleteComment(target entity.CombinedId) (*bug.DeleteCommentOperation, error) {
	author, err := c.repoCache.GetUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.DeleteCommentRaw(author, time.Now().Unix(), target, nil)
}

func (c *BugCache) DeleteCommentRaw(author *IdentityCache, unixTime int64, target entity.CombinedId, metadata map[string]string) (*bug.DeleteCommentOperation, error) {
	comment, err := c.Snapshot().SearchComment(target)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	cid, op, err := bug.DeleteComment(c.bug, author.Identity, unixTime, comment.TargetId(), metadata)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}

	if cid != target {
		panic("EditComment returned unexpected comment id")
	}

	return op, c.notifyUpdated()
}

func (c *BugCache) SetMetadata(target entity.Id, newMetadata map[string]string) (*dag.SetMetadataOperation[*bug.Snapshot], error) {
	author, err := c.repoCache.GetUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.SetMetadataRaw(author, time.Now().Unix(), target, newMetadata)
}

func (c *BugCache) SetMetadataRaw(author *IdentityCache, unixTime int64, target entity.Id, newMetadata map[string]string) (*dag.SetMetadataOperation[*bug.Snapshot], error) {
	c.mu.Lock()
	op, err := bug.SetMetadata(c.bug, author.Identity, unixTime, target, newMetadata)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return op, c.notifyUpdated()
}

func (c *BugCache) Commit() error {
	c.mu.Lock()
	err := c.bug.Commit(c.repoCache.repo)
	if err != nil {
		c.mu.Unlock()
		return err
	}
	c.mu.Unlock()
	return c.notifyUpdated()
}

func (c *BugCache) CommitAsNeeded() error {
	c.mu.Lock()
	err := c.bug.CommitAsNeeded(c.repoCache.repo)
	if err != nil {
		c.mu.Unlock()
		return err
	}
	c.mu.Unlock()
	return c.notifyUpdated()
}

func (c *BugCache) NeedCommit() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.bug.NeedCommit()
}
