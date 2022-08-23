package board

import (
	"testing"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
)

func TestSetDescriptionSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*SetDescriptionOperation, entity.Resolvers) {
		return NewSetDescriptionOp(author, unixTime, "description", "was"), nil
	})
}
