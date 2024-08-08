// Package repository contains helper methods for working with a Git repo.
package repository

import (
	"errors"
	"io"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"

	"github.com/MichaelMure/git-bug/util/lamport"
)

var (
	// ErrNotARepo is the error returned when the git repo root can't be found
	ErrNotARepo = errors.New("not a git repository")
	// ErrClockNotExist is the error returned when a clock can't be found
	ErrClockNotExist = errors.New("clock doesn't exist")
	// ErrNotFound is the error returned when a git object can't be found
	ErrNotFound = errors.New("ref not found")
)

// Repo represents a source code repository.
type Repo interface {
	RepoConfig
	RepoKeyring
	RepoCommon
	RepoStorage
	RepoIndex
	RepoData

	Close() error
}

type RepoCommonStorage interface {
	RepoCommon
	RepoStorage
}

// ClockedRepo is a Repo that also has Lamport clocks
type ClockedRepo interface {
	Repo
	RepoClock
}

// RepoConfig access the configuration of a repository
type RepoConfig interface {
	// LocalConfig give access to the repository scoped configuration
	LocalConfig() Config

	// GlobalConfig give access to the global scoped configuration
	GlobalConfig() Config

	// AnyConfig give access to a merged local/global configuration
	AnyConfig() ConfigRead
}

// RepoKeyring give access to a user-wide storage for secrets
type RepoKeyring interface {
	// Keyring give access to a user-wide storage for secrets
	Keyring() Keyring
}

// RepoCommon represent the common function we want all repos to implement
type RepoCommon interface {
	// GetUserName returns the name the user has used to configure git
	GetUserName() (string, error)

	// GetUserEmail returns the email address that the user has used to configure git.
	GetUserEmail() (string, error)

	// GetCoreEditor returns the name of the editor that the user has used to configure git.
	GetCoreEditor() (string, error)

	// GetRemotes returns the configured remotes repositories.
	GetRemotes() (map[string]string, error)
}

type LocalStorage interface {
	billy.Filesystem
	RemoveAll(path string) error
}

// RepoStorage give access to the filesystem
type RepoStorage interface {
	// LocalStorage return a billy.Filesystem giving access to $RepoPath/.git/git-bug
	LocalStorage() LocalStorage
}

// RepoIndex gives access to full-text search indexes
type RepoIndex interface {
	GetIndex(name string) (Index, error)
}

// Index is a full-text search index
type Index interface {
	// IndexOne indexes one document, for the given ID. If the document already exist,
	// it replaces it.
	IndexOne(id string, texts []string) error

	// IndexBatch start a batch indexing. The returned indexer function is used the same
	// way as IndexOne, and the closer function complete the batch insertion.
	IndexBatch() (indexer func(id string, texts []string) error, closer func() error)

	// Search returns the list of IDs matching the given terms.
	Search(terms []string) (ids []string, err error)

	// DocCount returns the number of document in the index.
	DocCount() (uint64, error)

	// Remove delete one document in the index.
	Remove(id string) error

	// Clear empty the index.
	Clear() error

	// Close closes the index and make sure everything is safely written. After this call
	// the index can't be used anymore.
	Close() error
}

type Commit struct {
	Hash       Hash
	Parents    []Hash    // hashes of the parents, if any
	TreeHash   Hash      // hash of the git Tree
	SignedData io.Reader // if signed, reader for the signed data (likely, the serialized commit)
	Signature  io.Reader // if signed, reader for the (non-armored) signature
}

