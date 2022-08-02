package board

import (
	"time"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
)

type Column struct {
	Name  string
	Cards []CardItem
}

type CardItem interface {
	// Status() common.Status
}

type Snapshot struct {
	id entity.Id

	Title       string
	Description string
	Columns     []Column
	Actors      []identity.Interface

	CreateTime time.Time
	Operations []Operation
}

// Id returns the Board identifier
func (snap *Snapshot) Id() entity.Id {
	if snap.id == "" {
		// simply panic as it would be a coding error (no id provided at construction)
		panic("no id")
	}
	return snap.id
}

// EditTime returns the last time the board was modified
func (snap *Snapshot) EditTime() time.Time {
	if len(snap.Operations) == 0 {
		return time.Unix(0, 0)
	}

	return snap.Operations[len(snap.Operations)-1].Time()
}

// append the operation author to the actors list
func (snap *Snapshot) addActor(actor identity.Interface) {
	for _, a := range snap.Actors {
		if actor.Id() == a.Id() {
			return
		}
	}

	snap.Actors = append(snap.Actors, actor)
}
