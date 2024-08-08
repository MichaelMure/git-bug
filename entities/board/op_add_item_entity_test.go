package board

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
)

func TestAddItemEntityOpSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*AddItemEntityOperation, entity.Resolvers) {
		b, _, err := bug.Create(author, unixTime, "title", "message", nil, nil)
		require.NoError(t, err)

		resolvers := entity.Resolvers{
			&bug.Bug{}: entity.MakeResolver(b),
		}

		return NewAddItemEntityOp(author, unixTime, "foo", b), resolvers
	})
}
