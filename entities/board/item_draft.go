package board

import (
	"github.com/dustin/go-humanize"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"

	"github.com/MichaelMure/git-bug/util/timestamp"
)

var _ Item = &Draft{}

type Draft struct {
	// combinedId should be the result of entity.CombineIds with the Board id and the id
	// of the Operation that created the Draft
	combinedId entity.CombinedId

	Author  identity.Interface
	Title   string
	Message string

	// Creation time of the comment.
	// Should be used only for human display, never for ordering as we can't rely on it in a distributed system.
	unixTime timestamp.Timestamp
}

func (d *Draft) CombinedId() entity.CombinedId {
	if d.combinedId == "" || d.combinedId == entity.UnsetCombinedId {
		// simply panic as it would be a coding error (no id provided at construction)
		panic("no combined id")
	}
	return d.combinedId
}

// FormatTimeRel format the UnixTime of the comment for human consumption
func (d *Draft) FormatTimeRel() string {
	return humanize.Time(d.unixTime.Time())
}

func (d *Draft) FormatTime() string {
	return d.unixTime.Time().Format("Mon Jan 2 15:04:05 2006 +0200")
}

// IsAuthored is a sign post method for gqlgen
func (d *Draft) IsAuthored() {}
