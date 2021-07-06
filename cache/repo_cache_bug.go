package cache

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/blevesearch/bleve"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/query"
	"github.com/MichaelMure/git-bug/repository"
)

const bugCacheFile = "bug-cache"

var errBugNotInCache = errors.New("bug missing from cache")

// bugUpdated is a callback to trigger when the excerpt of a bug changed,
// that is each time a bug is updated
func (c *RepoCache) bugUpdated(id entity.Id) error {
	c.muBug.Lock()
	b, ok := c.bugs[id]
	if !ok {
		c.muBug.Unlock()

		// if the bug is not loaded at this point, it means it was loaded before
		// but got evicted. Which means we potentially have multiple copies in
		// memory and thus concurrent write.
		// Failing immediately here is the simple and safe solution to avoid
		// complicated data loss.
		return errBugNotInCache
	}
	c.loadedBugs.Get(id)
	c.bugExcerpts[id] = NewBugExcerpt(b.bug, b.Snapshot())
	c.muBug.Unlock()

	if err := c.addBugToSearchIndex(b.Snapshot()); err != nil {
		return err
	}

	// we only need to write the bug cache
	return c.writeBugCache()
}

// load will try to read from the disk the bug cache file
func (c *RepoCache) loadBugCache() error {
	c.muBug.Lock()
	defer c.muBug.Unlock()

	f, err := c.repo.LocalStorage().Open(bugCacheFile)
	if err != nil {
		return err
	}

	decoder := gob.NewDecoder(f)

	aux := struct {
		Version  uint
		Excerpts map[entity.Id]*BugExcerpt
	}{}

	err = decoder.Decode(&aux)
	if err != nil {
		return err
	}

	if aux.Version != formatVersion {
		return fmt.Errorf("unknown cache format version %v", aux.Version)
	}

	c.bugExcerpts = aux.Excerpts

	index, err := c.repo.GetBleveIndex("bug")
	if err != nil {
		return err
	}

	// simple heuristic to detect a mismatch between the index and the bugs
	count, err := index.DocCount()
	if err != nil {
		return err
	}
	if count != uint64(len(c.bugExcerpts)) {
		return fmt.Errorf("count mismatch between bleve and bug excerpts")
	}

	return nil
}

// write will serialize on disk the bug cache file
func (c *RepoCache) writeBugCache() error {
	c.muBug.RLock()
	defer c.muBug.RUnlock()

	var data bytes.Buffer

	aux := struct {
		Version  uint
		Excerpts map[entity.Id]*BugExcerpt
	}{
		Version:  formatVersion,
		Excerpts: c.bugExcerpts,
	}

	encoder := gob.NewEncoder(&data)

	err := encoder.Encode(aux)
	if err != nil {
		return err
	}

	f, err := c.repo.LocalStorage().Create(bugCacheFile)
	if err != nil {
		return err
	}

	_, err = f.Write(data.Bytes())
	if err != nil {
		return err
	}

	return f.Close()
}

// ResolveBugExcerpt retrieve a BugExcerpt matching the exact given id
func (c *RepoCache) ResolveBugExcerpt(id entity.Id) (*BugExcerpt, error) {
	c.muBug.RLock()
	defer c.muBug.RUnlock()

	excerpt, ok := c.bugExcerpts[id]
	if !ok {
		return nil, bug.ErrBugNotExist
	}

	return excerpt, nil
}

// ResolveBug retrieve a bug matching the exact given id
func (c *RepoCache) ResolveBug(id entity.Id) (*BugCache, error) {
	c.muBug.RLock()
	cached, ok := c.bugs[id]
	if ok {
		c.loadedBugs.Get(id)
		c.muBug.RUnlock()
		return cached, nil
	}
	c.muBug.RUnlock()

	b, err := bug.ReadWithResolver(c.repo, newIdentityCacheResolver(c), id)
	if err != nil {
		return nil, err
	}

	cached = NewBugCache(c, b)

	c.muBug.Lock()
	c.bugs[id] = cached
	c.loadedBugs.Add(id)
	c.muBug.Unlock()

	c.evictIfNeeded()

	return cached, nil
}

