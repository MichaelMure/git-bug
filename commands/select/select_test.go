package _select

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
)

func TestSelect(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(t, false)

	repoCache, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	_, _, err = ResolveBug(repoCache, []string{})
	require.Equal(t, ErrNoValidId, err)

	err = Select(repoCache, "invalid")
	require.NoError(t, err)

	// Resolve without a pattern should fail when no bug is selected
	_, _, err = ResolveBug(repoCache, []string{})
	require.Error(t, err)

	// generate a bunch of bugs

	rene, err := repoCache.NewIdentity("René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		_, _, err := repoCache.NewBugRaw(rene, time.Now().Unix(), "title", "message", nil, nil)
		require.NoError(t, err)
	}

	// and two more for testing
	b1, _, err := repoCache.NewBugRaw(rene, time.Now().Unix(), "title", "message", nil, nil)
	require.NoError(t, err)
	b2, _, err := repoCache.NewBugRaw(rene, time.Now().Unix(), "title", "message", nil, nil)
	require.NoError(t, err)

	err = Select(repoCache, b1.Id())
	require.NoError(t, err)

	// normal select without args
	b3, _, err := ResolveBug(repoCache, []string{})
	require.NoError(t, err)
	require.Equal(t, b1.Id(), b3.Id())

	// override selection with same id
	b4, _, err := ResolveBug(repoCache, []string{b1.Id().String()})
	require.NoError(t, err)
	require.Equal(t, b1.Id(), b4.Id())

	// override selection with a prefix
	b5, _, err := ResolveBug(repoCache, []string{b1.Id().Human()})
	require.NoError(t, err)
	require.Equal(t, b1.Id(), b5.Id())

	// args that shouldn't override
	b6, _, err := ResolveBug(repoCache, []string{"arg"})
	require.NoError(t, err)
	require.Equal(t, b1.Id(), b6.Id())

	// override with a different id
	b7, _, err := ResolveBug(repoCache, []string{b2.Id().String()})
	require.NoError(t, err)
	require.Equal(t, b2.Id(), b7.Id())

	err = Clear(repoCache)
	require.NoError(t, err)

	// Resolve without a pattern should error again after clearing the selected bug
	_, _, err = ResolveBug(repoCache, []string{})
	require.Error(t, err)
}
