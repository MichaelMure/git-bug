package identity

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	bootstrap "github.com/MichaelMure/git-bug/entity/boostrap"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
	"github.com/MichaelMure/git-bug/util/text"
)

// 1: original format
// 2: Identity Ids are generated from the first version serialized data instead of from the first git
// commit + Identity hold multiple lamport clocks from other entities, instead of just bug edit
const formatVersion = 2

// version is a complete set of information about an Identity at a point in time.
type version struct {
	name      string
	email     string // as defined in git or from a bridge when importing the identity
	login     string // from a bridge when importing the identity
	avatarURL string

	// The lamport times of the other entities at which this version become effective
	times    map[string]lamport.Time
	unixTime int64

	// The set of keys valid at that time, from this version onward, until they get removed
	// in a new version. This allows to have multiple key for the same identity (e.g. one per
	// device) as well as revoke key.
	keys []*Key

	// mandatory random bytes to ensure a better randomness of the data of the first
	// version of an identity, used to later generate the ID
	// len(Nonce) should be > 20 and < 64 bytes
	// It has no functional purpose and should be ignored.
	// TODO: optional after first version?
	nonce []byte

	// A set of arbitrary key/value to store metadata about a version or about an Identity in general.
	metadata map[string]string

	// Not serialized. Store the version's id in memory.
	id bootstrap.Id
	// Not serialized
	commitHash repository.Hash
}

func newVersion(repo repository.RepoClock, name string, email string, login string, avatarURL string, keys []*Key) (*version, error) {
	clocks, err := repo.AllClocks()
	if err != nil {
		return nil, err
	}

	times := make(map[string]lamport.Time)
	for name, clock := range clocks {
		times[name] = clock.Time()
	}

	return &version{
		id:        bootstrap.UnsetId,
		name:      name,
		email:     email,
		login:     login,
		avatarURL: avatarURL,
		times:     times,
		unixTime:  time.Now().Unix(),
		keys:      keys,
		nonce:     makeNonce(20),
	}, nil
}

type versionJSON struct {
	// Additional field to version the data
	FormatVersion uint `json:"version"`

	Times     map[string]lamport.Time `json:"times"`
	UnixTime  int64                   `json:"unix_time"`
	Name      string                  `json:"name,omitempty"`
	Email     string                  `json:"email,omitempty"`
	Login     string                  `json:"login,omitempty"`
	AvatarUrl string                  `json:"avatar_url,omitempty"`
	Keys      []*Key                  `json:"pub_keys,omitempty"`
	Nonce     []byte                  `json:"nonce"`
	Metadata  map[string]string       `json:"metadata,omitempty"`
}

// Id return the identifier of the version
func (v *version) Id() bootstrap.Id {
	if v.id == "" {
		// something went really wrong
		panic("version's id not set")
	}
	if v.id == bootstrap.UnsetId {
		// This means we are trying to get the version's Id *before* it has been stored.
		// As the Id is computed based on the actual bytes written on the disk, we are going to predict
		// those and then get the Id. This is safe as it will be the exact same code writing on disk later.
		data, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		v.id = bootstrap.DeriveId(data)
	}
	return v.id
}

// Make a deep copy
func (v *version) Clone() *version {
	// copy direct fields
	clone := *v

	// reset some fields
	clone.commitHash = ""
	clone.id = bootstrap.UnsetId

	clone.times = make(map[string]lamport.Time)
	for name, t := range v.times {
		clone.times[name] = t
	}

	clone.keys = make([]*Key, len(v.keys))
	for i, key := range v.keys {
		clone.keys[i] = key.Clone()
	}

	clone.nonce = make([]byte, len(v.nonce))
	copy(clone.nonce, v.nonce)

	// not copying metadata

	return &clone
}

func (v *version) MarshalJSON() ([]byte, error) {
	return json.Marshal(versionJSON{
		FormatVersion: formatVersion,
		Times:         v.times,
		UnixTime:      v.unixTime,
		Name:          v.name,
		Email:         v.email,
		Login:         v.login,
		AvatarUrl:     v.avatarURL,
		Keys:          v.keys,
		Nonce:         v.nonce,
		Metadata:      v.metadata,
	})
}

func (v *version) UnmarshalJSON(data []byte) error {
	var aux versionJSON

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.FormatVersion != formatVersion {
		return bootstrap.NewErrInvalidFormat(aux.FormatVersion, formatVersion)
	}

	v.id = bootstrap.DeriveId(data)
	v.times = aux.Times
	v.unixTime = aux.UnixTime
	v.name = aux.Name
	v.email = aux.Email
	v.login = aux.Login
	v.avatarURL = aux.AvatarUrl
	v.keys = aux.Keys
	v.nonce = aux.Nonce
	v.metadata = aux.Metadata

	return nil
}

func (v *version) Validate() error {
	// time must be set after a commit
	if v.commitHash != "" && v.unixTime == 0 {
		return fmt.Errorf("unix time not set")
	}

	if text.Empty(v.name) && text.Empty(v.login) {
		return fmt.Errorf("either name or login should be set")
	}
	if !text.SafeOneLine(v.name) {
		return fmt.Errorf("name has unsafe characters")
	}

	if !text.SafeOneLine(v.login) {
		return fmt.Errorf("login has unsafe characters")
	}

	if !text.SafeOneLine(v.email) {
		return fmt.Errorf("email has unsafe characters")
	}

	if v.avatarURL != "" && !text.ValidUrl(v.avatarURL) {
		return fmt.Errorf("avatarUrl is not a valid URL")
	}

	if len(v.nonce) > 64 {
		return fmt.Errorf("nonce is too big")
	}
	if len(v.nonce) < 20 {
		return fmt.Errorf("nonce is too small")
	}

	for _, k := range v.keys {
		if err := k.Validate(); err != nil {
			return errors.Wrap(err, "invalid key")
		}
	}

	return nil
}

// Write will serialize and store the version as a git blob and return
// its hash
func (v *version) Write(repo repository.Repo) (repository.Hash, error) {
	// make sure we don't write invalid data
	err := v.Validate()
	if err != nil {
		return "", errors.Wrap(err, "validation error")
	}

	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	hash, err := repo.StoreData(data)
	if err != nil {
		return "", err
	}

	// make sure we set the Id when writing in the repo
	v.id = bootstrap.DeriveId(data)

	return hash, nil
}

func makeNonce(len int) []byte {
	result := make([]byte, len)
	_, err := rand.Read(result)
	if err != nil {
		panic(err)
	}
	return result
}

// SetMetadata store arbitrary metadata about a version or an Identity in general
// If the version has been commit to git already, it won't be overwritten.
// Beware: changing the metadata on a version will change it's ID
func (v *version) SetMetadata(key string, value string) {
	if v.metadata == nil {
		v.metadata = make(map[string]string)
	}
	v.metadata[key] = value
}

// GetMetadata retrieve arbitrary metadata about the version
func (v *version) GetMetadata(key string) (string, bool) {
	val, ok := v.metadata[key]
	return val, ok
}

// AllMetadata return all metadata for this version
func (v *version) AllMetadata() map[string]string {
	return v.metadata
}
