package board

import (
	"fmt"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/text"
	"github.com/MichaelMure/git-bug/util/timestamp"
)

var _ Operation = &AddItemDraftOperation{}

type AddItemDraftOperation struct {
	dag.OpBase
	ColumnId entity.Id         `json:"column"`
	Title    string            `json:"title"`
	Message  string            `json:"message"`
	Files    []repository.Hash `json:"files"`
}

func (op *AddItemDraftOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *AddItemDraftOperation) GetFiles() []repository.Hash {
	return op.Files
}

func (op *AddItemDraftOperation) Validate() error {
	if err := op.OpBase.Validate(op, AddItemDraftOp); err != nil {
		return err
	}

	if err := op.ColumnId.Validate(); err != nil {
		return err
	}

	if text.Empty(op.Title) {
		return fmt.Errorf("title is empty")
	}
	if !text.SafeOneLine(op.Title) {
		return fmt.Errorf("title has unsafe characters")
	}

	if !text.Safe(op.Message) {
		return fmt.Errorf("message is not fully printable")
	}

	for _, file := range op.Files {
		if !file.IsValid() {
			return fmt.Errorf("invalid file hash")
		}
	}

	return nil
}

func (op *AddItemDraftOperation) Apply(snapshot *Snapshot) {
	// Recreate the combined Id to match on
	combinedId := entity.CombineIds(snapshot.Id(), op.ColumnId)

	for _, column := range snapshot.Columns {
		if column.CombinedId == combinedId {
			column.Items = append(column.Items, &Draft{
				combinedId: entity.CombineIds(snapshot.id, op.Id()),
				Author:     op.Author(),
				Title:      op.Title,
				Message:    op.Message,
				unixTime:   timestamp.Timestamp(op.UnixTime),
			})

			snapshot.addParticipant(op.Author())
			return
		}
	}
}

func NewAddItemDraftOp(author identity.Interface, unixTime int64, columnId entity.Id, title, message string, files []repository.Hash) *AddItemDraftOperation {
	return &AddItemDraftOperation{
		OpBase:   dag.NewOpBase(AddItemDraftOp, author, unixTime),
		ColumnId: columnId,
		Title:    title,
		Message:  message,
		Files:    files,
	}
}

// AddItemDraft is a convenience function to add a draft item to a Board
func AddItemDraft(b Interface, author identity.Interface, unixTime int64, columnId entity.Id, title, message string, files []repository.Hash, metadata map[string]string) (entity.CombinedId, *AddItemDraftOperation, error) {
	op := NewAddItemDraftOp(author, unixTime, columnId, title, message, files)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return entity.UnsetCombinedId, nil, err
	}
	b.Append(op)
	return entity.CombineIds(b.Id(), op.Id()), op, nil
}
