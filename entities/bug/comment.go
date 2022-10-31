package bug

import (
	"github.com/dustin/go-humanize"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/timestamp"
)

// Comment represent a comment in a Bug
type Comment struct {
	// combinedId should be the result of entity.CombineIds with the Bug id and the id
	// of the Operation that created the comment
	combinedId entity.CombinedId

	// targetId is the Id of the Operation that originally created that Comment
	targetId entity.Id

	Author  identity.Interface
	Message string
	Files   []repository.Hash

	// Creation time of the comment.
	// Should be used only for human display, never for ordering as we can't rely on it in a distributed system.
	unixTime timestamp.Timestamp
}

func (c Comment) CombinedId() entity.CombinedId {
	if c.combinedId == "" {
		// simply panic as it would be a coding error (no id provided at construction)
		panic("no combined id")
	}
	return c.combinedId
}

func (c Comment) TargetId() entity.Id {
	return c.targetId
}

// FormatTimeRel format the unixTime of the comment for human consumption
func (c Comment) FormatTimeRel() string {
	return humanize.Time(c.unixTime.Time())
}

func (c Comment) FormatTime() string {
	return c.unixTime.Time().Format("Mon Jan 2 15:04:05 2006 +0200")
}

// IsAuthored is a sign post method for gqlgen
func (c Comment) IsAuthored() {}

func (c Comment) Shortened(len int) {

}
