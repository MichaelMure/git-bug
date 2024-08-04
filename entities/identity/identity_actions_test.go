package identity

import (
	"testing"

	"github.com/stretchr/testify/require"

	bootstrap "github.com/MichaelMure/git-bug/entity/boostrap"
	"github.com/MichaelMure/git-bug/repository"
)

func TestIdentityPushPull(t *testing.T) {
	repoA, repoB, _ := repository.SetupGoGitReposAndRemote(t)

	identity1, err := NewIdentity(repoA, "name1", "email1")
	require.NoError(t, err)
	err = identity1.Commit(repoA)
	require.NoError(t, err)

	// A --> remote --> B
	_, err = Push(repoA, "origin")
	require.NoError(t, err)

	err = Pull(repoB, "origin")
	require.NoError(t, err)

	identities := allIdentities(t, ReadAllLocal(repoB))

	if len(identities) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	// B --> remote --> A
	identity2, err := NewIdentity(repoB, "name2", "email2")
	require.NoError(t, err)
	err = identity2.Commit(repoB)
	require.NoError(t, err)

	_, err = Push(repoB, "origin")
	require.NoError(t, err)

	err = Pull(repoA, "origin")
	require.NoError(t, err)

	identities = allIdentities(t, ReadAllLocal(repoA))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}

	// Update both

	err = identity1.Mutate(repoA, func(orig *Mutator) {
		orig.Name = "name1b"
		orig.Email = "email1b"
	})
	require.NoError(t, err)
	err = identity1.Commit(repoA)
	require.NoError(t, err)

	err = identity2.Mutate(repoB, func(orig *Mutator) {
		orig.Name = "name2b"
		orig.Email = "email2b"
	})
	require.NoError(t, err)
	err = identity2.Commit(repoB)
	require.NoError(t, err)

	//  A --> remote --> B

	_, err = Push(repoA, "origin")
	require.NoError(t, err)

	err = Pull(repoB, "origin")
	require.NoError(t, err)

	identities = allIdentities(t, ReadAllLocal(repoB))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}

	// B --> remote --> A

	_, err = Push(repoB, "origin")
	require.NoError(t, err)

	err = Pull(repoA, "origin")
	require.NoError(t, err)

	identities = allIdentities(t, ReadAllLocal(repoA))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}

	// Concurrent update

	err = identity1.Mutate(repoA, func(orig *Mutator) {
		orig.Name = "name1c"
		orig.Email = "email1c"
	})
	require.NoError(t, err)
	err = identity1.Commit(repoA)
	require.NoError(t, err)

	identity1B, err := ReadLocal(repoB, identity1.Id())
	require.NoError(t, err)

	err = identity1B.Mutate(repoB, func(orig *Mutator) {
		orig.Name = "name1concurrent"
		orig.Email = "name1concurrent"
	})
	require.NoError(t, err)
	err = identity1B.Commit(repoB)
	require.NoError(t, err)

	//  A --> remote --> B

	_, err = Push(repoA, "origin")
	require.NoError(t, err)

	// Pulling a non-fast-forward update should fail
	err = Pull(repoB, "origin")
	require.Error(t, err)

	identities = allIdentities(t, ReadAllLocal(repoB))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}

	// B --> remote --> A

	// Pushing a non-fast-forward update should fail
	_, err = Push(repoB, "origin")
	require.Error(t, err)

	err = Pull(repoA, "origin")
	require.NoError(t, err)

	identities = allIdentities(t, ReadAllLocal(repoA))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}
}

func allIdentities(t testing.TB, identities <-chan bootstrap.StreamedEntity[*Identity]) []*Identity {
	var result []*Identity
	for streamed := range identities {
		if streamed.Err != nil {
			t.Fatal(streamed.Err)
		}
		result = append(result, streamed.Entity)
	}
	return result
}
