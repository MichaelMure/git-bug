package tests

import (
	"testing"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/misc/random_bugs"
	"github.com/MichaelMure/git-bug/repository"
)

func TestReadBugs(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(t, false)

	random_bugs.FillRepoWithSeed(repo, 15, 42)

	bugs := bug.ReadAll(repo)
	for b := range bugs {
		if b.Err != nil {
			t.Fatal(b.Err)
		}
	}
}

func benchmarkReadBugs(bugNumber int, t *testing.B) {
	repo := repository.CreateGoGitTestRepo(t, false)

	random_bugs.FillRepoWithSeed(repo, bugNumber, 42)
	t.ResetTimer()

	for n := 0; n < t.N; n++ {
		bugs := bug.ReadAll(repo)
		for b := range bugs {
			if b.Err != nil {
				t.Fatal(b.Err)
			}
		}
	}
}

func BenchmarkReadBugs5(b *testing.B)   { benchmarkReadBugs(5, b) }
func BenchmarkReadBugs25(b *testing.B)  { benchmarkReadBugs(25, b) }
func BenchmarkReadBugs150(b *testing.B) { benchmarkReadBugs(150, b) }
