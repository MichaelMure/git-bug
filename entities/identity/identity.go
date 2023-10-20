// Package identity contains the identity data model and low-level related functions
package identity

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/pkg/errors"

	bootstrap "github.com/MichaelMure/git-bug/entity/boostrap"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
	"github.com/MichaelMure/git-bug/util/timestamp"
)

const identityRefPattern = "refs/identities/"
const identityRemoteRefPattern = "refs/remotes/%s/identities/"
const versionEntryName = "version"
const identityConfigKey = "git-bug.identity"

const Typename = "identity"
const Namespace = "identities"

var ErrNonFastForwardMerge = errors.New("non fast-forward identity merge")
var ErrNoIdentitySet = errors.New("No identity is set.\n" +
	"To interact with bugs, an identity first needs to be created using " +
	"\"git bug user new\" or adopted with \"git bug user adopt\"")
var ErrMultipleIdentitiesSet = errors.New("multiple user identities set")

var _ Interface = &Identity{}
var _ bootstrap.Entity = &Identity{}

type Identity struct {
	// all the successive version of the identity
	versions []*version
}

func NewIdentity(repo repository.RepoClock, name string, email string) (*Identity, error) {
	return NewIdentityFull(repo, name, email, "", "", nil)
}

func NewIdentityFull(repo repository.RepoClock, name string, email string, login string, avatarUrl string, keys []*Key) (*Identity, error) {
	v, err := newVersion(repo, name, email, login, avatarUrl, keys)
	if err != nil {
		return nil, err
	}
	return &Identity{
		versions: []*version{v},
	}, nil
}

// NewFromGitUser will query the repository for user detail and
// build the corresponding Identity
func NewFromGitUser(repo repository.ClockedRepo) (*Identity, error) {
	name, err := repo.GetUserName()
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, errors.New("user name is not configured in git yet. Please use `git config --global user.name \"John Doe\"`")
	}

	email, err := repo.GetUserEmail()
	if err != nil {
		return nil, err
	}
	if email == "" {
		return nil, errors.New("user name is not configured in git yet. Please use `git config --global user.email johndoe@example.com`")
	}

	return NewIdentity(repo, name, email)
}

// MarshalJSON will only serialize the id
func (i *Identity) MarshalJSON() ([]byte, error) {
	return json.Marshal(&IdentityStub{
		id: i.Id(),
	})
}

// UnmarshalJSON will only read the id
// Users of this package are expected to run Load() to load
// the remaining data from the identities data in git.
func (i *Identity) UnmarshalJSON(data []byte) error {
	panic("identity should be loaded with identity.UnmarshalJSON")
}

// ReadLocal load a local Identity from the identities data available in git
func ReadLocal(repo repository.Repo, id bootstrap.Id) (*Identity, error) {
	ref := fmt.Sprintf("%s%s", identityRefPattern, id)
	return read(repo, ref)
}

// ReadRemote load a remote Identity from the identities data available in git
func ReadRemote(repo repository.Repo, remote string, id string) (*Identity, error) {
	ref := fmt.Sprintf(identityRemoteRefPattern, remote) + id
	return read(repo, ref)
}

// read will load and parse an identity from git
func read(repo repository.Repo, ref string) (*Identity, error) {
	id := bootstrap.RefToId(ref)

	if err := id.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid ref")
	}

	hashes, err := repo.ListCommits(ref)
	if err != nil {
		return nil, bootstrap.NewErrNotFound(Typename)
	}
	if len(hashes) == 0 {
		return nil, fmt.Errorf("empty identity")
	}

	i := &Identity{}

	for _, hash := range hashes {
		entries, err := repo.ReadTree(hash)
		if err != nil {
			return nil, errors.Wrap(err, "can't list git tree entries")
		}
		if len(entries) != 1 {
			return nil, fmt.Errorf("invalid identity data at hash %s", hash)
		}

		entry := entries[0]
		if entry.Name != versionEntryName {
			return nil, fmt.Errorf("invalid identity data at hash %s", hash)
		}

		data, err := repo.ReadData(entry.Hash)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read git blob data")
		}

		var version version
		err = json.Unmarshal(data, &version)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to decode Identity version json %s", hash)
		}

		// tag the version with the commit hash
		version.commitHash = hash

		i.versions = append(i.versions, &version)
	}

	if id != i.versions[0].Id() {
		return nil, fmt.Errorf("identity ID doesn't math the first version ID")
	}

	return i, nil
}

