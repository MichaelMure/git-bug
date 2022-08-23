package board

import (
	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/entity"
)

var _ Item = &BugItem{}

type BugItem struct {
	combinedId entity.CombinedId
	bug        bug.Interface
}

func (e *BugItem) CombinedId() entity.CombinedId {
	if e.combinedId == "" || e.combinedId == entity.UnsetCombinedId {
		// simply panic as it would be a coding error (no id provided at construction)
		panic("no combined id")
	}
	return e.combinedId
}