// RepoData give access to the git data storage
type RepoData interface {
	// FetchRefs fetch git refs matching a directory prefix to a remote
	// Ex: prefix="foo" will fetch any remote refs matching "refs/foo/*" locally.
	// The equivalent git refspec would be "refs/foo/*:refs/remotes/<remote>/foo/*"
	FetchRefs(remote string, prefixes ...string) (string, error)

	// PushRefs push git refs matching a directory prefix to a remote
	// Ex: prefix="foo" will push any local refs matching "refs/foo/*" to the remote.
	// The equivalent git refspec would be "refs/foo/*:refs/foo/*"
	//
	// Additionally, PushRefs will update the local references in refs/remotes/<remote>/foo to match
	// the remote state.
	PushRefs(remote string, prefixes ...string) (string, error)

	// SSHAuth will attempt to read public keys for SSH auth
	SSHAuth(remote string) (*ssh.PublicKeys, error)

	// StoreData will store arbitrary data and return the corresponding hash
	StoreData(data []byte) (Hash, error)

	// ReadData will attempt to read arbitrary data from the given hash
	// Returns ErrNotFound if not found.
	ReadData(hash Hash) ([]byte, error)

	// StoreTree will store a mapping key-->Hash as a Git tree
	StoreTree(mapping []TreeEntry) (Hash, error)

	// ReadTree will return the list of entries in a Git tree
	// The given hash could be from either a commit or a tree
	// Returns ErrNotFound if not found.
	ReadTree(hash Hash) ([]TreeEntry, error)

	// StoreCommit will store a Git commit with the given Git tree
	StoreCommit(treeHash Hash, parents ...Hash) (Hash, error)

	// StoreSignedCommit will store a Git commit with the given Git tree. If signKey is not nil, the commit
	// will be signed accordingly.
	StoreSignedCommit(treeHash Hash, signKey *openpgp.Entity, parents ...Hash) (Hash, error)

	// ReadCommit read a Git commit and returns some of its characteristic
	// Returns ErrNotFound if not found.
	ReadCommit(hash Hash) (Commit, error)

	// ResolveRef returns the hash of the target commit of the given ref
	// Returns ErrNotFound if not found.
	ResolveRef(ref string) (Hash, error)

	// UpdateRef will create or update a Git reference
	UpdateRef(ref string, hash Hash) error

	// RemoveRef will remove a Git reference
	// RemoveRef is idempotent.
	RemoveRef(ref string) error

	// ListRefs will return a list of Git ref matching the given refspec
	ListRefs(refPrefix string) ([]string, error)

	// RefExist will check if a reference exist in Git
	RefExist(ref string) (bool, error)

	// CopyRef will create a new reference with the same value as another one
	// Returns ErrNotFound if not found.
	CopyRef(source string, dest string) error

	// ListCommits will return the list of tree hashes of a ref, in chronological order
	ListCommits(ref string) ([]Hash, error)
}

// RepoClock give access to Lamport clocks
type RepoClock interface {
	// AllClocks return all the known clocks
	AllClocks() (map[string]lamport.Clock, error)

	// GetOrCreateClock return a Lamport clock stored in the Repo.
	// If the clock doesn't exist, it's created.
	GetOrCreateClock(name string) (lamport.Clock, error)

	// Increment is equivalent to c = GetOrCreateClock(name) + c.Increment()
	Increment(name string) (lamport.Time, error)

	// Witness is equivalent to c = GetOrCreateClock(name) + c.Witness(time)
	Witness(name string, time lamport.Time) error
}

// ClockLoader hold which logical clock need to exist for an entity and
// how to create them if they don't.
type ClockLoader struct {
	// Clocks hold the name of all the clocks this loader deal with.
	// Those clocks will be checked when the repo load. If not present or broken,
	// Witnesser will be used to create them.
	Clocks []string
	// Witnesser is a function that will initialize the clocks of a repo
	// from scratch
	Witnesser func(repo ClockedRepo) error
}

// TestedRepo is an extended ClockedRepo with function for testing only
type TestedRepo interface {
	ClockedRepo
	repoTest
}

// repoTest give access to test only functions
type repoTest interface {
	// AddRemote add a new remote to the repository
	AddRemote(name string, url string) error

	// GetLocalRemote return the URL to use to add this repo as a local remote
	GetLocalRemote() string

	// EraseFromDisk delete this repository entirely from the disk
	EraseFromDisk() error
}
