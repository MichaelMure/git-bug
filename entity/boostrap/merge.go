package bootstrap

import (
	"fmt"
)

// MergeStatus represent the result of a merge operation of an entity
type MergeStatus int

const (
	_                  MergeStatus = iota
	MergeStatusNew                 // a new Entity was created locally
	MergeStatusInvalid             // the remote data is invalid
	MergeStatusUpdated             // a local Entity has been updated
	MergeStatusNothing             // no changes were made to a local Entity (already up to date)
	MergeStatusError               // a terminal error happened
)

// MergeResult hold the result of a merge operation on an Entity.
type MergeResult struct {
	// Err is set when a terminal error occur in the process
	Err error

	Id     Id
	Status MergeStatus

	// Only set for Invalid status
	Reason string

	// Only set for New or Updated status
	Entity Entity
}

func (mr MergeResult) String() string {
	switch mr.Status {
	case MergeStatusNew:
		return "new"
	case MergeStatusInvalid:
		return fmt.Sprintf("invalid data: %s", mr.Reason)
	case MergeStatusUpdated:
		return "updated"
	case MergeStatusNothing:
		return "nothing to do"
	case MergeStatusError:
		if mr.Id != "" {
			return fmt.Sprintf("merge error on %s: %s", mr.Id, mr.Err.Error())
		}
		return fmt.Sprintf("merge error: %s", mr.Err.Error())
	default:
		panic("unknown merge status")
	}
}

func NewMergeNewStatus(id Id, entity Entity) MergeResult {
	return MergeResult{
		Id:     id,
		Status: MergeStatusNew,
		Entity: entity,
	}
}

func NewMergeInvalidStatus(id Id, reason string) MergeResult {
	return MergeResult{
		Id:     id,
		Status: MergeStatusInvalid,
		Reason: reason,
	}
}

func NewMergeUpdatedStatus(id Id, entity Entity) MergeResult {
	return MergeResult{
		Id:     id,
		Status: MergeStatusUpdated,
		Entity: entity,
	}
}

func NewMergeNothingStatus(id Id) MergeResult {
	return MergeResult{
		Id:     id,
		Status: MergeStatusNothing,
	}
}

func NewMergeError(err error, id Id) MergeResult {
	return MergeResult{
		Id:     id,
		Status: MergeStatusError,
		Err:    err,
	}
}
