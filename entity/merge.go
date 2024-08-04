package entity

import (
	bootstrap "github.com/MichaelMure/git-bug/entity/boostrap"
)

// MergeStatus represent the result of a merge operation of an entity
type MergeStatus = bootstrap.MergeStatus

const (
	MergeStatusNew     = bootstrap.MergeStatusNew     // a new Entity was created locally
	MergeStatusInvalid = bootstrap.MergeStatusInvalid // the remote data is invalid
	MergeStatusUpdated = bootstrap.MergeStatusUpdated // a local Entity has been updated
	MergeStatusNothing = bootstrap.MergeStatusNothing // no changes were made to a local Entity (already up to date)
	MergeStatusError   = bootstrap.MergeStatusError   // a terminal error happened
)

// MergeResult hold the result of a merge operation on an Entity.
type MergeResult = bootstrap.MergeResult

var NewMergeNewStatus = bootstrap.NewMergeNewStatus

var NewMergeInvalidStatus = bootstrap.NewMergeInvalidStatus

var NewMergeUpdatedStatus = bootstrap.NewMergeUpdatedStatus

var NewMergeNothingStatus = bootstrap.NewMergeNothingStatus

var NewMergeError = bootstrap.NewMergeError
