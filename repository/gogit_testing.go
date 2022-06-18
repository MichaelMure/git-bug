package repository

import (
	"log"
	"testing"

	"github.com/99designs/keyring"
	"github.com/stretchr/testify/require"
)

const namespace = "git-bug"

type TestingT interface {
	Cleanup(func())
	Errorf(string, ...interface{})
	FailNow()
	Helper()
	TempDir() string
}

// This is intended for testing only

func CreateGoGitTestRepo(t TestingT, bare bool) TestedRepo {
	t.Helper()

	dir := t.TempDir()

	var creator func(string, string) (*GoGitRepo, error)

	if bare {
		creator = InitBareGoGitRepo
	} else {
		creator = InitGoGitRepo
	}

	repo, err := creator(dir, namespace)
	if err != nil {
		log.Fatal(err)
	}

	t.Cleanup(func() {
		require.NoError(t, repo.Close())
	})

	config := repo.LocalConfig()
	if err := config.StoreString("user.name", "testuser"); err != nil {
		log.Fatal("failed to set user.name for test repository: ", err)
	}
	if err := config.StoreString("user.email", "testuser@example.com"); err != nil {
		log.Fatal("failed to set user.email for test repository: ", err)
	}

	// make sure we use a mock keyring for testing to not interact with the global system
	return &replaceKeyring{
		TestedRepo: repo,
		keyring:    keyring.NewArrayKeyring(nil),
	}
}

func SetupGoGitReposAndRemote(t *testing.T) (repoA, repoB, remote TestedRepo) {
	repoA = CreateGoGitTestRepo(t, false)
	repoB = CreateGoGitTestRepo(t, false)
	remote = CreateGoGitTestRepo(t, true)

	err := repoA.AddRemote("origin", remote.GetLocalRemote())
	if err != nil {
		log.Fatal(err)
	}

	err = repoB.AddRemote("origin", remote.GetLocalRemote())
	if err != nil {
		log.Fatal(err)
	}

	return repoA, repoB, remote
}
