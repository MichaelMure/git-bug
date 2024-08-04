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

var _ Operation = &CreateOperation{}
var _ entity.OperationWithFiles = &CreateOperation{}

// CreateOperation define the initial creation of a bug
type CreateOperation struct {
	dag.OpBase
	Title   string            `json:"title"`
	Message string            `json:"message"`
	Files   []repository.Hash `json:"files"`
}

func (op *CreateOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *CreateOperation) Apply(snapshot *Snapshot) {
	// sanity check: will fail when adding a second Create
	if snapshot.id != "" && snapshot.id != entity.UnsetId && snapshot.id != op.Id() {
		return
	}

	// the Id of the Bug/Snapshot is the Id of the first Operation: CreateOperation
	opId := op.Id()
	snapshot.id = opId

	snapshot.addActor(op.Author())
	snapshot.addParticipant(op.Author())

	snapshot.Title = op.Title

	comment := Comment{
		combinedId: entity.CombineIds(snapshot.id, opId),
		targetId:   opId,
		Message:    op.Message,
		Author:     op.Author(),
		unixTime:   timestamp.Timestamp(op.UnixTime),
	}

	snapshot.Comments = []Comment{comment}
	snapshot.Author = op.Author()
	snapshot.CreateTime = op.Time()

	snapshot.Timeline = []TimelineItem{
		&CreateTimelineItem{
			CommentTimelineItem: NewCommentTimelineItem(comment),
		},
	}
}

func (op *CreateOperation) GetFiles() []repository.Hash {
	return op.Files
}

func (op *CreateOperation) Validate() error {
	if err := op.OpBase.Validate(op, CreateOp); err != nil {
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

func NewCreateOp(author identity.Interface, unixTime int64, title, message string, files []repository.Hash) *CreateOperation {
	return &CreateOperation{
		OpBase:  dag.NewOpBase(CreateOp, author, unixTime),
		Title:   title,
		Message: message,
		Files:   files,
	}
}

// CreateTimelineItem replace a Create operation in the Timeline and hold its edition history
type CreateTimelineItem struct {
	CommentTimelineItem
}

// IsAuthored is a sign post method for gqlgen
func (c *CreateTimelineItem) IsAuthored() {}

// Create is a convenience function to create a bug
func Create(author identity.Interface, unixTime int64, title, message string, files []repository.Hash, metadata map[string]string) (*Bug, *CreateOperation, error) {
	b := NewBug()
	op := NewCreateOp(author, unixTime, title, message, files)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return nil, op, err
	}
	b.Append(op)
	return b, op, nil
}
