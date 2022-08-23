package board

import (
	"fmt"

	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
)

// itemEntityType indicate the type of entity board item
type itemEntityType string

const (
	entityTypeBug itemEntityType = "bug"
)

var _ Operation = &AddItemEntityOperation{}

type AddItemEntityOperation struct {
	dag.OpBase
	ColumnId   entity.Id        `json:"column"`
	EntityType itemEntityType   `json:"entity_type"`
	EntityId   entity.Id        `json:"entity_id"`
	entity     entity.Interface // not serialized
}

func (op *AddItemEntityOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *AddItemEntityOperation) Validate() error {
	if err := op.OpBase.Validate(op, AddItemEntityOp); err != nil {
		return err
	}

	if err := op.ColumnId.Validate(); err != nil {
		return err
	}

	switch op.EntityType {
	case entityTypeBug:
	default:
		return fmt.Errorf("unknown entity type")
	}

	if err := op.EntityId.Validate(); err != nil {
		return err
	}

	return nil
}

func (op *AddItemEntityOperation) Apply(snapshot *Snapshot) {
	if op.entity == nil {
		return
	}

	snapshot.addActor(op.Author())

	for _, column := range snapshot.Columns {
		if column.Id == op.ColumnId {
			switch e := op.entity.(type) {
			case bug.Interface:
				column.Items = append(column.Items, &BugItem{
					combinedId: entity.CombineIds(snapshot.Id(), e.Id()),
					bug:        e,
				})
			}
			return
		}
	}
}

func NewAddItemEntityOp(author identity.Interface, unixTime int64, columnId entity.Id, e entity.Interface) *AddItemEntityOperation {
	switch e := e.(type) {
	case bug.Interface:
		return &AddItemEntityOperation{
			OpBase:     dag.NewOpBase(AddItemEntityOp, author, unixTime),
			ColumnId:   columnId,
			EntityType: entityTypeBug,
			EntityId:   e.Id(),
			entity:     e,
		}
	default:
		panic("invalid entity type")
	}
}

// AddItemEntity is a convenience function to add an entity item to a Board
func AddItemEntity(b *Board, author identity.Interface, unixTime int64, columnId entity.Id, e entity.Interface, metadata map[string]string) (*AddItemEntityOperation, error) {
	op := NewAddItemEntityOp(author, unixTime, columnId, e)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}
