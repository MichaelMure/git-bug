package resolvers

import (
	"context"

	"github.com/MichaelMure/git-bug/api/graphql/graph"
	"github.com/MichaelMure/git-bug/api/graphql/models"
	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/entity"
)

var _ graph.CommentResolver = &commentResolver{}

type commentResolver struct{}

func (c commentResolver) ID(ctx context.Context, obj *bug.Comment) (entity.CombinedId, error) {
	return obj.CombinedId(), nil
}

func (c commentResolver) Author(_ context.Context, obj *bug.Comment) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author), nil
}
