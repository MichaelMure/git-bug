package identity

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	bootstrap "github.com/MichaelMure/git-bug/entity/boostrap"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
)

// Test the commit and load of an Identity with multiple versions
func TestIdentityCommitLoad(t *testing.T) {
	repo := makeIdentityTestRepo(t)

	// single version

	identity, err := NewIdentity(repo, "René Descartes", "rene.descartes@example.com")
	require.NoError(t, err)

	idBeforeCommit := identity.Id()

	err = identity.Commit(repo)
	require.NoError(t, err)

	commitsAreSet(t, identity)
	require.NotEmpty(t, identity.Id())
	require.Equal(t, idBeforeCommit, identity.Id())
	require.Equal(t, idBeforeCommit, identity.versions[0].Id())

	loaded, err := ReadLocal(repo, identity.Id())
	require.NoError(t, err)
	commitsAreSet(t, loaded)
	require.Equal(t, identity, loaded)

	// multiple versions

	identity, err = NewIdentityFull(repo, "René Descartes", "rene.descartes@example.com", "", "", []*Key{generatePublicKey()})
	require.NoError(t, err)

	idBeforeCommit = identity.Id()

	err = identity.Mutate(repo, func(orig *Mutator) {
		orig.Keys = []*Key{generatePublicKey()}
	})
	require.NoError(t, err)

	err = identity.Mutate(repo, func(orig *Mutator) {
		orig.Keys = []*Key{generatePublicKey()}
	})
	require.NoError(t, err)

	require.Equal(t, idBeforeCommit, identity.Id())

	err = identity.Commit(repo)
	require.NoError(t, err)

	commitsAreSet(t, identity)
	require.NotEmpty(t, identity.Id())
	require.Equal(t, idBeforeCommit, identity.Id())
	require.Equal(t, idBeforeCommit, identity.versions[0].Id())

	loaded, err = ReadLocal(repo, identity.Id())
	require.NoError(t, err)
	commitsAreSet(t, loaded)
	require.Equal(t, identity, loaded)

	// add more version

	err = identity.Mutate(repo, func(orig *Mutator) {
		orig.Email = "rene@descartes.com"
		orig.Keys = []*Key{generatePublicKey()}
	})
	require.NoError(t, err)

	err = identity.Mutate(repo, func(orig *Mutator) {
		orig.Email = "rene@descartes.com"
		orig.Keys = []*Key{generatePublicKey(), generatePublicKey()}
	})
	require.NoError(t, err)

	err = identity.Commit(repo)
	require.NoError(t, err)

	commitsAreSet(t, identity)
	require.NotEmpty(t, identity.Id())
	require.Equal(t, idBeforeCommit, identity.Id())
	require.Equal(t, idBeforeCommit, identity.versions[0].Id())

	loaded, err = ReadLocal(repo, identity.Id())
	require.NoError(t, err)
	commitsAreSet(t, loaded)
	require.Equal(t, identity, loaded)
}

func TestIdentityMutate(t *testing.T) {
	repo := makeIdentityTestRepo(t)

	identity, err := NewIdentity(repo, "René Descartes", "rene.descartes@example.com")
	require.NoError(t, err)

	require.Len(t, identity.versions, 1)

	err = identity.Mutate(repo, func(orig *Mutator) {
		orig.Email = "rene@descartes.fr"
		orig.Name = "René"
		orig.Login = "rene"
	})
	require.NoError(t, err)

	require.Len(t, identity.versions, 2)
	require.Equal(t, identity.Email(), "rene@descartes.fr")
	require.Equal(t, identity.Name(), "René")
	require.Equal(t, identity.Login(), "rene")
}

func commitsAreSet(t *testing.T, identity *Identity) {
	for _, version := range identity.versions {
		require.NotEmpty(t, version.commitHash)
	}
}

// Test that the correct crypto keys are returned for a given lamport time
func TestIdentity_ValidKeysAtTime(t *testing.T) {
	pubKeyA := generatePublicKey()
	pubKeyB := generatePublicKey()
	pubKeyC := generatePublicKey()
	pubKeyD := generatePublicKey()
	pubKeyE := generatePublicKey()

	identity := Identity{
		versions: []*version{
			{
				times: map[string]lamport.Time{"foo": 100},
				keys:  []*Key{pubKeyA},
			},
			{
				times: map[string]lamport.Time{"foo": 200},
				keys:  []*Key{pubKeyB},
			},
			{
				times: map[string]lamport.Time{"foo": 201},
				keys:  []*Key{pubKeyC},
			},
			{
				times: map[string]lamport.Time{"foo": 201},
				keys:  []*Key{pubKeyD},
			},
			{
				times: map[string]lamport.Time{"foo": 300},
				keys:  []*Key{pubKeyE},
			},
		},
	}

	require.Nil(t, identity.ValidKeysAtTime("foo", 10))
	require.Equal(t, identity.ValidKeysAtTime("foo", 100), []*Key{pubKeyA})
	require.Equal(t, identity.ValidKeysAtTime("foo", 140), []*Key{pubKeyA})
	require.Equal(t, identity.ValidKeysAtTime("foo", 200), []*Key{pubKeyB})
	require.Equal(t, identity.ValidKeysAtTime("foo", 201), []*Key{pubKeyD})
	require.Equal(t, identity.ValidKeysAtTime("foo", 202), []*Key{pubKeyD})
	require.Equal(t, identity.ValidKeysAtTime("foo", 300), []*Key{pubKeyE})
	require.Equal(t, identity.ValidKeysAtTime("foo", 3000), []*Key{pubKeyE})
}

