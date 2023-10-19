package bug

import (
	"strings"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/timestamp"
)

type TimelineItem interface {
	// CombinedId returns the global identifier of the item
	CombinedId() entity.CombinedId
}

// CommentHistoryStep hold one version of a message in the history
type CommentHistoryStep struct {
	// The author of the edition, not necessarily the same as the author of the
	// original comment
	Author identity.Interface
	// The new message
	Message  string
	Deleted  bool
	UnixTime timestamp.Timestamp
}

// CommentTimelineItem is a TimelineItem that holds a Comment and its edition history
type CommentTimelineItem struct {
	combinedId entity.CombinedId
	Author     identity.Interface
	Message    string
	Deleted    bool
	Files      []repository.Hash
	CreatedAt  timestamp.Timestamp
	LastEdit   timestamp.Timestamp
	History    []CommentHistoryStep
}

func NewCommentTimelineItem(comment Comment) CommentTimelineItem {
	return CommentTimelineItem{
		// id: comment.id,
		combinedId: comment.combinedId,
		Author:     comment.Author,
		Message:    comment.Message,
		Files:      comment.Files,
		CreatedAt:  comment.unixTime,
		LastEdit:   comment.unixTime,
		History: []CommentHistoryStep{
			{
				Message:  comment.Message,
				UnixTime: comment.unixTime,
			},
		},
	}
}

func (c *CommentTimelineItem) CombinedId() entity.CombinedId {
	return c.combinedId
}

// Append will append a new comment in the history and update the other values
func (c *CommentTimelineItem) Append(comment Comment) {
	c.Message = comment.Message
	c.Deleted = comment.Deleted
	c.Files = comment.Files
	c.LastEdit = comment.unixTime
	c.History = append(c.History, CommentHistoryStep{
		Author:   comment.Author,
		Message:  comment.Message,
		Deleted:  comment.Deleted,
		UnixTime: comment.unixTime,
	})
}

// Edited say if the comment was edited
func (c *CommentTimelineItem) Edited() bool {
	return len(c.History) > 1
}

func (c *CommentTimelineItem) DeletedBy() identity.Interface {
	return c.History[len(c.History)-1].Author
}

// MessageIsEmpty return true is the message is empty or only made of spaces
func (c *CommentTimelineItem) MessageIsEmpty() bool {
	return len(strings.TrimSpace(c.Message)) == 0
}
