package resolvers

import (
	"context"
	"time"

	"github.com/MichaelMure/git-bug/api/auth"
	"github.com/MichaelMure/git-bug/api/graphql/graph"
	"github.com/MichaelMure/git-bug/api/graphql/models"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/util/text"
)

var _ graph.MutationResolver = &mutationResolver{}

type mutationResolver struct {
	cache *cache.MultiRepoCache
}

func (r mutationResolver) getRepo(ref *string) (*cache.RepoCache, error) {
	if ref != nil {
		return r.cache.ResolveRepo(*ref)
	}

	return r.cache.DefaultRepo()
}

func (r mutationResolver) getBug(repoRef *string, bugPrefix string) (*cache.RepoCache, *cache.BugCache, error) {
	repo, err := r.getRepo(repoRef)
	if err != nil {
		return nil, nil, err
	}

	b, err := repo.Bugs().ResolvePrefix(bugPrefix)
	if err != nil {
		return nil, nil, err
	}
	return repo, b, nil
}

func (r mutationResolver) NewBug(ctx context.Context, input models.NewBugInput) (*models.NewBugPayload, error) {
	repo, err := r.getRepo(input.RepoRef)
	if err != nil {
		return nil, err
	}

	author, err := auth.UserFromCtx(ctx, repo)
	if err != nil {
		return nil, err
	}

	b, op, err := repo.Bugs().NewRaw(author,
		time.Now().Unix(),
		text.CleanupOneLine(input.Title),
		text.Cleanup(input.Message),
		input.Files,
		nil)
	if err != nil {
		return nil, err
	}

	return &models.NewBugPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Compile()),
		Operation:        op,
	}, nil
}

func (r mutationResolver) AddComment(ctx context.Context, input models.AddCommentInput) (*models.AddCommentPayload, error) {
	repo, b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	author, err := auth.UserFromCtx(ctx, repo)
	if err != nil {
		return nil, err
	}

	_, op, err := b.AddCommentRaw(author,
		time.Now().Unix(),
		text.Cleanup(input.Message),
		input.Files,
		nil)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.AddCommentPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Compile()),
		Operation:        op,
	}, nil
}

func (r mutationResolver) AddCommentAndClose(ctx context.Context, input models.AddCommentAndCloseBugInput) (*models.AddCommentAndCloseBugPayload, error) {
	repo, b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	author, err := auth.UserFromCtx(ctx, repo)
	if err != nil {
		return nil, err
	}

	_, opAddComment, err := b.AddCommentRaw(author,
		time.Now().Unix(),
		text.Cleanup(input.Message),
		input.Files,
		nil)
	if err != nil {
		return nil, err
	}

	opClose, err := b.CloseRaw(author, time.Now().Unix(), nil)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.AddCommentAndCloseBugPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Compile()),
		CommentOperation: opAddComment,
		StatusOperation:  opClose,
	}, nil
}

func (r mutationResolver) AddCommentAndReopen(ctx context.Context, input models.AddCommentAndReopenBugInput) (*models.AddCommentAndReopenBugPayload, error) {
	repo, b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	author, err := auth.UserFromCtx(ctx, repo)
	if err != nil {
		return nil, err
	}

	_, opAddComment, err := b.AddCommentRaw(author,
		time.Now().Unix(),
		text.Cleanup(input.Message),
		input.Files,
		nil)
	if err != nil {
		return nil, err
	}

	opReopen, err := b.OpenRaw(author, time.Now().Unix(), nil)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.AddCommentAndReopenBugPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Compile()),
		CommentOperation: opAddComment,
		StatusOperation:  opReopen,
	}, nil
}

func (r mutationResolver) EditComment(ctx context.Context, input models.EditCommentInput) (*models.EditCommentPayload, error) {
	repo, err := r.getRepo(input.RepoRef)
	if err != nil {
		return nil, err
	}

	b, target, err := repo.Bugs().ResolveComment(input.TargetPrefix)
	if err != nil {
		return nil, err
	}

	author, err := auth.UserFromCtx(ctx, repo)
	if err != nil {
		return nil, err
	}

	op, err := b.EditCommentRaw(
		author,
		time.Now().Unix(),
		target,
		text.Cleanup(input.Message),
		nil,
	)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.EditCommentPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Compile()),
		Operation:        op,
	}, nil
}

func (r mutationResolver) ChangeLabels(ctx context.Context, input *models.ChangeLabelInput) (*models.ChangeLabelPayload, error) {
	repo, b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	author, err := auth.UserFromCtx(ctx, repo)
	if err != nil {
		return nil, err
	}

	results, op, err := b.ChangeLabelsRaw(
		author,
		time.Now().Unix(),
		text.CleanupOneLineArray(input.Added),
		text.CleanupOneLineArray(input.Removed),
		nil,
	)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	resultsPtr := make([]*bug.LabelChangeResult, len(results))
	for i, result := range results {
		resultsPtr[i] = &result
	}

	return &models.ChangeLabelPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Compile()),
		Operation:        op,
		Results:          resultsPtr,
	}, nil
}

func (r mutationResolver) OpenBug(ctx context.Context, input models.OpenBugInput) (*models.OpenBugPayload, error) {
	repo, b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	author, err := auth.UserFromCtx(ctx, repo)
	if err != nil {
		return nil, err
	}

	op, err := b.OpenRaw(author, time.Now().Unix(), nil)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.OpenBugPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Compile()),
		Operation:        op,
	}, nil
}

func (r mutationResolver) CloseBug(ctx context.Context, input models.CloseBugInput) (*models.CloseBugPayload, error) {
	repo, b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	author, err := auth.UserFromCtx(ctx, repo)
	if err != nil {
		return nil, err
	}

	op, err := b.CloseRaw(author, time.Now().Unix(), nil)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.CloseBugPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Compile()),
		Operation:        op,
	}, nil
}

func (r mutationResolver) SetTitle(ctx context.Context, input models.SetTitleInput) (*models.SetTitlePayload, error) {
	repo, b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	author, err := auth.UserFromCtx(ctx, repo)
	if err != nil {
		return nil, err
	}

	op, err := b.SetTitleRaw(
		author,
		time.Now().Unix(),
		text.CleanupOneLine(input.Title),
		nil,
	)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.SetTitlePayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Compile()),
		Operation:        op,
	}, nil
}
