package iterator

import (
	"context"
	"time"

	"code.gitea.io/sdk/gitea"
)

type Iterator struct {
	// shared context
	ctx context.Context

	// to pass to sub-iterators
	conf config

	// sticky error
	err error

	// issues iterator
	issue *issueIterator

	// comments iterator
	comment *commentIterator

	// labels iterator
	label *labelIterator
}

type config struct {
	// gitea api v1 client
	gc *gitea.Client

	timeout time.Duration

	// if since is given the iterator will query only the issues
	// updated after this date
	since time.Time

	// name of the repository owner on Gitea
	owner string

	// name of the Gitea repository
	project string

	// number of issues and notes to query at once
	capacity int
}

// NewIterator create a new iterator
func NewIterator(ctx context.Context, client *gitea.Client, capacity int, owner, project string, timeout time.Duration, since time.Time) *Iterator {
	return &Iterator{
		ctx: ctx,
		conf: config{
			gc:       client,
			timeout:  timeout,
			since:    since,
			owner:    owner,
			project:  project,
			capacity: capacity,
		},
		issue:   newIssueIterator(),
		comment: newCommentIterator(),
		label:   newLabelIterator(),
	}
}

// Error return last encountered error
func (i *Iterator) Error() error {
	return i.err
}

func (i *Iterator) NextIssue() bool {
	if i.err != nil {
		return false
	}

	if i.ctx.Err() != nil {
		return false
	}

	more, err := i.issue.Next(i.ctx, i.conf)
	if err != nil {
		i.err = err
		return false
	}

	// Also reset the other sub iterators as they would
	// no longer be valid
	i.comment.Reset(i.issue.Value().Index)
	i.label.Reset(i.issue.Value().Index)

	return more
}

func (i *Iterator) IssueValue() *gitea.Issue {
	return i.issue.Value()
}

func (i *Iterator) NextComment() bool {
	if i.err != nil {
		return false
	}

	if i.ctx.Err() != nil {
		return false
	}

	more, err := i.comment.Next(i.ctx, i.conf)
	if err != nil {
		i.err = err
		return false
	}

	return more
}

func (i *Iterator) CommentValue() *gitea.Comment {
	return i.comment.Value()
}

func (i *Iterator) NextLabel() bool {
	if i.err != nil {
		return false
	}

	if i.ctx.Err() != nil {
		return false
	}

	more, err := i.label.Next(i.ctx, i.conf)
	if err != nil {
		i.err = err
		return false
	}

	return more
}

func (i *Iterator) LabelValue() *gitea.Label {
	return i.label.Value()
}