// evictIfNeeded will evict a bug from the cache if needed
// it also removes references of the bug from the bugs
func (c *RepoCache) evictIfNeeded() {
	c.muBug.Lock()
	defer c.muBug.Unlock()
	if c.loadedBugs.Len() <= c.maxLoadedBugs {
		return
	}

	for _, id := range c.loadedBugs.GetOldestToNewest() {
		b := c.bugs[id]
		if b.NeedCommit() {
			continue
		}

		b.mu.Lock()
		c.loadedBugs.Remove(id)
		delete(c.bugs, id)

		if c.loadedBugs.Len() <= c.maxLoadedBugs {
			return
		}
	}
}

// ResolveBugExcerptPrefix retrieve a BugExcerpt matching an id prefix. It fails if multiple
// bugs match.
func (c *RepoCache) ResolveBugExcerptPrefix(prefix string) (*BugExcerpt, error) {
	return c.ResolveBugExcerptMatcher(func(excerpt *BugExcerpt) bool {
		return excerpt.Id.HasPrefix(prefix)
	})
}

// ResolveBugPrefix retrieve a bug matching an id prefix. It fails if multiple
// bugs match.
func (c *RepoCache) ResolveBugPrefix(prefix string) (*BugCache, error) {
	return c.ResolveBugMatcher(func(excerpt *BugExcerpt) bool {
		return excerpt.Id.HasPrefix(prefix)
	})
}

// ResolveBugCreateMetadata retrieve a bug that has the exact given metadata on
// its Create operation, that is, the first operation. It fails if multiple bugs
// match.
func (c *RepoCache) ResolveBugCreateMetadata(key string, value string) (*BugCache, error) {
	return c.ResolveBugMatcher(func(excerpt *BugExcerpt) bool {
		return excerpt.CreateMetadata[key] == value
	})
}

func (c *RepoCache) ResolveBugExcerptMatcher(f func(*BugExcerpt) bool) (*BugExcerpt, error) {
	id, err := c.resolveBugMatcher(f)
	if err != nil {
		return nil, err
	}
	return c.ResolveBugExcerpt(id)
}

func (c *RepoCache) ResolveBugMatcher(f func(*BugExcerpt) bool) (*BugCache, error) {
	id, err := c.resolveBugMatcher(f)
	if err != nil {
		return nil, err
	}
	return c.ResolveBug(id)
}

func (c *RepoCache) resolveBugMatcher(f func(*BugExcerpt) bool) (entity.Id, error) {
	c.muBug.RLock()
	defer c.muBug.RUnlock()

	// preallocate but empty
	matching := make([]entity.Id, 0, 5)

	for _, excerpt := range c.bugExcerpts {
		if f(excerpt) {
			matching = append(matching, excerpt.Id)
		}
	}

	if len(matching) > 1 {
		return entity.UnsetId, bug.NewErrMultipleMatchBug(matching)
	}

	if len(matching) == 0 {
		return entity.UnsetId, bug.ErrBugNotExist
	}

	return matching[0], nil
}

