package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

const (
	keyringKeyPrefix     = "auth-"
	keyringKeyKind       = "kind"
	keyringKeyTarget     = "target"
	keyringKeyCreateTime = "createtime"
	keyringKeySalt       = "salt"
	keyringKeyPrefixMeta = "meta."

	MetaKeyLogin    = "login"
	MetaKeyBaseURL  = "base-url"
	MetaKeyIdentity = "identity"
)

type CredentialKind string

const (
	KindToken         CredentialKind = "token"
	KindLogin         CredentialKind = "login"
	KindLoginPassword CredentialKind = "login-password"
)

var ErrCredentialNotExist = errors.New("credential doesn't exist")

func NewErrMultipleMatchCredential(matching []entity.Id) *entity.ErrMultipleMatch {
	return entity.NewErrMultipleMatch("credential", matching)
}

type Credential interface {
	ID() entity.Id
	Kind() CredentialKind
	Target() string
	CreateTime() time.Time
	Salt() []byte
	Validate() error

	Metadata() map[string]string
	GetMetadata(key string) (string, bool)
	SetMetadata(key string, value string)

	// Return all the specific properties of the credential that need to be saved into the configuration.
	// This does not include Target, Kind, CreateTime, Metadata or Salt.
	toConfig() map[string]string
}

// Load loads a credential from the repo config
func LoadWithId(repo repository.RepoKeyring, id entity.Id) (Credential, error) {
	key := fmt.Sprintf("%s%s", keyringKeyPrefix, id)

	item, err := repo.Keyring().Get(key)
	if err == repository.ErrKeyringKeyNotFound {
		return nil, ErrCredentialNotExist
	}
	if err != nil {
		return nil, err
	}

	return decode(item)
}

// LoadWithPrefix load a credential from the repo config with a prefix
func LoadWithPrefix(repo repository.RepoKeyring, prefix string) (Credential, error) {
	keys, err := repo.Keyring().Keys()
	if err != nil {
		return nil, err
	}

	// preallocate but empty
	matching := make([]Credential, 0, 5)

	for _, key := range keys {
		if !strings.HasPrefix(key, keyringKeyPrefix+prefix) {
			continue
		}

		item, err := repo.Keyring().Get(key)
		if err != nil {
			return nil, err
		}

		cred, err := decode(item)
		if err != nil {
			return nil, err
		}

		matching = append(matching, cred)
	}

	if len(matching) > 1 {
		ids := make([]entity.Id, len(matching))
		for i, cred := range matching {
			ids[i] = cred.ID()
		}
		return nil, NewErrMultipleMatchCredential(ids)
	}

	if len(matching) == 0 {
		return nil, ErrCredentialNotExist
	}

	return matching[0], nil
}

// decode is a helper to construct a Credential from the keyring Item
func decode(item repository.Item) (Credential, error) {
	data := make(map[string]string)

	err := json.Unmarshal(item.Data, &data)
	if err != nil {
		return nil, err
	}

	var cred Credential
	switch CredentialKind(data[keyringKeyKind]) {
	case KindToken:
		cred, err = NewTokenFromConfig(data)
	case KindLogin:
		cred, err = NewLoginFromConfig(data)
	case KindLoginPassword:
		cred, err = NewLoginPasswordFromConfig(data)
	default:
		return nil, fmt.Errorf("unknown credential type \"%s\"", data[keyringKeyKind])
	}

	if err != nil {
		return nil, fmt.Errorf("loading credential: %v", err)
	}

	return cred, nil
}

// List load all existing credentials
func List(repo repository.RepoKeyring, opts ...ListOption) ([]Credential, error) {
	keys, err := repo.Keyring().Keys()
	if err != nil {
		return nil, err
	}

	matcher := matcher(opts)

	var credentials []Credential
	for _, key := range keys {
		if !strings.HasPrefix(key, keyringKeyPrefix) {
			continue
		}

		item, err := repo.Keyring().Get(key)
		if err != nil {
			// skip unreadable items, nothing much we can do for them anyway
			continue
		}

		cred, err := decode(item)
		if err != nil {
			return nil, err
		}
		if matcher.Match(cred) {
			credentials = append(credentials, cred)
		}
	}

	return credentials, nil
}

// IdExist return whether a credential id exist or not
func IdExist(repo repository.RepoKeyring, id entity.Id) bool {
	_, err := LoadWithId(repo, id)
	return err == nil
}

// PrefixExist return whether a credential id prefix exist or not
func PrefixExist(repo repository.RepoKeyring, prefix string) bool {
	_, err := LoadWithPrefix(repo, prefix)
	return err == nil
}

// Store stores a credential in the global git config
func Store(repo repository.RepoKeyring, cred Credential) error {
	if len(cred.Salt()) != 16 {
		panic("credentials need to be salted")
	}

	confs := cred.toConfig()

	confs[keyringKeyKind] = string(cred.Kind())
	confs[keyringKeyTarget] = cred.Target()
	confs[keyringKeyCreateTime] = strconv.Itoa(int(cred.CreateTime().Unix()))
	confs[keyringKeySalt] = base64.StdEncoding.EncodeToString(cred.Salt())

	for key, val := range cred.Metadata() {
		confs[keyringKeyPrefixMeta+key] = val
	}

	data, err := json.Marshal(confs)
	if err != nil {
		return err
	}

	return repo.Keyring().Set(repository.Item{
		Key:  keyringKeyPrefix + cred.ID().String(),
		Data: data,
	})
}

// Remove removes a credential from the global git config
func Remove(repo repository.RepoKeyring, id entity.Id) error {
	return repo.Keyring().Remove(keyringKeyPrefix + id.String())
}

/*
 * Sorting
 */

type ById []Credential

func (b ById) Len() int {
	return len(b)
}

func (b ById) Less(i, j int) bool {
	return b[i].ID() < b[j].ID()
}

func (b ById) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
