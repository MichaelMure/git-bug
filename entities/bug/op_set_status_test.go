package bug

import (
	"testing"

	"github.com/MichaelMure/git-bug/entities/common"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
)

func TestSetStatusSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author entity.Identity, unixTime int64) (*SetStatusOperation, entity.Resolvers) {
		return NewSetStatusOp(author, unixTime, common.ClosedStatus), nil
	})
}
