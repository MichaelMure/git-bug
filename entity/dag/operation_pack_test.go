package dag

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

func TestOperationPackReadWrite(t *testing.T) {
	repo, author, _, resolver, def := makeTestContext()

	opp := &operationPack{
		Author: author,
		Operations: []Operation{
			newOp1(author, "foo"),
			newOp2(author, "bar"),
		},
		CreateTime: 123,
		EditTime:   456,
	}

	commitHash, err := opp.Write(def, repo)
	require.NoError(t, err)

	commit, err := repo.ReadCommit(commitHash)
	require.NoError(t, err)

	opp2, err := readOperationPack(def, repo, resolver, commit)
	require.NoError(t, err)

	for _, op := range opp.Operations {
		// force the creation of the id
		op.Id()
	}
	require.Equal(t, opp, opp2)
}

func TestOperationPackSignedReadWrite(t *testing.T) {
	type makerFn func() (repository.ClockedRepo, identity.Interface, identity.Interface, entity.Resolvers, Definition)

	for _, maker := range []makerFn{
		makeTestContext,
		func() (repository.ClockedRepo, identity.Interface, identity.Interface, entity.Resolvers, Definition) {
			return makeTestContextGoGit(t)
		},
	} {
		repo, author, _, resolver, def := maker()

		err := author.(*identity.Identity).Mutate(repo, func(orig *identity.Mutator) {
			orig.Keys = append(orig.Keys, identity.GenerateKey())
		})
		require.NoError(t, err)

		opp := &operationPack{
			Author: author,
			Operations: []Operation{
				newOp1(author, "foo"),
				newOp2(author, "bar"),
			},
			CreateTime: 123,
			EditTime:   456,
		}

		commitHash, err := opp.Write(def, repo)
		require.NoError(t, err)

		commit, err := repo.ReadCommit(commitHash)
		require.NoError(t, err)

		opp2, err := readOperationPack(def, repo, resolver, commit)
		require.NoError(t, err)

		for _, op := range opp.Operations {
			// force the creation of the id
			op.Id()
		}
		require.Equal(t, opp, opp2)
	}
}

func TestOperationPackFiles(t *testing.T) {
	repo, author, _, resolver, def := makeTestContext()

	blobHash1, err := repo.StoreData(randomData())
	require.NoError(t, err)

	blobHash2, err := repo.StoreData(randomData())
	require.NoError(t, err)

	opp := &operationPack{
		Author: author,
		Operations: []Operation{
			newOp1(author, "foo", blobHash1, blobHash2),
			newOp1(author, "foo", blobHash2),
		},
		CreateTime: 123,
		EditTime:   456,
	}

	commitHash, err := opp.Write(def, repo)
	require.NoError(t, err)

	commit, err := repo.ReadCommit(commitHash)
	require.NoError(t, err)

	opp2, err := readOperationPack(def, repo, resolver, commit)
	require.NoError(t, err)

	for _, op := range opp.Operations {
		// force the creation of the id
		op.Id()
	}
	require.Equal(t, opp, opp2)

	require.ElementsMatch(t, opp2.Operations[0].(entity.OperationWithFiles).GetFiles(), []repository.Hash{
		blobHash1,
		blobHash2,
	})
	require.ElementsMatch(t, opp2.Operations[1].(entity.OperationWithFiles).GetFiles(), []repository.Hash{
		blobHash2,
	})

	tree, err := repo.ReadTree(commit.TreeHash)
	require.NoError(t, err)

	extraTreeHash, ok := repository.SearchTreeEntry(tree, extraEntryName)
	require.True(t, ok)

	extraTree, err := repo.ReadTree(extraTreeHash.Hash)
	require.NoError(t, err)
	require.ElementsMatch(t, extraTree, []repository.TreeEntry{
		{
			ObjectType: repository.Blob,
			Hash:       blobHash1,
			Name:       "file0",
		},
		{
			ObjectType: repository.Blob,
			Hash:       blobHash2,
			Name:       "file1",
		},
	})
}

func randomData() []byte {
	var letterRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 32)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return b
}
