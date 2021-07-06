package bug

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/timestamp"
)

var _ Operation = &SetStatusOperation{}

// SetStatusOperation will change the status of a bug
type SetStatusOperation struct {
	OpBase
	Status Status `json:"status"`
}

func (op *SetStatusOperation) Id() entity.Id {
	return idOperation(op, &op.OpBase)
}

func (op *SetStatusOperation) Apply(snapshot *Snapshot) {
	snapshot.Status = op.Status
	snapshot.addActor(op.Author_)

	item := &SetStatusTimelineItem{
		id:       entity.CombineIds(snapshot.Id(), op.Id()),
		Author:   op.Author_,
		UnixTime: timestamp.Timestamp(op.UnixTime),
		Status:   op.Status,
	}

	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *SetStatusOperation) Validate() error {
	if err := op.OpBase.Validate(op, SetStatusOp); err != nil {
		return err
	}

	if err := op.Status.Validate(); err != nil {
		return errors.Wrap(err, "status")
	}

	return nil
}

// UnmarshalJSON is a two step JSON unmarshalling
// This workaround is necessary to avoid the inner OpBase.MarshalJSON
// overriding the outer op's MarshalJSON
func (op *SetStatusOperation) UnmarshalJSON(data []byte) error {
	// Unmarshal OpBase and the op separately

	base := OpBase{}
	err := json.Unmarshal(data, &base)
	if err != nil {
		return err
	}

	aux := struct {
		Status Status `json:"status"`
	}{}

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	op.OpBase = base
	op.Status = aux.Status

	return nil
}

// Sign post method for gqlgen
func (op *SetStatusOperation) IsAuthored() {}

func NewSetStatusOp(author identity.Interface, unixTime int64, status Status) *SetStatusOperation {
	return &SetStatusOperation{
		OpBase: newOpBase(SetStatusOp, author, unixTime),
		Status: status,
	}
}

type SetStatusTimelineItem struct {
	id       entity.CombinedId
	Author   identity.Interface
	UnixTime timestamp.Timestamp
	Status   Status
}

func (s SetStatusTimelineItem) Id() entity.CombinedId {
	return s.id
}

// Sign post method for gqlgen
func (s *SetStatusTimelineItem) IsAuthored() {}

// Convenience function to apply the operation
func Open(b Interface, author identity.Interface, unixTime int64) (*SetStatusOperation, error) {
	op := NewSetStatusOp(author, unixTime, OpenStatus)
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}

// Convenience function to apply the operation
func Close(b Interface, author identity.Interface, unixTime int64) (*SetStatusOperation, error) {
	op := NewSetStatusOp(author, unixTime, ClosedStatus)
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}
