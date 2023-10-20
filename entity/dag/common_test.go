package dag

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

// This file contains an example dummy entity to be used in the tests

/*
 Operations
*/

const (
	_ entity.OperationType = iota
	Op1
	Op2
)

type op1 struct {
	OpBase
	Field1 string            `json:"field_1"`
	Files  []repository.Hash `json:"files"`
}

func newOp1(author identity.Interface, field1 string, files ...repository.Hash) *op1 {
	return &op1{OpBase: NewOpBase(Op1, author, 0), Field1: field1, Files: files}
}

func (op *op1) Id() entity.Id {
	return IdOperation(op, &op.OpBase)
}

func (op *op1) Validate() error { return nil }

func (op *op1) GetFiles() []repository.Hash {
	return op.Files
}

type op2 struct {
	OpBase
	Field2 string `json:"field_2"`
}

func newOp2(author identity.Interface, field2 string) *op2 {
	return &op2{OpBase: NewOpBase(Op2, author, 0), Field2: field2}
}

func (op *op2) Id() entity.Id {
	return IdOperation(op, &op.OpBase)
}

func (op *op2) Validate() error { return nil }

func unmarshaler(raw json.RawMessage, resolvers entity.Resolvers) (Operation, error) {
	var t struct {
		OperationType entity.OperationType `json:"type"`
	}

	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}

	var op Operation

	switch t.OperationType {
	case Op1:
		op = &op1{}
	case Op2:
		op = &op2{}
	default:
		return nil, fmt.Errorf("unknown operation type %v", t.OperationType)
	}

	err := json.Unmarshal(raw, &op)
	if err != nil {
		return nil, err
	}

	return op, nil
}

/*
  Entity
*/

type Foo struct {
	*Entity
}

func wrapper(e *Entity) *Foo {
	return &Foo{Entity: e}
}

/*
  Identities + repo + definition
*/

func makeTestContext() (repository.ClockedRepo, identity.Interface, identity.Interface, entity.Resolvers, Definition) {
	repo := repository.NewMockRepo()
	id1, id2, resolvers, def := makeTestContextInternal(repo)
	return repo, id1, id2, resolvers, def
}

func makeTestContextGoGit(t *testing.T) (repository.ClockedRepo, identity.Interface, identity.Interface, entity.Resolvers, Definition) {
	repo := repository.CreateGoGitTestRepo(t, false)
	id1, id2, resolvers, def := makeTestContextInternal(repo)
	return repo, id1, id2, resolvers, def
}

func makeTestContextRemote(t *testing.T) (repository.ClockedRepo, repository.ClockedRepo, repository.ClockedRepo, identity.Interface, identity.Interface, entity.Resolvers, Definition) {
	repoA := repository.CreateGoGitTestRepo(t, false)
	repoB := repository.CreateGoGitTestRepo(t, false)
	remote := repository.CreateGoGitTestRepo(t, true)

	err := repoA.AddRemote("remote", remote.GetLocalRemote())
	require.NoError(t, err)
	err = repoA.AddRemote("repoB", repoB.GetLocalRemote())
	require.NoError(t, err)
	err = repoB.AddRemote("remote", remote.GetLocalRemote())
	require.NoError(t, err)
	err = repoB.AddRemote("repoA", repoA.GetLocalRemote())
	require.NoError(t, err)

	id1, id2, resolver, def := makeTestContextInternal(repoA)

	// distribute the identities
	_, err = identity.Push(repoA, "remote")
	require.NoError(t, err)
	err = identity.Pull(repoB, "remote")
	require.NoError(t, err)

	return repoA, repoB, remote, id1, id2, resolver, def
}

func makeTestContextInternal(repo repository.ClockedRepo) (identity.Interface, identity.Interface, entity.Resolvers, Definition) {
	id1, err := identity.NewIdentity(repo, "name1", "email1")
	if err != nil {
		panic(err)
	}
	err = id1.Commit(repo)
	if err != nil {
		panic(err)
	}
	id2, err := identity.NewIdentity(repo, "name2", "email2")
	if err != nil {
		panic(err)
	}
	err = id2.Commit(repo)
	if err != nil {
		panic(err)
	}

	resolvers := entity.Resolvers{
		&identity.Identity{}: entity.MakeResolver(id1, id2),
	}

	def := Definition{
		Typename:             "foo",
		Namespace:            "foos",
		OperationUnmarshaler: unmarshaler,
		FormatVersion:        1,
	}

	return id1, id2, resolvers, def
}
