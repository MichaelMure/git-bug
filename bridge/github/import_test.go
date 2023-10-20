package github

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/entities/common"
	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func TestGithubImporter(t *testing.T) {
	envToken := os.Getenv("GITHUB_TOKEN_PRIVATE")
	if envToken == "" {
		t.Skip("Env var GITHUB_TOKEN_PRIVATE missing")
	}

	repo := repository.CreateGoGitTestRepo(t, false)

	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)

	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	author, err := identity.NewIdentity(repo, "Michael Muré", "batolettre@gmail.com")
	require.NoError(t, err)

	tests := []struct {
		name string
		url  string
		bug  *bug.Snapshot
	}{
		{
			name: "simple issue",
			url:  "https://github.com/MichaelMure/git-bug-test-github-bridge/issues/1",
			bug: &bug.Snapshot{
				Operations: []dag.Operation{
					bug.NewCreateOp(author, 0, "simple issue", "initial comment", nil),
					bug.NewAddCommentOp(author, 0, "first comment", nil),
					bug.NewAddCommentOp(author, 0, "second comment", nil),
				},
			},
		},
		{
			name: "empty issue",
			url:  "https://github.com/MichaelMure/git-bug-test-github-bridge/issues/2",
			bug: &bug.Snapshot{
				Operations: []dag.Operation{
					bug.NewCreateOp(author, 0, "empty issue", "", nil),
				},
			},
		},
		{
			name: "complex issue",
			url:  "https://github.com/MichaelMure/git-bug-test-github-bridge/issues/3",
			bug: &bug.Snapshot{
				Operations: []dag.Operation{
					bug.NewCreateOp(author, 0, "complex issue", "initial comment", nil),
					bug.NewLabelChangeOperation(author, 0, []bug.Label{"bug"}, []bug.Label{}),
					bug.NewLabelChangeOperation(author, 0, []bug.Label{"duplicate"}, []bug.Label{}),
					bug.NewLabelChangeOperation(author, 0, []bug.Label{}, []bug.Label{"duplicate"}),
					bug.NewAddCommentOp(author, 0, "### header\n\n**bold**\n\n_italic_\n\n> with quote\n\n`inline code`\n\n```\nmultiline code\n```\n\n- bulleted\n- list\n\n1. numbered\n1. list\n\n- [ ] task\n- [x] list\n\n@MichaelMure mention\n\n#2 reference issue\n#3 auto-reference issue\n\n![image](https://user-images.githubusercontent.com/294669/56870222-811faf80-6a0c-11e9-8f2c-f0beb686303f.png)", nil),
					bug.NewSetTitleOp(author, 0, "complex issue edited", "complex issue"),
					bug.NewSetTitleOp(author, 0, "complex issue", "complex issue edited"),
					bug.NewSetStatusOp(author, 0, common.ClosedStatus),
					bug.NewSetStatusOp(author, 0, common.OpenStatus),
				},
			},
		},
		{
			name: "editions",
			url:  "https://github.com/MichaelMure/git-bug-test-github-bridge/issues/4",
			bug: &bug.Snapshot{
				Operations: []dag.Operation{
					bug.NewCreateOp(author, 0, "editions", "initial comment edited", nil),
					bug.NewEditCommentOp(author, 0, "", "erased then edited again", nil),
					bug.NewAddCommentOp(author, 0, "first comment", nil),
					bug.NewEditCommentOp(author, 0, "", "first comment edited", nil),
				},
			},
		},
		{
			name: "comment deletion",
			url:  "https://github.com/MichaelMure/git-bug-test-github-bridge/issues/5",
			bug: &bug.Snapshot{
				Operations: []dag.Operation{
					bug.NewCreateOp(author, 0, "comment deletion", "", nil),
				},
			},
		},
		{
			name: "edition deletion",
			url:  "https://github.com/MichaelMure/git-bug-test-github-bridge/issues/6",
			bug: &bug.Snapshot{
				Operations: []dag.Operation{
					bug.NewCreateOp(author, 0, "edition deletion", "initial comment", nil),
					bug.NewEditCommentOp(author, 0, "", "initial comment edited again", nil),
					bug.NewAddCommentOp(author, 0, "first comment", nil),
					bug.NewEditCommentOp(author, 0, "", "first comment edited again", nil),
				},
			},
		},
		{
			name: "hidden comment",
			url:  "https://github.com/MichaelMure/git-bug-test-github-bridge/issues/7",
			bug: &bug.Snapshot{
				Operations: []dag.Operation{
					bug.NewCreateOp(author, 0, "hidden comment", "initial comment", nil),
					bug.NewAddCommentOp(author, 0, "first comment", nil),
				},
			},
		},
		{
			name: "transferred issue",
			url:  "https://github.com/MichaelMure/git-bug-test-github-bridge/issues/8",
			bug: &bug.Snapshot{
				Operations: []dag.Operation{
					bug.NewCreateOp(author, 0, "transfered issue", "", nil),
				},
			},
		},
		{
			name: "unicode control characters",
			url:  "https://github.com/MichaelMure/git-bug-test-github-bridge/issues/10",
			bug: &bug.Snapshot{
				Operations: []dag.Operation{
					bug.NewCreateOp(author, 0, "unicode control characters", "u0000: \nu0001: \nu0002: \nu0003: \nu0004: \nu0005: \nu0006: \nu0007: \nu0008: \nu0009: \t\nu0010: \nu0011: \nu0012: \nu0013: \nu0014: \nu0015: \nu0016: \nu0017: \nu0018: \nu0019:", nil),
				},
			},
		},
	}

	login := "test-identity"
	author.SetMetadata(metaKeyGithubLogin, login)

	token := auth.NewToken(target, envToken)
	token.SetMetadata(auth.MetaKeyLogin, login)
	err = auth.Store(repo, token)
	require.NoError(t, err)

	ctx := context.Background()

	importer := &githubImporter{}
	err = importer.Init(ctx, backend, core.Configuration{
		confKeyOwner:        "MichaelMure",
		confKeyProject:      "git-bug-test-github-bridge",
		confKeyDefaultLogin: login,
	})
	require.NoError(t, err)

	start := time.Now()

	events, err := importer.ImportAll(ctx, backend, time.Time{})
	require.NoError(t, err)

	for result := range events {
		require.NoError(t, result.Err)
	}

	fmt.Printf("test repository imported in %f seconds\n", time.Since(start).Seconds())

	require.Len(t, backend.Bugs().AllIds(), len(tests))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := backend.Bugs().ResolveBugCreateMetadata(metaKeyGithubUrl, tt.url)
			require.NoError(t, err)

			ops := b.Compile().Operations
			require.Len(t, tt.bug.Operations, len(b.Compile().Operations))

			for i, op := range tt.bug.Operations {
				require.IsType(t, ops[i], op)
				require.Equal(t, op.Author().Name(), ops[i].Author().Name())

				switch op := op.(type) {
				case *bug.CreateOperation:
					require.Equal(t, op.Title, ops[i].(*bug.CreateOperation).Title)
					require.Equal(t, op.Message, ops[i].(*bug.CreateOperation).Message)
				case *bug.SetStatusOperation:
					require.Equal(t, op.Status, ops[i].(*bug.SetStatusOperation).Status)
				case *bug.SetTitleOperation:
					require.Equal(t, op.Was, ops[i].(*bug.SetTitleOperation).Was)
					require.Equal(t, op.Title, ops[i].(*bug.SetTitleOperation).Title)
				case *bug.LabelChangeOperation:
					require.ElementsMatch(t, op.Added, ops[i].(*bug.LabelChangeOperation).Added)
					require.ElementsMatch(t, op.Removed, ops[i].(*bug.LabelChangeOperation).Removed)
				case *bug.AddCommentOperation:
					require.Equal(t, op.Message, ops[i].(*bug.AddCommentOperation).Message)
				case *bug.EditCommentOperation:
					require.Equal(t, op.Message, ops[i].(*bug.EditCommentOperation).Message)

				default:
					panic("unknown operation type")
				}
			}
		})
	}
}
