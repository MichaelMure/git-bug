package entity

import (
	"time"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/repository"
)

// OperationType is an operation type identifier
type OperationType int

// Operation is a piece of data defining a change to reflect on the state of an Entity.
// What this Operation or Entity's state looks like is not of the resort of this package as it only deals with the
// data structure and storage.
type Operation interface {
	// Id return the Operation identifier
	//
	// Some care need to be taken to define a correct Id derivation and enough entropy in the data used to avoid
	// collisions. Notably:
	// - the Id of the first Operation will be used as the Id of the Entity. Collision need to be avoided across entities
	//   of the same type (example: no collision within the "bug" namespace).
	// - collisions can also happen within the set of Operations of an Entity. Simple Operation might not have enough
	//   entropy to yield unique Ids (example: two "close" operation within the same second, same author).
	//   If this is a concern, it is recommended to include a piece of random data in the operation's data, to guarantee
	//   a minimal amount of entropy and avoid collision.
	//
	//   Author's note: I tried to find a clever way around that inelegance (stuffing random useless data into the stored
	//   structure is not exactly elegant), but I failed to find a proper way. Essentially, anything that would reuse some
	//   other data (parent operation's Id, lamport clock) or the graph structure (depth) impose that the Id would only
	//   make sense in the context of the graph and yield some deep coupling between Entity and Operation. This in turn
	//   make the whole thing even less elegant.
	//
	// A common way to derive an Id will be to use the entity.DeriveId() function on the serialized operation data.
	Id() Id
	// Type return the type of the operation
	Type() OperationType
	// Validate check if the Operation data is valid
	Validate() error
	// Author returns the author of this operation
	Author() identity.Interface
	// Time return the time when the operation was added
	Time() time.Time

	// SetMetadata store arbitrary metadata about the operation
	SetMetadata(key string, value string)
	// GetMetadata retrieve arbitrary metadata about the operation
	GetMetadata(key string) (string, bool)
	// AllMetadata return all metadata for this operation
	AllMetadata() map[string]string
}

type OperationWithApply[OpT Operation, SnapT Snapshot] interface {
	Operation

	// Apply the operation to a Snapshot to create the final state
	Apply(snapshot SnapT)
}

// OperationWithFiles is an optional extension for an Operation that has files dependency, stored in git.
type OperationWithFiles interface {
	Operation

	// GetFiles return the files needed by this operation
	// This implies that the Operation maintain and store internally the references to those files. This is how
	// this information is read later, when loading from storage.
	// For example, an operation that has a text value referencing some files would maintain a mapping (text ref -->
	// hash).
	GetFiles() []repository.Hash
}

// OperationDoesntChangeSnapshot is an interface signaling that the Operation implementing it doesn't change the
// snapshot, for example a metadata operation that act on other operations.
type OperationDoesntChangeSnapshot interface {
	DoesntChangeSnapshot()
}
