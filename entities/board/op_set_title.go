package board

import (
	"fmt"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/util/text"
)

var _ Operation = &SetTitleOperation{}

// SetTitleOperation will change the title of a board
type SetTitleOperation struct {
	dag.OpBase
	Title string `json:"title"`
	Was   string `json:"was"`
}

func (op *SetTitleOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *SetTitleOperation) Validate() error {
	if err := op.OpBase.Validate(op, SetTitleOp); err != nil {
		return err
	}

	if text.Empty(op.Title) {
		return fmt.Errorf("title is empty")
	}

	if !text.SafeOneLine(op.Title) {
		return fmt.Errorf("title has unsafe characters")
	}

	if !text.SafeOneLine(op.Was) {
		return fmt.Errorf("previous title has unsafe characters")
	}

	return nil
}

func (op *SetTitleOperation) Apply(snapshot *Snapshot) {
	snapshot.Title = op.Title
	snapshot.addActor(op.Author())
}

func NewSetTitleOp(author identity.Interface, unixTime int64, title string, was string) *SetTitleOperation {
	return &SetTitleOperation{
		OpBase: dag.NewOpBase(SetTitleOp, author, unixTime),
		Title:  title,
		Was:    was,
	}
}

// SetTitle is a convenience function to change a board title
func SetTitle(b Interface, author identity.Interface, unixTime int64, title string, metadata map[string]string) (*SetTitleOperation, error) {
	var lastTitleOp *SetTitleOperation
	for _, op := range b.Operations() {
		switch op := op.(type) {
		case *SetTitleOperation:
			lastTitleOp = op
		}
	}

	var was string
	if lastTitleOp != nil {
		was = lastTitleOp.Title
	} else {
		was = b.FirstOp().(*CreateOperation).Title
	}

	op := NewSetTitleOp(author, unixTime, title, was)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}

	b.Append(op)
	return op, nil
}
