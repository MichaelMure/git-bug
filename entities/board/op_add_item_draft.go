package board

import (
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
)

var _ Operation = &AddItemDraftOperation{}

type AddItemDraftOperation struct {
	dag.OpBase
	Title   string `json:"title"`
	Message string `json:"message"`
}

func (op *AddItemDraftOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *AddItemDraftOperation) Validate() error {
	// TODO implement me
	panic("implement me")
}

func (op *AddItemDraftOperation) Apply(snapshot *Snapshot) {
	// TODO implement me
	panic("implement me")
}
