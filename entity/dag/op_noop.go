package dag

import (
	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
)

var _ Operation = &NoOpOperation[Snapshot]{}
var _ entity.OperationDoesntChangeSnapshot = &NoOpOperation[Snapshot]{}

// NoOpOperation is an operation that does not change the entity state. It can
// however be used to store arbitrary metadata in the entity history, for example
// to support a bridge feature.
type NoOpOperation[SnapT Snapshot] struct {
	OpBase
}

func NewNoOpOp[SnapT Snapshot](opType entity.OperationType, author identity.Interface, unixTime int64) *NoOpOperation[SnapT] {
	return &NoOpOperation[SnapT]{
		OpBase: NewOpBase(opType, author, unixTime),
	}
}

func (op *NoOpOperation[SnapT]) Id() entity.Id {
	return IdOperation(op, &op.OpBase)
}

func (op *NoOpOperation[SnapT]) Apply(snapshot SnapT) {
	// Nothing to do
}

func (op *NoOpOperation[SnapT]) Validate() error {
	if err := op.OpBase.Validate(op, op.OperationType); err != nil {
		return err
	}
	return nil
}

func (op *NoOpOperation[SnapT]) DoesntChangeSnapshot() {}