// ResolveComment search for a Bug/Comment combination matching the merged
// bug/comment Id prefix. Returns the Bug containing the Comment and the Comment's
// Id.
func (c *RepoCache) ResolveComment(prefix string) (*BugCache, entity.CombinedId, error) {
	bugPrefix, _ := entity.SeparateIds(prefix)
	bugCandidate := make([]entity.Id, 0, 5)

	// build a list of possible matching bugs
	c.muBug.RLock()
	for _, excerpt := range c.bugExcerpts {
		if excerpt.Id.HasPrefix(bugPrefix) {
			bugCandidate = append(bugCandidate, excerpt.Id)
		}
	}
	c.muBug.RUnlock()

	matchingBugIds := make([]entity.Id, 0, 5)
	matchingCommentId := entity.UnsetCombinedId
	var matchingBug *BugCache

	// search for matching comments
	// searching every bug candidate allow for some collision with the bug prefix only,
	// before being refined with the full comment prefix
	for _, bugId := range bugCandidate {
		b, err := c.ResolveBug(bugId)
		if err != nil {
			return nil, entity.UnsetCombinedId, err
		}

		for _, comment := range b.Snapshot().Comments {
			if comment.Id().HasPrefix(prefix) {
				matchingBugIds = append(matchingBugIds, bugId)
				matchingBug = b
				matchingCommentId = comment.Id()
			}
		}
	}

	if len(matchingBugIds) > 1 {
		return nil, entity.UnsetCombinedId, entity.NewErrMultipleMatch("bug/comment", matchingBugIds)
	} else if len(matchingBugIds) == 0 {
		return nil, entity.UnsetCombinedId, errors.New("comment doesn't exist")
	}

	return matchingBug, matchingCommentId, nil
}

// QueryBugs return the id of all Bug matching the given Query
func (c *RepoCache) QueryBugs(q *query.Query) ([]entity.Id, error) {
	c.muBug.RLock()
	defer c.muBug.RUnlock()

	if q == nil {
		return c.AllBugsIds(), nil
	}

	matcher := compileMatcher(q.Filters)

	var filtered []*BugExcerpt
	var foundBySearch map[entity.Id]*BugExcerpt

	if q.Search != nil {
		foundBySearch = map[entity.Id]*BugExcerpt{}

		terms := make([]string, len(q.Search))
		copy(terms, q.Search)
		for i, search := range q.Search {
			if strings.Contains(search, " ") {
				terms[i] = fmt.Sprintf("\"%s\"", search)
			}
		}

		bleveQuery := bleve.NewQueryStringQuery(strings.Join(terms, " "))
		bleveSearch := bleve.NewSearchRequest(bleveQuery)

		index, err := c.repo.GetBleveIndex("bug")
		if err != nil {
			return nil, err
		}

		searchResults, err := index.Search(bleveSearch)
		if err != nil {
			return nil, err
		}

		for _, hit := range searchResults.Hits {
			foundBySearch[entity.Id(hit.ID)] = c.bugExcerpts[entity.Id(hit.ID)]
		}
	} else {
		foundBySearch = c.bugExcerpts
	}

	for _, excerpt := range foundBySearch {
		if matcher.Match(excerpt, c) {
			filtered = append(filtered, excerpt)
		}
	}

	var sorter sort.Interface

	switch q.OrderBy {
	case query.OrderById:
		sorter = BugsById(filtered)
	case query.OrderByCreation:
		sorter = BugsByCreationTime(filtered)
	case query.OrderByEdit:
		sorter = BugsByEditTime(filtered)
	default:
		return nil, errors.New("missing sort type")
	}

	switch q.OrderDirection {
	case query.OrderAscending:
		// Nothing to do
	case query.OrderDescending:
		sorter = sort.Reverse(sorter)
	default:
		return nil, errors.New("missing sort direction")
	}

	sort.Sort(sorter)

	result := make([]entity.Id, len(filtered))

	for i, val := range filtered {
		result[i] = val.Id
	}

	return result, nil
}

// AllBugsIds return all known bug ids
func (c *RepoCache) AllBugsIds() []entity.Id {
	c.muBug.RLock()
	defer c.muBug.RUnlock()

	result := make([]entity.Id, len(c.bugExcerpts))

	i := 0
	for _, excerpt := range c.bugExcerpts {
		result[i] = excerpt.Id
		i++
	}

	return result
}

