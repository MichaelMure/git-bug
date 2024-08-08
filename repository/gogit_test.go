package repository

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGoGitRepo(t *testing.T) {
	// Plain
	plainRepo := CreateGoGitTestRepo(t, false)
	plainRoot := goGitRepoDir(t, plainRepo)
	require.NoError(t, plainRepo.Close())
	plainGitDir := filepath.Join(plainRoot, ".git")

	// Bare
	bareRepo := CreateGoGitTestRepo(t, true)
	bareRoot := goGitRepoDir(t, bareRepo)
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
	RepoTest(t, CreateGoGitTestRepo)
}

func TestGoGitRepo_Indexes(t *testing.T) {
	repo := CreateGoGitTestRepo(t, false)
	plainRoot := goGitRepoDir(t, repo)

	// Can create indices
	indexA, err := repo.GetIndex("a")
	require.NoError(t, err)
	require.NotZero(t, indexA)
	require.FileExists(t, filepath.Join(plainRoot, ".git", namespace, "indexes", "a", "index_meta.json"))
	require.FileExists(t, filepath.Join(plainRoot, ".git", namespace, "indexes", "a", "store"))

	indexB, err := repo.GetIndex("b")
	require.NoError(t, err)
	require.NotZero(t, indexB)
	require.DirExists(t, filepath.Join(plainRoot, ".git", namespace, "indexes", "b"))

	// Can get an existing index
	indexA, err = repo.GetIndex("a")
	require.NoError(t, err)
	require.NotZero(t, indexA)
}

func TestGoGit_DetectsSubmodules(t *testing.T) {
	repo := CreateGoGitTestRepo(t, false)
	expected := filepath.Join(goGitRepoDir(t, repo), "/.git")

	d := t.TempDir()
	err := os.WriteFile(filepath.Join(d, ".git"), []byte(fmt.Sprintf("gitdir: %s", expected)), 0600)
	require.NoError(t, err)

	result, err := detectGitPath(d, 0)
	assert.Empty(t, err)
	assert.Equal(t, expected, result)
}

func TestGoGitRepoSSH(t *testing.T) {
	repo := CreateGoGitTestRepo(t, false)

	err := repo.AddRemote("ssh", "ssh://git@github.com:MichaelMure/git-bug.git")
	if err != nil {
		log.Fatal(err)
	}
	keys, err := repo.SSHAuth("ssh")
	require.NotNil(t, keys)
	require.Empty(t, err)

	err = repo.AddRemote("http", "http://github.com/MichaelMure/git-bug.git")
	if err != nil {
		log.Fatal(err)
	}
	keys, err = repo.SSHAuth("http")
	require.Nil(t, keys)
	require.Empty(t, err)

	err = repo.AddRemote("https", "https://github.com/MichaelMure/git-bug.git")
	if err != nil {
		log.Fatal(err)
	}
	keys, err = repo.SSHAuth("https")
	require.Nil(t, keys)
	require.Empty(t, err)

	err = repo.AddRemote("git", "git://github.com/MichaelMure/git-bug.git")
	if err != nil {
		log.Fatal(err)
	}
	keys, err = repo.SSHAuth("git")
	require.Nil(t, keys)
	require.Empty(t, err)

	err = repo.AddRemote("scp-like", "git@github.com:MichaelMure/git-bug.git")
	if err != nil {
		log.Fatal(err)
	}
	keys, err = repo.SSHAuth("scp-like")
	require.NotNil(t, keys)
	require.Empty(t, err)

}
