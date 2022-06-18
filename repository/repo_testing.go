package repository

import (
	"math/rand"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/util/lamport"
)

// TODO: add tests for RepoBleve
// TODO: add tests for RepoStorage

// func CleanupTestRepos(repos ...Repo) { // TODO
// 	var firstErr error
// 	for _, repo := range repos {
// 		if repo, ok := repo.(TestedRepo); ok {
// 			err := repo.EraseFromDisk()
// 			if err != nil {
// 				log.Println(err)
// 				if firstErr == nil {
// 					firstErr = err
// 				}
// 			}
// 		}
// 	}

// 	if firstErr != nil {
// 		log.Fatal(firstErr)
// 	}
// }

type RepoCreator func(t TestingT, bare bool) TestedRepo

// type RepoCleaner func(repos ...Repo) // TODO

// Test suite for a Repo implementation
func RepoTest(t *testing.T, creator RepoCreator) {
	for bare, name := range map[bool]string{
		false: "Plain",
		true:  "Bare",
	} {
		t.Run(name, func(t *testing.T) {
			repo := creator(t, bare)

			t.Run("Data", func(t *testing.T) {
				RepoDataTest(t, repo)
				RepoDataSignatureTest(t, repo)
			})

			t.Run("Config", func(t *testing.T) {
				RepoConfigTest(t, repo)
			})

			t.Run("Clocks", func(t *testing.T) {
				RepoClockTest(t, repo)
			})
		})
	}
}

// helper to test a RepoConfig
func RepoConfigTest(t *testing.T, repo RepoConfig) {
	testConfig(t, repo.LocalConfig())
}

// helper to test a RepoData
func RepoDataTest(t *testing.T, repo RepoData) {
	// Blob

	data := randomData()

	blobHash1, err := repo.StoreData(data)
	require.NoError(t, err)
	require.True(t, blobHash1.IsValid())

	blob1Read, err := repo.ReadData(blobHash1)
	require.NoError(t, err)
	require.Equal(t, data, blob1Read)

	// Tree

	blobHash2, err := repo.StoreData(randomData())
	require.NoError(t, err)
	blobHash3, err := repo.StoreData(randomData())
	require.NoError(t, err)

	tree1 := []TreeEntry{
		{
			ObjectType: Blob,
			Hash:       blobHash1,
			Name:       "blob1",
		},
		{
			ObjectType: Blob,
			Hash:       blobHash2,
			Name:       "blob2",
		},
	}

	treeHash1, err := repo.StoreTree(tree1)
	require.NoError(t, err)
	require.True(t, treeHash1.IsValid())

	tree1Read, err := repo.ReadTree(treeHash1)
	require.NoError(t, err)
	require.ElementsMatch(t, tree1, tree1Read)

	tree2 := []TreeEntry{
		{
			ObjectType: Tree,
			Hash:       treeHash1,
			Name:       "tree1",
		},
		{
			ObjectType: Blob,
			Hash:       blobHash3,
			Name:       "blob3",
		},
	}

	treeHash2, err := repo.StoreTree(tree2)
	require.NoError(t, err)
	require.True(t, treeHash2.IsValid())

	tree2Read, err := repo.ReadTree(treeHash2)
	require.NoError(t, err)
	require.ElementsMatch(t, tree2, tree2Read)

	// Commit

	commit1, err := repo.StoreCommit(treeHash1)
	require.NoError(t, err)
	require.True(t, commit1.IsValid())

	treeHash1Read, err := repo.GetTreeHash(commit1)
	require.NoError(t, err)
	require.Equal(t, treeHash1, treeHash1Read)

	// commit with a parent
	commit2, err := repo.StoreCommit(treeHash2, commit1)
	require.NoError(t, err)
	require.True(t, commit2.IsValid())

	treeHash2Read, err := repo.GetTreeHash(commit2)
	require.NoError(t, err)
	require.Equal(t, treeHash2, treeHash2Read)

	// ReadTree should accept tree and commit hashes
	tree1read, err := repo.ReadTree(commit1)
	require.NoError(t, err)
	require.Equal(t, tree1read, tree1)

	c2, err := repo.ReadCommit(commit2)
	require.NoError(t, err)
	c2expected := Commit{Hash: commit2, Parents: []Hash{commit1}, TreeHash: treeHash2}
	require.Equal(t, c2expected, c2)

	// Ref

	exist1, err := repo.RefExist("refs/bugs/ref1")
	require.NoError(t, err)
	require.False(t, exist1)

	err = repo.UpdateRef("refs/bugs/ref1", commit2)
	require.NoError(t, err)

	exist1, err = repo.RefExist("refs/bugs/ref1")
	require.NoError(t, err)
	require.True(t, exist1)

	h, err := repo.ResolveRef("refs/bugs/ref1")
	require.NoError(t, err)
	require.Equal(t, commit2, h)

	ls, err := repo.ListRefs("refs/bugs")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"refs/bugs/ref1"}, ls)

	err = repo.CopyRef("refs/bugs/ref1", "refs/bugs/ref2")
	require.NoError(t, err)

	ls, err = repo.ListRefs("refs/bugs")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"refs/bugs/ref1", "refs/bugs/ref2"}, ls)

	commits, err := repo.ListCommits("refs/bugs/ref2")
	require.NoError(t, err)
	require.Equal(t, []Hash{commit1, commit2}, commits)

	// Graph

	commit3, err := repo.StoreCommit(treeHash1, commit1)
	require.NoError(t, err)

	ancestorHash, err := repo.FindCommonAncestor(commit2, commit3)
	require.NoError(t, err)
	require.Equal(t, commit1, ancestorHash)

	err = repo.RemoveRef("refs/bugs/ref1")
	require.NoError(t, err)

	// RemoveRef is idempotent
	err = repo.RemoveRef("refs/bugs/ref1")
	require.NoError(t, err)
}

