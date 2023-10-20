package dag

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/entity"
	bootstrap "github.com/MichaelMure/git-bug/entity/boostrap"
	"github.com/MichaelMure/git-bug/repository"
)

func allEntities(t testing.TB, bugs <-chan bootstrap.StreamedEntity[*Foo]) []*Foo {
	t.Helper()

	var result []*Foo
	for streamed := range bugs {
		require.NoError(t, streamed.Err)

		result = append(result, streamed.Entity)
	}
	return result
}

func TestEntityPushPull(t *testing.T) {
	repoA, repoB, _, id1, id2, resolvers, def := makeTestContextRemote(t)

	// A --> remote --> B
	e := New(def)
	e.Append(newOp1(id1, "foo"))

	err := e.Commit(repoA)
	require.NoError(t, err)

	_, err = Push(def, repoA, "remote")
	require.NoError(t, err)

	err = Pull(def, wrapper, repoB, resolvers, "remote", id1)
	require.NoError(t, err)

	entities := allEntities(t, ReadAll(def, wrapper, repoB, resolvers))
	require.Len(t, entities, 1)

	// B --> remote --> A
	e = New(def)
	e.Append(newOp2(id2, "bar"))

	err = e.Commit(repoB)
	require.NoError(t, err)

	_, err = Push(def, repoB, "remote")
	require.NoError(t, err)

	err = Pull(def, wrapper, repoA, resolvers, "remote", id1)
	require.NoError(t, err)

	entities = allEntities(t, ReadAll(def, wrapper, repoB, resolvers))
	require.Len(t, entities, 2)
}

func TestListLocalIds(t *testing.T) {
	repoA, repoB, _, id1, id2, resolvers, def := makeTestContextRemote(t)

	// A --> remote --> B
	e := New(def)
	e.Append(newOp1(id1, "foo"))
	err := e.Commit(repoA)
	require.NoError(t, err)

	e = New(def)
	e.Append(newOp2(id2, "bar"))
	err = e.Commit(repoA)
	require.NoError(t, err)

	listLocalIds(t, def, repoA, 2)
	listLocalIds(t, def, repoB, 0)

	_, err = Push(def, repoA, "remote")
	require.NoError(t, err)

	_, err = Fetch(def, repoB, "remote")
	require.NoError(t, err)

	listLocalIds(t, def, repoA, 2)
	listLocalIds(t, def, repoB, 0)

	err = Pull(def, wrapper, repoB, resolvers, "remote", id1)
	require.NoError(t, err)

	listLocalIds(t, def, repoA, 2)
	listLocalIds(t, def, repoB, 2)
}

func listLocalIds(t *testing.T, def Definition, repo repository.RepoData, expectedCount int) {
	ids, err := ListLocalIds(def, repo)
	require.NoError(t, err)
	require.Len(t, ids, expectedCount)
}

func assertMergeResults(t *testing.T, expected []entity.MergeResult, results <-chan entity.MergeResult) {
	t.Helper()

	var allResults []entity.MergeResult
	for result := range results {
		allResults = append(allResults, result)
	}

	require.Equal(t, len(expected), len(allResults))

	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Id < allResults[j].Id
	})
	sort.Slice(expected, func(i, j int) bool {
		return expected[i].Id < expected[j].Id
	})

	for i, result := range allResults {
		require.NoError(t, result.Err)

		require.Equal(t, expected[i].Id, result.Id)
		require.Equal(t, expected[i].Status, result.Status)

		switch result.Status {
		case entity.MergeStatusNew, entity.MergeStatusUpdated:
			require.NotNil(t, result.Entity)
			require.Equal(t, expected[i].Id, result.Entity.Id())
		}

		i++
	}
}

func assertEqualRefs(t *testing.T, repoA, repoB repository.RepoData, prefix string) {
	t.Helper()

	refsA, err := repoA.ListRefs("")
	require.NoError(t, err)

	var refsAFiltered []string
	for _, ref := range refsA {
		if strings.HasPrefix(ref, prefix) {
			refsAFiltered = append(refsAFiltered, ref)
		}
	}

	refsB, err := repoB.ListRefs("")
	require.NoError(t, err)

	var refsBFiltered []string
	for _, ref := range refsB {
		if strings.HasPrefix(ref, prefix) {
			refsBFiltered = append(refsBFiltered, ref)
		}
	}

	require.NotEmpty(t, refsAFiltered)
	require.Equal(t, refsAFiltered, refsBFiltered)

	for _, ref := range refsAFiltered {
		commitA, err := repoA.ResolveRef(ref)
		require.NoError(t, err)
		commitB, err := repoB.ResolveRef(ref)
		require.NoError(t, err)

		require.Equal(t, commitA, commitB)
	}
}

