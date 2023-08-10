package iterator

import (
	"context"

	"code.gitea.io/sdk/gitea"
)

type labelIterator struct {
	issue    int64
	page     int
	lastPage bool
	index    int
	cache    []*gitea.Label
}

func newLabelIterator() *labelIterator {
	li := &labelIterator{}
	li.Reset(-1)
	return li
}

func (li *labelIterator) Next(ctx context.Context, conf config) (bool, error) {
	// first query
	if li.cache == nil {
		return li.getNext(ctx, conf)
	}

	// move cursor index
	if li.index < len(li.cache)-1 {
		li.index++
		return true, nil
	}

	return li.getNext(ctx, conf)
}

func (li *labelIterator) Value() *gitea.Label {
	return li.cache[li.index]
}

func (li *labelIterator) getNext(ctx context.Context, conf config) (bool, error) {
	if li.lastPage {
		return false, nil
	}

	ctx, cancel := context.WithTimeout(ctx, conf.timeout)
	defer cancel()
	conf.gc.SetContext(ctx)

	labels, _, err := conf.gc.GetIssueLabels(
		conf.owner,
		conf.project,
		li.issue,
		gitea.ListLabelsOptions{
			ListOptions: gitea.ListOptions{
				Page:     li.page,
				PageSize: conf.capacity,
			},
		},
	)
	if err != nil {
		li.Reset(-1)
		return false, err
	}

	li.lastPage = true

	if len(labels) == 0 {
		return false, nil
	}

	li.cache = labels
	li.index = 0
	li.page++

	return true, nil
}

func (li *labelIterator) Reset(issue int64) {
	li.issue = issue
	li.index = -1
	li.page = 1
	li.lastPage = false
	li.cache = nil
}
