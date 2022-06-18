package repository

import (
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGoGitRepo(t *testing.T) {
	t.Parallel()

	// Plain
	plainRoot := t.TempDir()

	plainRepo, err := InitGoGitRepo(plainRoot, namespace)
	require.NoError(t, err)
	require.NoError(t, plainRepo.Close())
	plainGitDir := filepath.Join(plainRoot, ".git")

	// Bare
	bareRoot := t.TempDir()

	bareRepo, err := InitBareGoGitRepo(bareRoot, namespace)
	require.NoError(t, err)
	require.NoError(t, bareRepo.Close())
	bareGitDir := bareRoot

	tests := []struct {
		inPath  string
		outPath string
		err     bool
	}{
		// errors
		{"/", "", true},
		// parent dir of a repo
		{filepath.Dir(plainRoot), "", true},

		// Plain repo
		{plainRoot, plainGitDir, false},
		{plainGitDir, plainGitDir, false},
		{path.Join(plainGitDir, "objects"), plainGitDir, false},

		// Bare repo
		{bareRoot, bareGitDir, false},
		{bareGitDir, bareGitDir, false},
		{path.Join(bareGitDir, "objects"), bareGitDir, false},
	}

	for i, tc := range tests {
		r, err := OpenGoGitRepo(tc.inPath, namespace, nil)

		if tc.err {
			require.Error(t, err, i)
		} else {
			require.NoError(t, err, i)
			assert.Equal(t, filepath.ToSlash(tc.outPath), filepath.ToSlash(r.path), i)
			require.NoError(t, r.Close())
		}
	}
}

func TestGoGitRepo(t *testing.T) {
	t.Parallel()

	RepoTest(t, CreateGoGitTestRepo)
}

// func TestGoGitRepo_Indexes(t *testing.T) {
// 	t.Parallel()

// 	plainRoot := t.TempDir()

// 	repo, err := InitGoGitRepo(plainRoot, namespace)
// 	require.NoError(t, err)
// 	t.Cleanup(func() {
// 		require.NoError(t, repo.Close())
// 	})

// 	// Can create indices
// 	indexA, err := repo.GetBleveIndex("a")
// 	require.NoError(t, err)
// 	require.NotZero(t, indexA)
// 	require.FileExists(t, filepath.Join(plainRoot, ".git", namespace, "indexes", "a", "index_meta.json"))
// 	require.FileExists(t, filepath.Join(plainRoot, ".git", namespace, "indexes", "a", "store"))

// 	indexB, err := repo.GetBleveIndex("b")
// 	require.NoError(t, err)
// 	require.NotZero(t, indexB)

// 	// Can get an existing index
// 	indexA, err = repo.GetBleveIndex("a")
// 	require.NoError(t, err)
// 	require.NotZero(t, indexA)

// 	// Can delete an index
// 	err = repo.ClearBleveIndex("a")
// 	require.NoError(t, err)
// 	require.NoDirExists(t, filepath.Join(plainRoot, ".git", namespace, "indexes", "a"))
// }
