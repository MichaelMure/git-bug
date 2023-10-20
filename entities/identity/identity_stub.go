package identity

import (
	"encoding/json"

	bootstrap "github.com/MichaelMure/git-bug/entity/boostrap"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
	"github.com/MichaelMure/git-bug/util/timestamp"
)

var _ Interface = &IdentityStub{}

// IdentityStub is an almost empty Identity, holding only the id.
// When a normal Identity is serialized into JSON, only the id is serialized.
// All the other data are stored in git in a chain of commit + a ref.
// When this JSON is deserialized, an IdentityStub is returned instead, to be replaced
// later by the proper Identity, loaded from the Repo.
type IdentityStub struct {
	id bootstrap.Id
}

func (i *IdentityStub) MarshalJSON() ([]byte, error) {
	// TODO: add a type marker
	return json.Marshal(struct {
		Id bootstrap.Id `json:"id"`
	}{
		Id: i.id,
	})
}

func (i *IdentityStub) UnmarshalJSON(data []byte) error {
	aux := struct {
		Id bootstrap.Id `json:"id"`
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	i.id = aux.Id

	return nil
}

// Id return the Identity identifier
func (i *IdentityStub) Id() bootstrap.Id {
	return i.id
}

func (IdentityStub) Name() string {
	panic("identities needs to be properly loaded with identity.ReadLocal()")
}

func (IdentityStub) DisplayName() string {
	panic("identities needs to be properly loaded with identity.ReadLocal()")
}

func (IdentityStub) Email() string {
	panic("identities needs to be properly loaded with identity.ReadLocal()")
}

func (IdentityStub) Login() string {
	panic("identities needs to be properly loaded with identity.ReadLocal()")
}

func (IdentityStub) AvatarUrl() string {
	panic("identities needs to be properly loaded with identity.ReadLocal()")
}

func (IdentityStub) Keys() []*Key {
	panic("identities needs to be properly loaded with identity.ReadLocal()")
}

func (i *IdentityStub) SigningKey(repo repository.RepoKeyring) (*Key, error) {
	panic("identities needs to be properly loaded with identity.ReadLocal()")
}

func (IdentityStub) ValidKeysAtTime(_ string, _ lamport.Time) []*Key {
	panic("identities needs to be properly loaded with identity.ReadLocal()")
}

func (i *IdentityStub) LastModification() timestamp.Timestamp {
	panic("identities needs to be properly loaded with identity.ReadLocal()")
}

func (i *IdentityStub) LastModificationLamports() map[string]lamport.Time {
	panic("identities needs to be properly loaded with identity.ReadLocal()")
}

func (IdentityStub) IsProtected() bool {
	panic("identities needs to be properly loaded with identity.ReadLocal()")
}

func (IdentityStub) Validate() error {
	panic("identities needs to be properly loaded with identity.ReadLocal()")
}

func (i *IdentityStub) NeedCommit() bool {
	return false
}
