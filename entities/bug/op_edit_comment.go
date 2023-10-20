package bug

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/timestamp"

	"github.com/MichaelMure/git-bug/util/text"
)

var _ Operation = &EditCommentOperation{}
var _ entity.OperationWithFiles = &EditCommentOperation{}

// EditCommentOperation will change a comment in the bug
type EditCommentOperation struct {
	dag.OpBase
	Target  entity.Id         `json:"target"`
	Message string            `json:"message"`
	Files   []repository.Hash `json:"files"`
}

func (op *EditCommentOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *EditCommentOperation) Apply(snapshot *Snapshot) {
	// Todo: currently any message can be edited, even by a different author
	// crypto signature are needed.

	// Recreate the combined Id to match on
	combinedId := entity.CombineIds(snapshot.Id(), op.Target)

	var target TimelineItem
	for i, item := range snapshot.Timeline {
		if item.CombinedId() == combinedId {
			target = snapshot.Timeline[i]
			break
		}
	}

	if target == nil {
		// Target not found, edit is a no-op
		return
	}

	comment := Comment{
		combinedId: combinedId,
		targetId:   op.Target,
		Message:    op.Message,
		Files:      op.Files,
		unixTime:   timestamp.Timestamp(op.UnixTime),
	}

	switch target := target.(type) {
	case *CreateTimelineItem:
		target.Append(comment)
	case *AddCommentTimelineItem:
		target.Append(comment)
	default:
		// somehow, the target matched on something that is not a comment
		// we make the op a no-op
		return
	}

	snapshot.addActor(op.Author())

	// Updating the corresponding comment

	for i := range snapshot.Comments {
		if snapshot.Comments[i].CombinedId() == combinedId {
			snapshot.Comments[i].Message = op.Message
			snapshot.Comments[i].Files = op.Files
			break
		}
	}
}

func (op *EditCommentOperation) GetFiles() []repository.Hash {
	return op.Files
}

func (op *EditCommentOperation) Validate() error {
	if err := op.OpBase.Validate(op, EditCommentOp); err != nil {
		return err
	}

	if err := op.Target.Validate(); err != nil {
		return errors.Wrap(err, "target hash is invalid")
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

func NewEditCommentOp(author identity.Interface, unixTime int64, target entity.Id, message string, files []repository.Hash) *EditCommentOperation {
	return &EditCommentOperation{
		OpBase:  dag.NewOpBase(EditCommentOp, author, unixTime),
		Target:  target,
		Message: message,
		Files:   files,
	}
}

// EditComment is a convenience function to apply the operation
func EditComment(b Interface, author identity.Interface, unixTime int64, target entity.Id, message string, files []repository.Hash, metadata map[string]string) (entity.CombinedId, *EditCommentOperation, error) {
	op := NewEditCommentOp(author, unixTime, target, message, files)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return entity.UnsetCombinedId, nil, err
	}
	b.Append(op)
	return entity.CombineIds(b.Id(), target), op, nil
}

// EditCreateComment is a convenience function to edit the body of a bug (the first comment)
func EditCreateComment(b Interface, author identity.Interface, unixTime int64, message string, files []repository.Hash, metadata map[string]string) (entity.CombinedId, *EditCommentOperation, error) {
	createOp := b.FirstOp().(*CreateOperation)
	return EditComment(b, author, unixTime, createOp.Id(), message, files, metadata)
}
