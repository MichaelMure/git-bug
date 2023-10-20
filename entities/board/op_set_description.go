package board

import (
	"fmt"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/util/text"
)

var _ Operation = &SetDescriptionOperation{}

// SetDescriptionOperation will change the description of a board
type SetDescriptionOperation struct {
	dag.OpBase
	Description string `json:"description"`
	Was         string `json:"was"`
}

func (op *SetDescriptionOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *SetDescriptionOperation) Validate() error {
	if err := op.OpBase.Validate(op, SetDescriptionOp); err != nil {
		return err
	}

	if text.Empty(op.Description) {
		return fmt.Errorf("description is empty")
	}

	if !text.SafeOneLine(op.Description) {
		return fmt.Errorf("description has unsafe characters")
	}

	if !text.SafeOneLine(op.Was) {
		return fmt.Errorf("previous description has unsafe characters")
	}

	return nil
}

func (op *SetDescriptionOperation) Apply(snapshot *Snapshot) {
	snapshot.Description = op.Description
	snapshot.addParticipant(op.Author())
}

func NewSetDescriptionOp(author identity.Interface, unixTime int64, description string, was string) *SetDescriptionOperation {
	return &SetDescriptionOperation{
		OpBase:      dag.NewOpBase(SetDescriptionOp, author, unixTime),
		Description: description,
		Was:         was,
	}
}

// SetDescription is a convenience function to change a board description
func SetDescription(b Interface, author identity.Interface, unixTime int64, description string, metadata map[string]string) (*SetDescriptionOperation, error) {
	var lastDescriptionOp *SetDescriptionOperation
	for _, op := range b.Operations() {
		switch op := op.(type) {
		case *SetDescriptionOperation:
			lastDescriptionOp = op
		}
	}

	var was string
	if lastDescriptionOp != nil {
		was = lastDescriptionOp.Description
	} else {
		was = b.FirstOp().(*CreateOperation).Description
	}

	op := NewSetDescriptionOp(author, unixTime, description, was)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}

	b.Append(op)
	return op, nil
}