// ListLocalIds list all the available local identity ids
func ListLocalIds(repo repository.Repo) ([]bootstrap.Id, error) {
	refs, err := repo.ListRefs(identityRefPattern)
	if err != nil {
		return nil, err
	}

	return bootstrap.RefsToIds(refs), nil
}

// ReadAllLocal read and parse all local Identity
func ReadAllLocal(repo repository.ClockedRepo) <-chan bootstrap.StreamedEntity[*Identity] {
	return readAll(repo, identityRefPattern)
}

// ReadAllRemote read and parse all remote Identity for a given remote
func ReadAllRemote(repo repository.ClockedRepo, remote string) <-chan bootstrap.StreamedEntity[*Identity] {
	refPrefix := fmt.Sprintf(identityRemoteRefPattern, remote)
	return readAll(repo, refPrefix)
}

// readAll read and parse all available bug with a given ref prefix
func readAll(repo repository.ClockedRepo, refPrefix string) <-chan bootstrap.StreamedEntity[*Identity] {
	out := make(chan bootstrap.StreamedEntity[*Identity])

	go func() {
		defer close(out)

		refs, err := repo.ListRefs(refPrefix)
		if err != nil {
			out <- bootstrap.StreamedEntity[*Identity]{Err: err}
			return
		}

		total := int64(len(refs))
		current := int64(1)

		for _, ref := range refs {
			i, err := read(repo, ref)

			if err != nil {
				out <- bootstrap.StreamedEntity[*Identity]{Err: err}
				return
			}

			out <- bootstrap.StreamedEntity[*Identity]{
				Entity:        i,
				CurrentEntity: current,
				TotalEntities: total,
			}
			current++
		}
	}()

	return out
}

type Mutator struct {
	Name      string
	Login     string
	Email     string
	AvatarUrl string
	Keys      []*Key
}

// Mutate allow to create a new version of the Identity in one go
func (i *Identity) Mutate(repo repository.RepoClock, f func(orig *Mutator)) error {
	copyKeys := func(keys []*Key) []*Key {
		result := make([]*Key, len(keys))
		for i, key := range keys {
			result[i] = key.Clone()
		}
		return result
	}

	orig := Mutator{
		Name:      i.Name(),
		Email:     i.Email(),
		Login:     i.Login(),
		AvatarUrl: i.AvatarUrl(),
		Keys:      copyKeys(i.Keys()),
	}
	mutated := orig
	mutated.Keys = copyKeys(orig.Keys)

	f(&mutated)

	if reflect.DeepEqual(orig, mutated) {
		return nil
	}

	v, err := newVersion(repo,
		mutated.Name,
		mutated.Email,
		mutated.Login,
		mutated.AvatarUrl,
		mutated.Keys,
	)
	if err != nil {
		return err
	}

	i.versions = append(i.versions, v)
	return nil
}

// Commit write the identity into the Repository. In particular, this ensures that
// the Id is properly set.
func (i *Identity) Commit(repo repository.ClockedRepo) error {
	if !i.NeedCommit() {
		return fmt.Errorf("can't commit an identity with no pending version")
	}

	if err := i.Validate(); err != nil {
		return errors.Wrap(err, "can't commit an identity with invalid data")
	}

	var lastCommit repository.Hash
	for _, v := range i.versions {
		if v.commitHash != "" {
			lastCommit = v.commitHash
			// ignore already commit versions
			continue
		}

		blobHash, err := v.Write(repo)
		if err != nil {
			return err
		}

		// Make a git tree referencing the blob
		tree := []repository.TreeEntry{
			{ObjectType: repository.Blob, Hash: blobHash, Name: versionEntryName},
		}

		treeHash, err := repo.StoreTree(tree)
		if err != nil {
			return err
		}

		var commitHash repository.Hash
		if lastCommit != "" {
			commitHash, err = repo.StoreCommit(treeHash, lastCommit)
		} else {
			commitHash, err = repo.StoreCommit(treeHash)
		}
		if err != nil {
			return err
		}

		lastCommit = commitHash
		v.commitHash = commitHash
	}

	ref := fmt.Sprintf("%s%s", identityRefPattern, i.Id().String())
	return repo.UpdateRef(ref, lastCommit)
}

