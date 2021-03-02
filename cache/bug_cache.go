package cache

import (
	"fmt"
	"sync"
	"time"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

var ErrNoMatchingOp = fmt.Errorf("no matching operation found")

// BugCache is a wrapper around a Bug. It provide multiple functions:
//
// 1. Provide a higher level API to use than the raw API from Bug.
// 2. Maintain an up to date Snapshot available.
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
	return c.bug.Snapshot()
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

	it := bug.NewOperationIterator(c.bug)
	for it.Next() {
		op := it.Value()
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

func (c *BugCache) AddComment(message string) (*bug.AddCommentOperation, error) {
	return c.AddCommentWithFiles(message, nil)
}

func (c *BugCache) AddCommentWithFiles(message string, files []repository.Hash) (*bug.AddCommentOperation, error) {
	return c.AddCommentWithFilesAndTime(time.Now().Unix(), message, files)
}

func (c *BugCache) AddCommentWithFilesAndTime(unixTime int64, message string, files []repository.Hash) (*bug.AddCommentOperation, error) {
	author, err := c.repoCache.GetUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.AddCommentRaw(author, unixTime, message, files, nil)
}

func (c *BugCache) AddCommentRaw(author *IdentityCache, unixTime int64, message string, files []repository.Hash, metadata map[string]string) (*bug.AddCommentOperation, error) {
	c.mu.Lock()
	op, err := bug.AddCommentWithFiles(c.bug, author.Identity, unixTime, message, files)
	if err != nil {
		c.mu.Unlock()
		return nil, err
	}

	for key, value := range metadata {
		op.SetMetadata(key, value)
	}

	c.mu.Unlock()

	return op, c.notifyUpdated()
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
	changes, op, err := bug.ChangeLabels(c.bug, author.Identity, unixTime, added, removed)
	if err != nil {
		c.mu.Unlock()
		return changes, nil, err
	}

	for key, value := range metadata {
		op.SetMetadata(key, value)
	}

	c.mu.Unlock()

	err = c.notifyUpdated()
	if err != nil {
		return nil, nil, err
	}

	return changes, op, nil
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
	op, err := bug.ForceChangeLabels(c.bug, author.Identity, unixTime, added, removed)
	if err != nil {
		c.mu.Unlock()
		return nil, err
	}

	for key, value := range metadata {
		op.SetMetadata(key, value)
	}

	c.mu.Unlock()
	err = c.notifyUpdated()
	if err != nil {
		return nil, err
	}

	return op, nil
}

func (c *BugCache) Open() (*bug.SetStatusOperation, error) {
	return c.OpenWithTime(time.Now().Unix())
}

func (c *BugCache) OpenWithTime(unixTime int64) (*bug.SetStatusOperation, error) {
	author, err := c.repoCache.GetUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.OpenRaw(author, unixTime, nil)
}

func (c *BugCache) OpenRaw(author *IdentityCache, unixTime int64, metadata map[string]string) (*bug.SetStatusOperation, error) {
	c.mu.Lock()
	op, err := bug.Open(c.bug, author.Identity, unixTime)
	if err != nil {
		c.mu.Unlock()
		return nil, err
	}

	for key, value := range metadata {
		op.SetMetadata(key, value)
	}

	c.mu.Unlock()
	return op, c.notifyUpdated()
}

func (c *BugCache) Close() (*bug.SetStatusOperation, error) {
	return c.CloseWithTime(time.Now().Unix())
}

func (c *BugCache) CloseWithTime(unixTime int64) (*bug.SetStatusOperation, error) {
	author, err := c.repoCache.GetUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.CloseRaw(author, unixTime, nil)
}

func (c *BugCache) CloseRaw(author *IdentityCache, unixTime int64, metadata map[string]string) (*bug.SetStatusOperation, error) {
	c.mu.Lock()
	op, err := bug.Close(c.bug, author.Identity, unixTime)
	if err != nil {
		c.mu.Unlock()
		return nil, err
	}

	for key, value := range metadata {
		op.SetMetadata(key, value)
	}

	c.mu.Unlock()
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
	op, err := bug.SetTitle(c.bug, author.Identity, unixTime, title)
	if err != nil {
		c.mu.Unlock()
		return nil, err
	}

	for key, value := range metadata {
		op.SetMetadata(key, value)
	}

	c.mu.Unlock()
	return op, c.notifyUpdated()
}

func (c *BugCache) EditCreateComment(body string) (*bug.EditCommentOperation, error) {
	author, err := c.repoCache.GetUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.EditCreateCommentRaw(author, time.Now().Unix(), body, nil)
}

func (c *BugCache) EditCreateCommentRaw(author *IdentityCache, unixTime int64, body string, metadata map[string]string) (*bug.EditCommentOperation, error) {
	c.mu.Lock()
	op, err := bug.EditCreateComment(c.bug, author.Identity, unixTime, body)
	if err != nil {
		c.mu.Unlock()
		return nil, err
	}

	for key, value := range metadata {
		op.SetMetadata(key, value)
	}

	c.mu.Unlock()
	return op, c.notifyUpdated()
}

func (c *BugCache) EditComment(target entity.Id, message string) (*bug.EditCommentOperation, error) {
	author, err := c.repoCache.GetUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.EditCommentRaw(author, time.Now().Unix(), target, message, nil)
}

func (c *BugCache) EditCommentRaw(author *IdentityCache, unixTime int64, target entity.Id, message string, metadata map[string]string) (*bug.EditCommentOperation, error) {
	c.mu.Lock()
	op, err := bug.EditComment(c.bug, author.Identity, unixTime, target, message)
	if err != nil {
		c.mu.Unlock()
		return nil, err
	}

	for key, value := range metadata {
		op.SetMetadata(key, value)
	}

	c.mu.Unlock()
	return op, c.notifyUpdated()
}

func (c *BugCache) SetMetadata(target entity.Id, newMetadata map[string]string) (*bug.SetMetadataOperation, error) {
	author, err := c.repoCache.GetUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.SetMetadataRaw(author, time.Now().Unix(), target, newMetadata)
}

func (c *BugCache) SetMetadataRaw(author *IdentityCache, unixTime int64, target entity.Id, newMetadata map[string]string) (*bug.SetMetadataOperation, error) {
	c.mu.Lock()
	op, err := bug.SetMetadata(c.bug, author.Identity, unixTime, target, newMetadata)
	if err != nil {
		c.mu.Unlock()
		return nil, err
	}

	c.mu.Unlock()
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
