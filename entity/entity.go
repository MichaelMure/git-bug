package entity

import (
	bootstrap "github.com/MichaelMure/git-bug/entity/boostrap"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
)

type Bare bootstrap.Entity

// Interface define the extended interface of an Entity
type Interface[OpT Operation, SnapT Snapshot] interface {
	Bare
	CompileToSnapshot[OpT, SnapT]

	// Validate checks if the Entity data is valid
	Validate() error

	// Append an operation into the staging area, to be committed later
	Append(op OpT)

	// Operations returns the ordered operations
	Operations() []OpT

	// FirstOp lookup for the very first operation of the Entity.
	FirstOp() OpT

	// // LastOp lookup for the very last operation of the Entity.
	// // For a valid Entity, should never be nil
	// LastOp() OpT

	// CreateLamportTime return the Lamport time of creation
	CreateLamportTime() lamport.Time

	// EditLamportTime return the Lamport time of the last edit
	EditLamportTime() lamport.Time
}

type WithCommit[OpT Operation, SnapT Snapshot] interface {
	Interface[OpT, SnapT]
	Committer
}

type Committer interface {
	// NeedCommit indicates that the in-memory state changed and need to be committed in the repository
	NeedCommit() bool

	// Commit writes the staging area in Git and move the operations to the packs
	Commit(repo repository.ClockedRepo) error

	// CommitAsNeeded execute a Commit only if necessary. This function is useful to avoid getting an error if the Entity
	// is already in sync with the repository.
	CommitAsNeeded(repo repository.ClockedRepo) error
}
