package repository

import (
	"testing"
)

func TestMockRepo(t *testing.T) {
	creator := func(t TestingT, bare bool) TestedRepo {
		t.Helper()

		return NewMockRepo()
	}

	RepoTest(t, creator)
}