func assertNotEqualRefs(t *testing.T, repoA, repoB repository.RepoData, prefix string) {
	t.Helper()

	refsA, err := repoA.ListRefs("")
	require.NoError(t, err)

	var refsAFiltered []string
	for _, ref := range refsA {
		if strings.HasPrefix(ref, prefix) {
			refsAFiltered = append(refsAFiltered, ref)
		}
	}

	refsB, err := repoB.ListRefs("")
	require.NoError(t, err)

	var refsBFiltered []string
	for _, ref := range refsB {
		if strings.HasPrefix(ref, prefix) {
			refsBFiltered = append(refsBFiltered, ref)
		}
	}

	require.NotEmpty(t, refsAFiltered)
	require.Equal(t, refsAFiltered, refsBFiltered)

	for _, ref := range refsAFiltered {
		commitA, err := repoA.ResolveRef(ref)
		require.NoError(t, err)
		commitB, err := repoB.ResolveRef(ref)
		require.NoError(t, err)

		require.NotEqual(t, commitA, commitB)
	}
}

func TestMerge(t *testing.T) {
	repoA, repoB, _, id1, id2, resolvers, def := makeTestContextRemote(t)

	// SCENARIO 1
	// if the remote Entity doesn't exist locally, it's created

	// 2 entities in repoA + push to remote
	e1A := New(def)
	e1A.Append(newOp1(id1, "foo"))
	err := e1A.Commit(repoA)
	require.NoError(t, err)

	e2A := New(def)
	e2A.Append(newOp2(id2, "bar"))
	err = e2A.Commit(repoA)
	require.NoError(t, err)

	_, err = Push(def, repoA, "remote")
	require.NoError(t, err)

	// repoB: fetch + merge from remote

	_, err = Fetch(def, repoB, "remote")
	require.NoError(t, err)

	results := MergeAll(def, wrapper, repoB, resolvers, "remote", id1)

	assertMergeResults(t, []entity.MergeResult{
		{
			Id:     e1A.Id(),
			Status: entity.MergeStatusNew,
		},
		{
			Id:     e2A.Id(),
			Status: entity.MergeStatusNew,
		},
	}, results)

	assertEqualRefs(t, repoA, repoB, "refs/"+def.Namespace)

	// SCENARIO 2
	// if the remote and local Entity have the same state, nothing is changed

	results = MergeAll(def, wrapper, repoB, resolvers, "remote", id1)

	assertMergeResults(t, []entity.MergeResult{
		{
			Id:     e1A.Id(),
			Status: entity.MergeStatusNothing,
		},
		{
			Id:     e2A.Id(),
			Status: entity.MergeStatusNothing,
		},
	}, results)

	assertEqualRefs(t, repoA, repoB, "refs/"+def.Namespace)

	// SCENARIO 3
	// if the local Entity has new commits but the remote don't, nothing is changed

	e1A.Append(newOp1(id1, "barbar"))
	err = e1A.Commit(repoA)
	require.NoError(t, err)

	e2A.Append(newOp2(id2, "barbarbar"))
	err = e2A.Commit(repoA)
	require.NoError(t, err)

	results = MergeAll(def, wrapper, repoA, resolvers, "remote", id1)

	assertMergeResults(t, []entity.MergeResult{
		{
			Id:     e1A.Id(),
			Status: entity.MergeStatusNothing,
		},
		{
			Id:     e2A.Id(),
			Status: entity.MergeStatusNothing,
		},
	}, results)

	assertNotEqualRefs(t, repoA, repoB, "refs/"+def.Namespace)

	// SCENARIO 4
	// if the remote has new commit, the local bug is updated to match the same history
	// (fast-forward update)

	_, err = Push(def, repoA, "remote")
	require.NoError(t, err)

	_, err = Fetch(def, repoB, "remote")
	require.NoError(t, err)

	results = MergeAll(def, wrapper, repoB, resolvers, "remote", id1)

	assertMergeResults(t, []entity.MergeResult{
		{
			Id:     e1A.Id(),
			Status: entity.MergeStatusUpdated,
		},
		{
			Id:     e2A.Id(),
			Status: entity.MergeStatusUpdated,
		},
	}, results)

	assertEqualRefs(t, repoA, repoB, "refs/"+def.Namespace)

	// SCENARIO 5
	// if both local and remote Entity have new commits (that is, we have a concurrent edition),
	// a merge commit with an empty operationPack is created to join both branch and form a DAG.

	e1A.Append(newOp1(id1, "barbarfoo"))
	err = e1A.Commit(repoA)
	require.NoError(t, err)

	e2A.Append(newOp2(id2, "barbarbarfoo"))
	err = e2A.Commit(repoA)
	require.NoError(t, err)

	e1B, err := Read(def, wrapper, repoB, resolvers, e1A.Id())
	require.NoError(t, err)

	e2B, err := Read(def, wrapper, repoB, resolvers, e2A.Id())
	require.NoError(t, err)

	e1B.Append(newOp1(id1, "barbarfoofoo"))
	err = e1B.Commit(repoB)
	require.NoError(t, err)

	e2B.Append(newOp2(id2, "barbarbarfoofoo"))
	err = e2B.Commit(repoB)
	require.NoError(t, err)

	_, err = Push(def, repoA, "remote")
	require.NoError(t, err)

	_, err = Fetch(def, repoB, "remote")
	require.NoError(t, err)

	results = MergeAll(def, wrapper, repoB, resolvers, "remote", id1)

	assertMergeResults(t, []entity.MergeResult{
		{
			Id:     e1A.Id(),
			Status: entity.MergeStatusUpdated,
		},
		{
			Id:     e2A.Id(),
			Status: entity.MergeStatusUpdated,
		},
	}, results)

	assertNotEqualRefs(t, repoA, repoB, "refs/"+def.Namespace)

	_, err = Push(def, repoB, "remote")
	require.NoError(t, err)

	_, err = Fetch(def, repoA, "remote")
	require.NoError(t, err)

	results = MergeAll(def, wrapper, repoA, resolvers, "remote", id1)

	assertMergeResults(t, []entity.MergeResult{
		{
			Id:     e1A.Id(),
			Status: entity.MergeStatusUpdated,
		},
		{
			Id:     e2A.Id(),
			Status: entity.MergeStatusUpdated,
		},
	}, results)

	// make sure that the graphs become stable over multiple repo, due to the
	// fast-forward
	assertEqualRefs(t, repoA, repoB, "refs/"+def.Namespace)
}

