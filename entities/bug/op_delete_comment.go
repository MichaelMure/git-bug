package bug

import (
	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/timestamp"
	"github.com/pkg/errors"
)

var _ Operation = &DeleteCommentOperation{}

// DeleteCommentOperation will change a comment in the bug
type DeleteCommentOperation struct {
	dag.OpBase
	Target entity.Id `json:"target"`
}

func (op *DeleteCommentOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op DeleteCommentOperation) Apply(snapshot *Snapshot) {
	// TODO: who can delete?

	cid := entity.CombineIds(snapshot.Id(), op.Target)

	// update timeline
	var target TimelineItem
	for i, item := range snapshot.Timeline {
		if item.CombinedId() == cid {
			target = snapshot.Timeline[i]
			break
		}
	}

	comment := Comment{
		combinedId: cid,
		targetId:   op.Target,
		unixTime:   timestamp.Timestamp(op.UnixTime),
		Author:     op.Author(),
		Deleted:    true,
	}

	switch target := target.(type) {
	case *CreateTimelineItem:
		target.Append(comment)
	case *AddCommentTimelineItem:
		target.Append(comment)
	default:
		// no-op
		return
	}

	snapshot.addActor(op.Author())

	// update snapshot
	for _, c := range snapshot.Comments {
		if c.CombinedId() == cid {
			c.Deleted = true
			c.Message = ""
			c.Files = []repository.Hash{}
			break
		}
	}
}

func (op *DeleteCommentOperation) Validate() error {
	if err := op.OpBase.Validate(op, DeleteCommentOp); err != nil {
		return err
	}

	if err := op.Target.Validate(); err != nil {
		return errors.Wrap(err, "target hash is invalid")
	}

	return nil
}

func NewDeleteCommentOp(author identity.Interface, unixTime int64, target entity.Id) *DeleteCommentOperation {
	return &DeleteCommentOperation{
		OpBase: dag.NewOpBase(DeleteCommentOp, author, unixTime),
		Target: target,
	}
}

func DeleteComment(b Interface, author identity.Interface, unixTime int64, target entity.Id, metadata map[string]string) (entity.CombinedId, *DeleteCommentOperation, error) {
	op := NewDeleteCommentOp(author, unixTime, target)
	for k, v := range metadata {
		op.SetMetadata(k, v)
	}

	if err := op.Validate(); err != nil {
		return entity.UnsetCombinedId, nil, err
	}

	b.Append(op)

	return entity.CombineIds(b.Id(), target), op, nil
}
