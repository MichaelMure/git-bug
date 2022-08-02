package board

import (
	"encoding/json"
	"fmt"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
)

// OperationType is an operation type identifier
type OperationType dag.OperationType

const (
	_ dag.OperationType = iota
	CreateOp
	SetTitleOp
	SetDescriptionOp
	AddItemEntityOp
	AddItemDraftOp
	MoveItemOp
	RemoveItemOp

	// TODO: change columns?
)

type Operation interface {
	dag.Operation
	// Apply the operation to a Snapshot to create the final state
	Apply(snapshot *Snapshot)
}

func operationUnmarshaller(raw json.RawMessage, resolvers entity.Resolvers) (dag.Operation, error) {
	var t struct {
		OperationType dag.OperationType `json:"type"`
	}

	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}

	var op dag.Operation

	switch t.OperationType {
	case CreateOp:
		op = &CreateOperation{}
	case SetTitleOp:
		op = &SetTitleOperation{}
	case SetDescriptionOp:
		op = &SetDescriptionOperation{}
	case AddItemDraftOp:
		op = &AddItemDraftOperation{}
	case AddItemEntityOp:
		op = &AddItemEntityOperation{}
	default:
		panic(fmt.Sprintf("unknown operation type %v", t.OperationType))
	}

	err := json.Unmarshal(raw, &op)
	if err != nil {
		return nil, err
	}

	switch op := op.(type) {
	case *AddItemEntityOperation:
		// TODO: resolve entity
		op.item = struct{}{}
	}

	return op, nil
}