func TestRemove(t *testing.T) {
	repoA, _, _, id1, _, resolvers, def := makeTestContextRemote(t)

	e := New(def)
	e.Append(newOp1(id1, "foo"))
	require.NoError(t, e.Commit(repoA))

	_, err := Push(def, repoA, "remote")
	require.NoError(t, err)

	err = Remove(def, repoA, e.Id())
	require.NoError(t, err)

	_, err = Read(def, wrapper, repoA, resolvers, e.Id())
	require.Error(t, err)

	_, err = readRemote(def, wrapper, repoA, resolvers, "remote", e.Id())
	require.Error(t, err)

	// Remove is idempotent
	err = Remove(def, repoA, e.Id())
	require.NoError(t, err)
}

func TestRemoveAll(t *testing.T) {
	repoA, _, _, id1, _, resolvers, def := makeTestContextRemote(t)

	var ids []entity.Id

	for i := 0; i < 10; i++ {
		e := New(def)
		e.Append(newOp1(id1, "foo"))
		require.NoError(t, e.Commit(repoA))
		ids = append(ids, e.Id())
	}

	_, err := Push(def, repoA, "remote")
	require.NoError(t, err)

	err = RemoveAll(def, repoA)
	require.NoError(t, err)

	for _, id := range ids {
		_, err = Read(def, wrapper, repoA, resolvers, id)
		require.Error(t, err)

		_, err = readRemote(def, wrapper, repoA, resolvers, "remote", id)
		require.Error(t, err)
	}

	// Remove is idempotent
	err = RemoveAll(def, repoA)
	require.NoError(t, err)
}