// ValidLabels list valid labels
//
// Note: in the future, a proper label policy could be implemented where valid
// labels are defined in a configuration file. Until that, the default behavior
// is to return the list of labels already used.
func (c *RepoCache) ValidLabels() []bug.Label {
	c.muBug.RLock()
	defer c.muBug.RUnlock()

	set := map[bug.Label]interface{}{}

	for _, excerpt := range c.bugExcerpts {
		for _, l := range excerpt.Labels {
			set[l] = nil
		}
	}

	result := make([]bug.Label, len(set))

	i := 0
	for l := range set {
		result[i] = l
		i++
	}

	// Sort
	sort.Slice(result, func(i, j int) bool {
		return string(result[i]) < string(result[j])
	})

	return result
}

// NewBug create a new bug
// The new bug is written in the repository (commit)
func (c *RepoCache) NewBug(title string, message string) (*BugCache, *bug.CreateOperation, error) {
	return c.NewBugWithFiles(title, message, nil)
}

// NewBugWithFiles create a new bug with attached files for the message
// The new bug is written in the repository (commit)
func (c *RepoCache) NewBugWithFiles(title string, message string, files []repository.Hash) (*BugCache, *bug.CreateOperation, error) {
	author, err := c.GetUserIdentity()
	if err != nil {
		return nil, nil, err
	}

	return c.NewBugRaw(author, time.Now().Unix(), title, message, files, nil)
}

// NewBugWithFilesMeta create a new bug with attached files for the message, as
// well as metadata for the Create operation.
// The new bug is written in the repository (commit)
func (c *RepoCache) NewBugRaw(author *IdentityCache, unixTime int64, title string, message string, files []repository.Hash, metadata map[string]string) (*BugCache, *bug.CreateOperation, error) {
	b, op, err := bug.CreateWithFiles(author.Identity, unixTime, title, message, files)
	if err != nil {
		return nil, nil, err
	}

	for key, value := range metadata {
		op.SetMetadata(key, value)
	}

	err = b.Commit(c.repo)
	if err != nil {
		return nil, nil, err
	}

	c.muBug.Lock()
	if _, has := c.bugs[b.Id()]; has {
		c.muBug.Unlock()
		return nil, nil, fmt.Errorf("bug %s already exist in the cache", b.Id())
	}

	cached := NewBugCache(c, b)
	c.bugs[b.Id()] = cached
	c.loadedBugs.Add(b.Id())
	c.muBug.Unlock()

	c.evictIfNeeded()

	// force the write of the excerpt
	err = c.bugUpdated(b.Id())
	if err != nil {
		return nil, nil, err
	}

	return cached, op, nil
}

// RemoveBug removes a bug from the cache and repo given a bug id prefix
func (c *RepoCache) RemoveBug(prefix string) error {
	c.muBug.RLock()

	b, err := c.ResolveBugPrefix(prefix)
	if err != nil {
		c.muBug.RUnlock()
		return err
	}
	c.muBug.RUnlock()

	c.muBug.Lock()
	err = bug.RemoveBug(c.repo, b.Id())

	delete(c.bugs, b.Id())
	delete(c.bugExcerpts, b.Id())
	c.loadedBugs.Remove(b.Id())

	c.muBug.Unlock()

	return c.writeBugCache()
}

func (c *RepoCache) addBugToSearchIndex(snap *bug.Snapshot) error {
	searchableBug := struct {
		Text []string
	}{}

	// See https://github.com/blevesearch/bleve/issues/1576
	var sb strings.Builder
	normalize := func(text string) string {
		sb.Reset()
		for _, field := range strings.Fields(text) {
			if utf8.RuneCountInString(field) < 100 {
				sb.WriteString(field)
				sb.WriteRune(' ')
			}
		}
		return sb.String()
	}

	for _, comment := range snap.Comments {
		searchableBug.Text = append(searchableBug.Text, normalize(comment.Message))
	}

	searchableBug.Text = append(searchableBug.Text, normalize(snap.Title))

	index, err := c.repo.GetBleveIndex("bug")
	if err != nil {
		return err
	}

	err = index.Index(snap.Id().String(), searchableBug)
	if err != nil {
		return err
	}

	return nil
}
