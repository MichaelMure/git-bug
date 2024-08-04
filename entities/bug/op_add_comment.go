package bug

import (
	"fmt"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/text"
	"github.com/MichaelMure/git-bug/util/timestamp"
)

var _ Operation = &AddCommentOperation{}
var _ dag.Operation = &AddCommentOperation{}
var _ entity.OperationWithFiles = &AddCommentOperation{}

// AddCommentOperation will add a new comment in the bug
type AddCommentOperation struct {
	dag.OpBase
	Message string `json:"message"`
	// TODO: change for a map[string]util.hash to store the filename ?
	Files []repository.Hash `json:"files"`
}

func (op *AddCommentOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *AddCommentOperation) Apply(snapshot *Snapshot) {
	snapshot.addActor(op.Author())
	snapshot.addParticipant(op.Author())

	opId := op.Id()

	comment := Comment{
		combinedId: entity.CombineIds(snapshot.Id(), opId),
		targetId:   opId,
		Message:    op.Message,
		Author:     op.Author(),
		Files:      op.Files,
		unixTime:   timestamp.Timestamp(op.UnixTime),
	}

	snapshot.Comments = append(snapshot.Comments, comment)

	item := &AddCommentTimelineItem{
		CommentTimelineItem: NewCommentTimelineItem(comment),
	}

	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *AddCommentOperation) GetFiles() []repository.Hash {
	return op.Files
}

func (op *AddCommentOperation) Validate() error {
	if err := op.OpBase.Validate(op, AddCommentOp); err != nil {
		return err
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

func NewAddCommentOp(author identity.Interface, unixTime int64, message string, files []repository.Hash) *AddCommentOperation {
	return &AddCommentOperation{
		OpBase:  dag.NewOpBase(AddCommentOp, author, unixTime),
		Message: message,
		Files:   files,
	}
}

// AddCommentTimelineItem replace a AddComment operation in the Timeline and hold its edition history
type AddCommentTimelineItem struct {
	CommentTimelineItem
}

// IsAuthored is a sign post method for gqlgen
func (a *AddCommentTimelineItem) IsAuthored() {}

// AddComment is a convenience function to add a comment to a bug
func AddComment(b Interface, author identity.Interface, unixTime int64, message string, files []repository.Hash, metadata map[string]string) (entity.CombinedId, *AddCommentOperation, error) {
	op := NewAddCommentOp(author, unixTime, message, files)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return entity.UnsetCombinedId, nil, err
	}
	b.Append(op)
	return entity.CombineIds(b.Id(), op.Id()), op, nil
}
