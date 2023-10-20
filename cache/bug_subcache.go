package cache

import (
	"errors"
	"sort"
	"time"

	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/query"
	"github.com/MichaelMure/git-bug/repository"
)

type RepoCacheBug struct {
	*SubCache[*bug.Bug, *BugExcerpt, *BugCache]
}

func NewRepoCacheBug(repo repository.ClockedRepo,
	resolvers func() entity.Resolvers,
	getUserIdentity getUserIdentityFunc) *RepoCacheBug {

	makeCached := func(b *bug.Bug, entityUpdated func(id entity.Id) error) *BugCache {
		return NewBugCache(b, repo, getUserIdentity, entityUpdated)
	}

	makeIndexData := func(b *BugCache) []string {
		snap := b.Compile()
		var res []string
		for _, comment := range snap.Comments {
			res = append(res, comment.Message)
		}
		res = append(res, snap.Title)
		return res
	}

	actions := Actions[*bug.Bug]{
		ReadWithResolver:    bug.ReadWithResolver,
		ReadAllWithResolver: bug.ReadAllWithResolver,
		Remove:              bug.Remove,
		RemoveAll:           bug.RemoveAll,
		MergeAll:            bug.MergeAll,
	}

	sc := NewSubCache[*bug.Bug, *BugExcerpt, *BugCache](
		repo, resolvers, getUserIdentity,
		makeCached, NewBugExcerpt, makeIndexData, actions,
		bug.Typename, bug.Namespace,
		formatVersion, defaultMaxLoadedBugs,
	)

	return &RepoCacheBug{SubCache: sc}
}

// ResolveBugCreateMetadata retrieve a bug that has the exact given metadata on
// its Create operation, that is, the first operation. It fails if multiple bugs
// match.
func (c *RepoCacheBug) ResolveBugCreateMetadata(key string, value string) (*BugCache, error) {
	return c.ResolveMatcher(func(excerpt *BugExcerpt) bool {
		return excerpt.CreateMetadata[key] == value
	})
}

// ResolveComment search for a Bug/Comment combination matching the merged
// bug/comment Id prefix. Returns the Bug containing the Comment and the Comment's
// Id.
func (c *RepoCacheBug) ResolveComment(prefix string) (*BugCache, entity.CombinedId, error) {
	bugPrefix, _ := entity.SeparateIds(prefix)
	bugCandidate := make([]entity.Id, 0, 5)

	// build a list of possible matching bugs
	c.mu.RLock()
	for _, excerpt := range c.excerpts {
		if excerpt.Id().HasPrefix(bugPrefix) {
			bugCandidate = append(bugCandidate, excerpt.Id())
		}
	}
	c.mu.RUnlock()

	matchingBugIds := make([]entity.Id, 0, 5)
	matchingCommentId := entity.UnsetCombinedId
	var matchingBug *BugCache

	// search for matching comments
	// searching every bug candidate allow for some collision with the bug prefix only,
	// before being refined with the full comment prefix
	for _, bugId := range bugCandidate {
		b, err := c.Resolve(bugId)
		if err != nil {
			return nil, entity.UnsetCombinedId, err
		}

		for _, comment := range b.Compile().Comments {
			if comment.CombinedId().HasPrefix(prefix) {
				matchingBugIds = append(matchingBugIds, bugId)
				matchingBug = b
				matchingCommentId = comment.CombinedId()
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

// Query return the id of all Bug matching the given Query
func (c *RepoCacheBug) Query(q *query.Query) ([]entity.Id, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if q == nil {
		return c.AllIds(), nil
	}

	matcher := compileMatcher(q.Filters)

	var filtered []*BugExcerpt
	var foundBySearch map[entity.Id]*BugExcerpt

	if q.Search != nil {
		foundBySearch = map[entity.Id]*BugExcerpt{}

		index, err := c.repo.GetIndex("bugs")
		if err != nil {
			return nil, err
		}

		res, err := index.Search(q.Search)
		if err != nil {
			return nil, err
		}

		for _, hit := range res {
			id := entity.Id(hit)
			foundBySearch[id] = c.excerpts[id]
		}
	} else {
		foundBySearch = c.excerpts
	}

	for _, excerpt := range foundBySearch {
		if matcher.Match(excerpt, c.resolvers()) {
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
		result[i] = val.Id()
	}

	return result, nil
}

// ValidLabels list valid labels
//
// Note: in the future, a proper label policy could be implemented where valid
// labels are defined in a configuration file. Until that, the default behavior
// is to return the list of labels already used.
func (c *RepoCacheBug) ValidLabels() []bug.Label {
	c.mu.RLock()
	defer c.mu.RUnlock()

	set := map[bug.Label]interface{}{}

	for _, excerpt := range c.excerpts {
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

// New create a new bug
// The new bug is written in the repository (commit)
func (c *RepoCacheBug) New(title string, message string) (*BugCache, *bug.CreateOperation, error) {
	return c.NewWithFiles(title, message, nil)
}

// NewWithFiles create a new bug with attached files for the message
// The new bug is written in the repository (commit)
func (c *RepoCacheBug) NewWithFiles(title string, message string, files []repository.Hash) (*BugCache, *bug.CreateOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, nil, err
	}

	return c.NewRaw(author, time.Now().Unix(), title, message, files, nil)
}

// NewRaw create a new bug with attached files for the message, as
// well as metadata for the Create operation.
// The new bug is written in the repository (commit)
func (c *RepoCacheBug) NewRaw(author entity.Interface, unixTime int64, title string, message string, files []repository.Hash, metadata map[string]string) (*BugCache, *bug.CreateOperation, error) {
	b, op, err := bug.Create(author, unixTime, title, message, files, metadata)
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
