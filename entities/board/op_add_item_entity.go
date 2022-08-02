package board

import (
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
)

var _ Operation = &AddItemEntityOperation{}

type AddItemEntityOperation struct {
	dag.OpBase
	// TODO: entity namespace + id ? or solve https://github.com/MichaelMure/git-bug/pull/664 ?
	item CardItem
}

func (op *AddItemEntityOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *AddItemEntityOperation) Validate() error {
	// TODO implement me
	panic("implement me")
}

func (op *AddItemEntityOperation) Apply(snapshot *Snapshot) {
	// TODO implement me
	panic("implement me")
}