func RepoDataSignatureTest(t *testing.T, repo RepoData) {
	data := randomData()

	blobHash, err := repo.StoreData(data)
	require.NoError(t, err)

	treeHash, err := repo.StoreTree([]TreeEntry{
		{
			ObjectType: Blob,
			Hash:       blobHash,
			Name:       "blob",
		},
	})
	require.NoError(t, err)

	pgpEntity1, err := openpgp.NewEntity("", "", "", nil)
	require.NoError(t, err)
	keyring1 := openpgp.EntityList{pgpEntity1}

	pgpEntity2, err := openpgp.NewEntity("", "", "", nil)
	require.NoError(t, err)
	keyring2 := openpgp.EntityList{pgpEntity2}

	commitHash1, err := repo.StoreSignedCommit(treeHash, pgpEntity1)
	require.NoError(t, err)

	commit1, err := repo.ReadCommit(commitHash1)
	require.NoError(t, err)

	_, err = openpgp.CheckDetachedSignature(keyring1, commit1.SignedData, commit1.Signature, nil)
	require.NoError(t, err)

	_, err = openpgp.CheckDetachedSignature(keyring2, commit1.SignedData, commit1.Signature, nil)
	require.Error(t, err)

	commitHash2, err := repo.StoreSignedCommit(treeHash, pgpEntity1, commitHash1)
	require.NoError(t, err)

	commit2, err := repo.ReadCommit(commitHash2)
	require.NoError(t, err)

	_, err = openpgp.CheckDetachedSignature(keyring1, commit2.SignedData, commit2.Signature, nil)
	require.NoError(t, err)

	_, err = openpgp.CheckDetachedSignature(keyring2, commit2.SignedData, commit2.Signature, nil)
	require.Error(t, err)
}

// helper to test a RepoClock
func RepoClockTest(t *testing.T, repo RepoClock) {
	allClocks, err := repo.AllClocks()
	require.NoError(t, err)
	require.Len(t, allClocks, 0)

	clock, err := repo.GetOrCreateClock("foo")
	require.NoError(t, err)
	require.Equal(t, lamport.Time(1), clock.Time())

	time, err := clock.Increment()
	require.NoError(t, err)
	require.Equal(t, lamport.Time(2), time)
	require.Equal(t, lamport.Time(2), clock.Time())

	clock2, err := repo.GetOrCreateClock("foo")
	require.NoError(t, err)
	require.Equal(t, lamport.Time(2), clock2.Time())

	clock3, err := repo.GetOrCreateClock("bar")
	require.NoError(t, err)
	require.Equal(t, lamport.Time(1), clock3.Time())

	allClocks, err = repo.AllClocks()
	require.NoError(t, err)
	require.Equal(t, map[string]lamport.Clock{
		"foo": clock,
		"bar": clock3,
	}, allClocks)
}

func randomData() []byte {
	var letterRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 32)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return b
}