func (i *Identity) CommitAsNeeded(repo repository.ClockedRepo) error {
	if !i.NeedCommit() {
		return nil
	}
	return i.Commit(repo)
}

func (i *Identity) NeedCommit() bool {
	for _, v := range i.versions {
		if v.commitHash == "" {
			return true
		}
	}

	return false
}

// Merge will merge a different version of the same Identity
//
// To make sure that an Identity history can't be altered, a strict fast-forward
// only policy is applied here. As an Identity should be tied to a single user, this
// should work in practice, but it does leave a possibility that a user would edit his
// Identity from two different repo concurrently and push the changes in a non-centralized
// network of repositories. In this case, it would result in some repo accepting one
// version and some other accepting another, preventing the network in general to converge
// to the same result. This would create a sort of partition of the network, and manual
// cleaning would be required.
//
// An alternative approach would be to have a determinist rebase:
//   - any commits present in both local and remote version would be kept, never changed.
//   - newer commits would be merged in a linear chain of commits, ordered based on the
//     Lamport time
//
// However, this approach leave the possibility, in the case of a compromised crypto keys,
// of forging a new version with a bogus Lamport time to be inserted before a legit version,
// invalidating the correct version and hijacking the Identity. There would only be a short
// period of time when this would be possible (before the network converge) but I'm not
// confident enough to implement that. I choose the strict fast-forward only approach,
// despite its potential problem with two different version as mentioned above.
func (i *Identity) Merge(repo repository.Repo, other *Identity) (bool, error) {
	if i.Id() != other.Id() {
		return false, errors.New("merging unrelated identities is not supported")
	}

	modified := false
	var lastCommit repository.Hash
	for j, otherVersion := range other.versions {
		// if there is more version in other, take them
		if len(i.versions) == j {
			i.versions = append(i.versions, otherVersion)
			lastCommit = otherVersion.commitHash
			modified = true
		}

		// we have a non fast-forward merge.
		// as explained in the doc above, refusing to merge
		if i.versions[j].commitHash != otherVersion.commitHash {
			return false, ErrNonFastForwardMerge
		}
	}

	if modified {
		err := repo.UpdateRef(identityRefPattern+i.Id().String(), lastCommit)
		if err != nil {
			return false, err
		}
	}

	return false, nil
}

// Validate check if the Identity data is valid
func (i *Identity) Validate() error {
	lastTimes := make(map[string]lamport.Time)

	if len(i.versions) == 0 {
		return fmt.Errorf("no version")
	}

	for _, v := range i.versions {
		if err := v.Validate(); err != nil {
			return err
		}

		// check for always increasing lamport time
		// check that a new version didn't drop a clock
		for name, previous := range lastTimes {
			if now, ok := v.times[name]; ok {
				if now < previous {
					return fmt.Errorf("non-chronological lamport clock %s (%d --> %d)", name, previous, now)
				}
			} else {
				return fmt.Errorf("version has less lamport clocks than before (missing %s)", name)
			}
		}

		for name, now := range v.times {
			lastTimes[name] = now
		}
	}

	return nil
}

func (i *Identity) lastVersion() *version {
	if len(i.versions) <= 0 {
		panic("no version at all")
	}

	return i.versions[len(i.versions)-1]
}

