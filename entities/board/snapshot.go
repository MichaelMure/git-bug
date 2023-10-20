package board

import (
	"fmt"
	"time"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
)

type Column struct {
	// id is the identifier of the column within the board context
	Id entity.Id
	// CombinedId is the global identifier of the column
	CombinedId entity.CombinedId
	Name       string
	Items      []Item
}

type Item interface {
	CombinedId() entity.CombinedId
	// TODO: all items have status?
	// Status() common.Status
}

var _ dag.Snapshot = &Snapshot{}

type Snapshot struct {
	id entity.Id

	Title        string
	Description  string
	Columns      []*Column
	Participants []identity.Interface

	CreateTime time.Time
	Operations []dag.Operation
}

// Id returns the Board identifier
func (snap *Snapshot) Id() entity.Id {
	if snap.id == "" {
		// simply panic as it would be a coding error (no id provided at construction)
		panic("no id")
	}
	return snap.id
}

func (snap *Snapshot) AllOperations() []dag.Operation {
	return snap.Operations
}

func (snap *Snapshot) AppendOperation(op dag.Operation) {
	snap.Operations = append(snap.Operations, op)
}

// EditTime returns the last time the board was modified
func (snap *Snapshot) EditTime() time.Time {
	if len(snap.Operations) == 0 {
		return time.Unix(0, 0)
	}

	return snap.Operations[len(snap.Operations)-1].Time()
}

// SearchColumn will search for a column matching the given id
func (snap *Snapshot) SearchColumn(id entity.CombinedId) (*Column, error) {
	for _, column := range snap.Columns {
		if column.CombinedId == id {
			return column, nil
		}
	}

	return nil, fmt.Errorf("column not found")
}

// append the operation author to the participants list
func (snap *Snapshot) addParticipant(participant identity.Interface) {
	for _, p := range snap.Participants {
		if participant.Id() == p.Id() {
			return
		}
	}

	snap.Participants = append(snap.Participants, participant)
}

// HasParticipant return true if the id is a participant
func (snap *Snapshot) HasParticipant(id entity.Id) bool {
	for _, p := range snap.Participants {
		if p.Id() == id {
			return true
		}
	}
	return false
}

// HasAnyParticipant return true if one of the ids is a participant
func (snap *Snapshot) HasAnyParticipant(ids ...entity.Id) bool {
	for _, id := range ids {
		if snap.HasParticipant(id) {
			return true
		}
	}
	return false
}

func (snap *Snapshot) ItemCount() int {
	var count int
	for _, column := range snap.Columns {
		count += len(column.Items)
	}
	return count
}

// IsAuthored is a sign post method for gqlgen
func (snap *Snapshot) IsAuthored() {}