// Test the immutable or mutable metadata search
func TestMetadata(t *testing.T) {
	repo := makeIdentityTestRepo(t)

	identity, err := NewIdentity(repo, "René Descartes", "rene.descartes@example.com")
	require.NoError(t, err)

	identity.SetMetadata("key1", "value1")
	assertHasKeyValue(t, identity.ImmutableMetadata(), "key1", "value1")
	assertHasKeyValue(t, identity.MutableMetadata(), "key1", "value1")

	err = identity.Commit(repo)
	require.NoError(t, err)

	assertHasKeyValue(t, identity.ImmutableMetadata(), "key1", "value1")
	assertHasKeyValue(t, identity.MutableMetadata(), "key1", "value1")

	// try override
	err = identity.Mutate(repo, func(orig *Mutator) {
		orig.Email = "rene@descartes.fr"
	})
	require.NoError(t, err)

	identity.SetMetadata("key1", "value2")
	assertHasKeyValue(t, identity.ImmutableMetadata(), "key1", "value1")
	assertHasKeyValue(t, identity.MutableMetadata(), "key1", "value2")

	err = identity.Commit(repo)
	require.NoError(t, err)

	// reload
	loaded, err := ReadLocal(repo, identity.Id())
	require.NoError(t, err)

	assertHasKeyValue(t, loaded.ImmutableMetadata(), "key1", "value1")
	assertHasKeyValue(t, loaded.MutableMetadata(), "key1", "value2")

	// set metadata after commit
	versionCount := len(identity.versions)
	identity.SetMetadata("foo", "bar")
	require.True(t, identity.NeedCommit())
	require.Len(t, identity.versions, versionCount+1)

	err = identity.Commit(repo)
	require.NoError(t, err)
	require.Len(t, identity.versions, versionCount+1)
}

func assertHasKeyValue(t *testing.T, metadata map[string]string, key, value string) {
	val, ok := metadata[key]
	require.True(t, ok)
	require.Equal(t, val, value)
}

func TestJSON(t *testing.T) {
	repo := makeIdentityTestRepo(t)

	identity, err := NewIdentity(repo, "René Descartes", "rene.descartes@example.com")
	require.NoError(t, err)

	// commit to make sure we have an Id
	err = identity.Commit(repo)
	require.NoError(t, err)
	require.NotEmpty(t, identity.Id())

	// serialize
	data, err := json.Marshal(identity)
	require.NoError(t, err)

	// deserialize, got a IdentityStub with the same id
	var i Interface
	i, err = UnmarshalJSON(data)
	require.NoError(t, err)
	require.Equal(t, identity.Id(), i.Id())

	// make sure we can load the identity properly
	i, err = ReadLocal(repo, i.Id())
	require.NoError(t, err)
}

func TestIdentityRemove(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(t, false)
	remoteA := repository.CreateGoGitTestRepo(t, true)
	remoteB := repository.CreateGoGitTestRepo(t, true)

	err := repo.AddRemote("remoteA", remoteA.GetLocalRemote())
	require.NoError(t, err)

	err = repo.AddRemote("remoteB", remoteB.GetLocalRemote())
	require.NoError(t, err)

	// generate an identity for testing
	rene, err := NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	err = rene.Commit(repo)
	require.NoError(t, err)

	_, err = Push(repo, "remoteA")
	require.NoError(t, err)

	_, err = Push(repo, "remoteB")
	require.NoError(t, err)

	_, err = Fetch(repo, "remoteA")
	require.NoError(t, err)

	_, err = Fetch(repo, "remoteB")
	require.NoError(t, err)

	err = Remove(repo, rene.Id())
	require.NoError(t, err)

	_, err = ReadLocal(repo, rene.Id())
	require.ErrorAs(t, bootstrap.ErrNotFound{}, err)

	_, err = ReadRemote(repo, "remoteA", string(rene.Id()))
	require.ErrorAs(t, bootstrap.ErrNotFound{}, err)

	_, err = ReadRemote(repo, "remoteB", string(rene.Id()))
	require.ErrorAs(t, bootstrap.ErrNotFound{}, err)

	ids, err := ListLocalIds(repo)
	require.NoError(t, err)
	require.Len(t, ids, 0)
}
