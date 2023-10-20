package board

import (
	"fmt"

	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
)

// ItemEntityType indicate the type of entity board item
type ItemEntityType string

const (
	EntityTypeBug ItemEntityType = "bug"
)

var _ Operation = &AddItemEntityOperation{}

type AddItemEntityOperation struct {
	dag.OpBase
	ColumnId   entity.Id        `json:"column"`
	EntityType ItemEntityType   `json:"entity_type"`
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
	case EntityTypeBug:
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

	// Recreate the combined Id to match on
	combinedId := entity.CombineIds(snapshot.Id(), op.ColumnId)

	for _, column := range snapshot.Columns {
		if column.CombinedId == combinedId {
			switch op.EntityType {
			case EntityTypeBug:
				column.Items = append(column.Items, &BugItem{
					combinedId: entity.CombineIds(snapshot.Id(), op.entity.Id()),
					Bug:        op.entity.(bug.Interface),
				})
			}
			snapshot.addParticipant(op.Author())
			return
		}
	}
}

func NewAddItemEntityOp(author identity.Interface, unixTime int64, columnId entity.Id, entityType ItemEntityType, e entity.Interface) *AddItemEntityOperation {
	// Note: due to import cycle we are not able to properly check the type of the entity here;
	// proceed with caution!
	return &AddItemEntityOperation{
		OpBase:     dag.NewOpBase(AddItemEntityOp, author, unixTime),
		ColumnId:   columnId,
		EntityType: entityType,
		EntityId:   e.Id(),
		entity:     e,
	}
}

// AddItemEntity is a convenience function to add an entity item to a Board
func AddItemEntity(b Interface, author identity.Interface, unixTime int64, columnId entity.Id, entityType ItemEntityType, e entity.Interface, metadata map[string]string) (entity.CombinedId, *AddItemEntityOperation, error) {
	op := NewAddItemEntityOp(author, unixTime, columnId, entityType, e)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return entity.UnsetCombinedId, nil, err
	}
	b.Append(op)
	return entity.CombineIds(b.Id(), op.Id()), op, nil
}
