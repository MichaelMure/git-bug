package board

import (
	"github.com/dustin/go-humanize"

	"github.com/MichaelMure/git-bug/entities/common"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/timestamp"
)

var _ CardItem = &Draft{}

type Draft struct {
	status  common.Status
	title   string
	message string
	files   []repository.Hash

	// Creation time of the comment.
	// Should be used only for human display, never for ordering as we can't rely on it in a distributed system.
	unixTime timestamp.Timestamp
}

func (d *Draft) Status() common.Status {
	// TODO implement me
	panic("implement me")
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
