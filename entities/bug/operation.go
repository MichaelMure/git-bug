package bug

import (
	"encoding/json"
	"fmt"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
)

const (
	_ entity.OperationType = iota
	CreateOp
	SetTitleOp
	AddCommentOp
	SetStatusOp
	LabelChangeOp
	EditCommentOp
	NoOpOp
	SetMetadataOp
)

// Operation define the interface to fulfill for an edit operation of a Bug
type Operation = dag.OperationWithApply[*Snapshot]

// make sure that package external operations do conform to our interface
var _ Operation = &dag.NoOpOperation[*Snapshot]{}
var _ Operation = &dag.SetMetadataOperation[*Snapshot]{}

func operationUnmarshaler(raw json.RawMessage, resolvers entity.Resolvers) (dag.Operation, error) {
	var t struct {
		OperationType entity.OperationType `json:"type"`
	}

	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}

	var op dag.Operation

	switch t.OperationType {
	case AddCommentOp:
		op = &AddCommentOperation{}
	case CreateOp:
		op = &CreateOperation{}
	case EditCommentOp:
		op = &EditCommentOperation{}
	case LabelChangeOp:
		op = &LabelChangeOperation{}
	case NoOpOp:
		op = &dag.NoOpOperation[*Snapshot]{}
	case SetMetadataOp:
		op = &dag.SetMetadataOperation[*Snapshot]{}
	case SetStatusOp:
		op = &SetStatusOperation{}
	case SetTitleOp:
		op = &SetTitleOperation{}
	default:
		panic(fmt.Sprintf("unknown operation type %v", t.OperationType))
	}

	err := json.Unmarshal(raw, &op)
	if err != nil {
		return nil, err
	}

	return op, nil
}
