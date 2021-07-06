package bug

import (
	"github.com/dustin/go-humanize"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/timestamp"
)

// Comment represent a comment in a Bug
type Comment struct {
	// id should be the result of entity.CombineIds with the Bug id and the id
	// of the Operation that created the comment
	id      entity.CombinedId
	Author  identity.Interface
	Message string
	Files   []repository.Hash

	// Creation time of the comment.
	// Should be used only for human display, never for ordering as we can't rely on it in a distributed system.
	UnixTime timestamp.Timestamp
}

// Id return the Comment identifier
func (c Comment) Id() entity.CombinedId {
	if c.id == "" {
		// simply panic as it would be a coding error (no id provided at construction)
		panic("no id")
	}
	return c.id
}

// FormatTimeRel format the UnixTime of the comment for human consumption
func (c Comment) FormatTimeRel() string {
	return humanize.Time(c.UnixTime.Time())
}

func (c Comment) FormatTime() string {
	return c.UnixTime.Time().Format("Mon Jan 2 15:04:05 2006 +0200")
}

// Sign post method for gqlgen
func (c Comment) IsAuthored() {}
