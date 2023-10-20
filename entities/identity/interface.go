package identity

import (
	bootstrap "github.com/MichaelMure/git-bug/entity/boostrap"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
	"github.com/MichaelMure/git-bug/util/timestamp"
)

type Interface interface {
	bootstrap.Entity

	// Name return the last version of the name
	// Can be empty.
	Name() string

	// DisplayName return a non-empty string to display, representing the
	// identity, based on the non-empty values.
	DisplayName() string

	// Email return the last version of the email
	// Can be empty.
	Email() string

	// Login return the last version of the login
	// Can be empty.
	// Warning: this login can be defined when importing from a bridge but should *not* be
	// used to identify an identity as multiple bridge with different login can map to the same
	// identity. Use the metadata system for that usage instead.
	Login() string

	// AvatarUrl return the last version of the Avatar URL
	// Can be empty.
	AvatarUrl() string

	// Keys return the last version of the valid keys
	// Can be empty.
	Keys() []*Key

	// SigningKey return the key that should be used to sign new messages. If no key is available, return nil.
	SigningKey(repo repository.RepoKeyring) (*Key, error)

	// ValidKeysAtTime return the set of keys valid at a given lamport time for a given clock of another entity
	// Can be empty.
	ValidKeysAtTime(clockName string, time lamport.Time) []*Key

	// LastModification return the timestamp at which the last version of the identity became valid.
	LastModification() timestamp.Timestamp

	// LastModificationLamports return the lamport times at which the last version of the identity became valid.
	LastModificationLamports() map[string]lamport.Time

	// IsProtected return true if the chain of git commits started to be signed.
	// If that's the case, only signed commit with a valid key for this identity can be added.
	IsProtected() bool

	// Validate check if the Identity data is valid
	Validate() error

	// NeedCommit indicate that the in-memory state changed and need to be committed in the repository
	NeedCommit() bool
}
