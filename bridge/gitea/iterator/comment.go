package iterator

import (
	"context"

	"code.gitea.io/sdk/gitea"
)

type commentIterator struct {
	issue    int64
	page     int
	lastPage bool
	index    int
	cache    []*gitea.Comment
}

func newCommentIterator() *commentIterator {
	ci := &commentIterator{}
	ci.Reset(-1)
	return ci
}

func (ci *commentIterator) Next(ctx context.Context, conf config) (bool, error) {
	// first query
	if ci.cache == nil {
		return ci.getNext(ctx, conf)
	}

	// move cursor index
	if ci.index < len(ci.cache)-1 {
		ci.index++
		return true, nil
	}

	return ci.getNext(ctx, conf)
}

func (ci *commentIterator) Value() *gitea.Comment {
	return ci.cache[ci.index]
}

func (ci *commentIterator) getNext(ctx context.Context, conf config) (bool, error) {
	if ci.lastPage {
		return false, nil
	}

	ctx, cancel := context.WithTimeout(ctx, conf.timeout)
	defer cancel()
	conf.gc.SetContext(ctx)

	comments, _, err := conf.gc.ListIssueComments(
		conf.owner,
		conf.project,
		ci.issue,
		gitea.ListIssueCommentOptions{
			ListOptions: gitea.ListOptions{
				Page:     ci.page,
				PageSize: conf.capacity,
			},
		},
	)

	if err != nil {
		ci.Reset(-1)
		return false, err
	}

	ci.lastPage = true

	if len(comments) == 0 {
		return false, nil
	}

	ci.cache = comments
	ci.index = 0
	ci.page++

	return true, nil
}

func (ci *commentIterator) Reset(issue int64) {
	ci.issue = issue
	ci.index = -1
	ci.page = 1
	ci.lastPage = false
	ci.cache = nil
}
