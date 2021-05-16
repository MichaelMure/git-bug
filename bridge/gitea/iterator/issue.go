package iterator

import (
	"context"
	"strconv"

	"code.gitea.io/sdk/gitea"
)

type issueIterator struct {
	page     int
	lastPage bool
	index    int
	cache    []*gitea.Issue
}

func newIssueIterator() *issueIterator {
	ii := &issueIterator{}
	ii.Reset()
	return ii
}

func (ii *issueIterator) Next(ctx context.Context, conf config) (bool, error) {
	// first query
	if ii.cache == nil {
		return ii.getNext(ctx, conf)
	}

	// move cursor index
	if ii.index < len(ii.cache)-1 {
		ii.index++
		return true, nil
	}

	return ii.getNext(ctx, conf)
}

func (ii *issueIterator) Value() *gitea.Issue {
	return ii.cache[ii.index]
}

func (ii *issueIterator) getNext(ctx context.Context, conf config) (bool, error) {
	if ii.lastPage {
		return false, nil
	}

	ctx, cancel := context.WithTimeout(ctx, conf.timeout)
	defer cancel()
	conf.gc.SetContext(ctx)

	issues, resp, err := conf.gc.ListRepoIssues(
		conf.owner,
		conf.project,
		gitea.ListIssueOption{
			ListOptions: gitea.ListOptions{
				Page:     ii.page,
				PageSize: conf.capacity,
			},
			State: gitea.StateAll,
			Type:  gitea.IssueTypeIssue,
		},
	)

	if err != nil {
		ii.Reset()
		return false, err
	}

	total, err := strconv.Atoi(resp.Header.Get("X-Total-Count"))
	if err != nil {
		ii.Reset()
		return false, err
	}

	if total <= ii.page*conf.capacity {
		ii.lastPage = true
	}

	// if repository doesn't have any issues
	if len(issues) == 0 {
		return false, nil
	}

	ii.cache = issues
	ii.index = 0
	ii.page++

	return true, nil
}

func (ii *issueIterator) Reset() {
	ii.index = -1
	ii.page = 1
	ii.lastPage = false
	ii.cache = nil
}