// Id return the Identity identifier
func (i *Identity) Id() bootstrap.Id {
	// id is the id of the first version
	return i.versions[0].Id()
}

// Name return the last version of the name
func (i *Identity) Name() string {
	return i.lastVersion().name
}

// DisplayName return a non-empty string to display, representing the
// identity, based on the non-empty values.
func (i *Identity) DisplayName() string {
	switch {
	case i.Name() == "" && i.Login() != "":
		return i.Login()
	case i.Name() != "" && i.Login() == "":
		return i.Name()
	case i.Name() != "" && i.Login() != "":
		return fmt.Sprintf("%s (%s)", i.Name(), i.Login())
	}

	panic("invalid person data")
}

// Email return the last version of the email
func (i *Identity) Email() string {
	return i.lastVersion().email
}

// Login return the last version of the login
func (i *Identity) Login() string {
	return i.lastVersion().login
}

// AvatarUrl return the last version of the Avatar URL
func (i *Identity) AvatarUrl() string {
	return i.lastVersion().avatarURL
}

// Keys return the last version of the valid keys
func (i *Identity) Keys() []*Key {
	return i.lastVersion().keys
}

// SigningKey return the key that should be used to sign new messages. If no key is available, return nil.
func (i *Identity) SigningKey(repo repository.RepoKeyring) (*Key, error) {
	keys := i.Keys()
	for _, key := range keys {
		err := key.ensurePrivateKey(repo)
		if err == errNoPrivateKey {
			continue
		}
		if err != nil {
			return nil, err
		}
		return key, nil
	}
	return nil, nil
}

// ValidKeysAtTime return the set of keys valid at a given lamport time
func (i *Identity) ValidKeysAtTime(clockName string, time lamport.Time) []*Key {
	var result []*Key

	var lastTime lamport.Time
	for _, v := range i.versions {
		refTime, ok := v.times[clockName]
		if !ok {
			refTime = lastTime
		}
		lastTime = refTime

		if refTime > time {
			return result
		}

		result = v.keys
	}

	return result
}

// LastModification return the timestamp at which the last version of the identity became valid.
func (i *Identity) LastModification() timestamp.Timestamp {
	return timestamp.Timestamp(i.lastVersion().unixTime)
}

// LastModificationLamports return the lamport times at which the last version of the identity became valid.
func (i *Identity) LastModificationLamports() map[string]lamport.Time {
	return i.lastVersion().times
}

// IsProtected return true if the chain of git commits started to be signed.
// If that's the case, only signed commit with a valid key for this identity can be added.
func (i *Identity) IsProtected() bool {
	// Todo
	return false
}

// SetMetadata store arbitrary metadata along the last not-commit version.
// If the version has been commit to git already, a new identical version is added and will need to be
// commit.
func (i *Identity) SetMetadata(key string, value string) {
	// once commit, data is immutable so we create a new version
	if i.lastVersion().commitHash != "" {
		i.versions = append(i.versions, i.lastVersion().Clone())
	}
	// if Id() has been called, we can't change the first version anymore, so we create a new version
	if len(i.versions) == 1 && i.versions[0].id != bootstrap.UnsetId && i.versions[0].id != "" {
		i.versions = append(i.versions, i.lastVersion().Clone())
	}

	i.lastVersion().SetMetadata(key, value)
}

// ImmutableMetadata return all metadata for this Identity, accumulated from each version.
// If multiple value are found, the first defined takes precedence.
func (i *Identity) ImmutableMetadata() map[string]string {
	metadata := make(map[string]string)

	for _, version := range i.versions {
		for key, value := range version.metadata {
			if _, has := metadata[key]; !has {
				metadata[key] = value
			}
		}
	}

	return metadata
}

// MutableMetadata return all metadata for this Identity, accumulated from each version.
// If multiple value are found, the last defined takes precedence.
func (i *Identity) MutableMetadata() map[string]string {
	metadata := make(map[string]string)

	for _, version := range i.versions {
		for key, value := range version.metadata {
			metadata[key] = value
		}
	}

	return metadata
}
