package gitlab

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/xanzy/go-gitlab"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

const (
	testRepoBaseName = "git-bug-test-gitlab-exporter"
)

type testCase struct {
	name     string
	bug      *cache.BugCache
	numOp    int // number of original operations
	numOpExp int // number of operations after export
	numOpImp int // number of operations after import
}

func testCases(t *testing.T, repo *cache.RepoCache) []*testCase {
	// simple bug
	simpleBug, _, err := repo.NewBug("simple bug", "new bug")
	require.NoError(t, err)

	// bug with comments
	bugWithComments, _, err := repo.NewBug("bug with comments", "new bug")
	require.NoError(t, err)

	_, err = bugWithComments.AddComment("new comment")
	require.NoError(t, err)

	// bug with label changes
	bugLabelChange, _, err := repo.NewBug("bug label change", "new bug")
	require.NoError(t, err)

	_, _, err = bugLabelChange.ChangeLabels([]string{"bug"}, nil)
	require.NoError(t, err)

	_, _, err = bugLabelChange.ChangeLabels([]string{"core"}, nil)
	require.NoError(t, err)

	_, _, err = bugLabelChange.ChangeLabels(nil, []string{"bug"})
	require.NoError(t, err)

	// bug with comments editions
	bugWithCommentEditions, createOp, err := repo.NewBug("bug with comments editions", "new bug")
	require.NoError(t, err)

	_, err = bugWithCommentEditions.EditComment(createOp.Id(), "first comment edited")
	require.NoError(t, err)

	commentOp, err := bugWithCommentEditions.AddComment("first comment")
	require.NoError(t, err)

	_, err = bugWithCommentEditions.EditComment(commentOp.Id(), "first comment edited")
	require.NoError(t, err)

	// bug status changed
	bugStatusChanged, _, err := repo.NewBug("bug status changed", "new bug")
	require.NoError(t, err)

	_, err = bugStatusChanged.Close()
	require.NoError(t, err)

	_, err = bugStatusChanged.Open()
	require.NoError(t, err)

	// bug title changed
	bugTitleEdited, _, err := repo.NewBug("bug title edited", "new bug")
	require.NoError(t, err)

	_, err = bugTitleEdited.SetTitle("bug title edited again")
	require.NoError(t, err)

	return []*testCase{
		&testCase{
			name:     "simple bug",
			bug:      simpleBug,
			numOp:    1,
			numOpExp: 2,
			numOpImp: 1,
		},
		&testCase{
			name:     "bug with comments",
			bug:      bugWithComments,
			numOp:    2,
			numOpExp: 4,
			numOpImp: 2,
		},
		&testCase{
			name:     "bug label change",
			bug:      bugLabelChange,
			numOp:    4,
			numOpExp: 8,
			numOpImp: 4,
		},
		&testCase{
			name:     "bug with comment editions",
			bug:      bugWithCommentEditions,
			numOp:    4,
			numOpExp: 8,
			numOpImp: 2,
		},
		&testCase{
			name:     "bug changed status",
			bug:      bugStatusChanged,
			numOp:    3,
			numOpExp: 6,
			numOpImp: 3,
		},
		&testCase{
			name:     "bug title edited",
			bug:      bugTitleEdited,
			numOp:    2,
			numOpExp: 4,
			numOpImp: 2,
		},
	}
}

func TestGitlabPushPull(t *testing.T) {
	// token must have 'repo' and 'delete_repo' scopes
	envToken := os.Getenv("GITLAB_API_TOKEN")
	if envToken == "" {
		t.Skip("Env var GITLAB_API_TOKEN missing")
	}

	// create repo backend
	repo := repository.CreateGoGitTestRepo(t, false)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	// set author identity
	login := "test-identity"
	author, err := backend.NewIdentity("test identity", "test@test.org")
	require.NoError(t, err)
	author.SetMetadata(metaKeyGitlabLogin, login)
	err = author.Commit()
	require.NoError(t, err)

	err = backend.SetUserIdentity(author)
	require.NoError(t, err)

	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	token := auth.NewToken(target, envToken)
	token.SetMetadata(auth.MetaKeyLogin, login)
	token.SetMetadata(auth.MetaKeyBaseURL, defaultBaseURL)
	err = auth.Store(repo, token)
	require.NoError(t, err)

	tests := testCases(t, backend)

	// generate project name
	projectName := generateRepoName()

	// create target Gitlab repository
	projectID, err := createRepository(context.TODO(), projectName, token)
	require.NoError(t, err)

	fmt.Println("created repository", projectName)

	// Make sure to remove the Gitlab repository when the test end
	defer func(t *testing.T) {
		if err := deleteRepository(context.TODO(), projectID, token); err != nil {
			t.Fatal(err)
		}
		fmt.Println("deleted repository:", projectName)
	}(t)

	interrupt.RegisterCleaner(func() error {
		return deleteRepository(context.TODO(), projectID, token)
	})

	ctx := context.Background()

	// initialize exporter
	exporter := &gitlabExporter{}
	err = exporter.Init(ctx, backend, core.Configuration{
		confKeyProjectID:     strconv.Itoa(projectID),
		confKeyGitlabBaseUrl: defaultBaseURL,
		confKeyDefaultLogin:  login,
	})
	require.NoError(t, err)

	start := time.Now()

	// export all bugs
	exportEvents, err := exporter.ExportAll(ctx, backend, time.Time{})
	require.NoError(t, err)

	for result := range exportEvents {
		require.NoError(t, result.Err)
	}
	require.NoError(t, err)

	fmt.Printf("test repository exported in %f seconds\n", time.Since(start).Seconds())

	repoTwo := repository.CreateGoGitTestRepo(t, false)

	// create a second backend
	backendTwo, err := cache.NewRepoCache(repoTwo)
	require.NoError(t, err)

	importer := &gitlabImporter{}
	err = importer.Init(ctx, backend, core.Configuration{
		confKeyProjectID:     strconv.Itoa(projectID),
		confKeyGitlabBaseUrl: defaultBaseURL,
		confKeyDefaultLogin:  login,
	})
	require.NoError(t, err)

	// import all exported bugs to the second backend
	importEvents, err := importer.ImportAll(ctx, backendTwo, time.Time{})
	require.NoError(t, err)

	for result := range importEvents {
		require.NoError(t, result.Err)
	}

	require.Len(t, backendTwo.AllBugsIds(), len(tests))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// for each operation a SetMetadataOperation will be added
			// so number of operations should double
			require.Len(t, tt.bug.Snapshot().Operations, tt.numOpExp)

			// verify operation have correct metadata
			for _, op := range tt.bug.Snapshot().Operations {
				// Check if the originals operations (*not* SetMetadata) are tagged properly
				if _, ok := op.(*bug.SetMetadataOperation); !ok {
					_, haveIDMetadata := op.GetMetadata(metaKeyGitlabId)
					require.True(t, haveIDMetadata)

					_, haveURLMetada := op.GetMetadata(metaKeyGitlabUrl)
					require.True(t, haveURLMetada)
				}
			}

			// get bug gitlab ID
			bugGitlabID, ok := tt.bug.Snapshot().GetCreateMetadata(metaKeyGitlabId)
			require.True(t, ok)

			// retrieve bug from backendTwo
			importedBug, err := backendTwo.ResolveBugCreateMetadata(metaKeyGitlabId, bugGitlabID)
			require.NoError(t, err)

			// verify bug have same number of original operations
			require.Len(t, importedBug.Snapshot().Operations, tt.numOpImp)

			// verify bugs are tagged with origin=gitlab
			issueOrigin, ok := importedBug.Snapshot().GetCreateMetadata(core.MetaKeyOrigin)
			require.True(t, ok)
			require.Equal(t, issueOrigin, target)

			//TODO: maybe more tests to ensure bug final state
		})
	}
}

func generateRepoName() string {
	rand.Seed(time.Now().UnixNano())
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, 8)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return fmt.Sprintf("%s-%s", testRepoBaseName, string(b))
}

// create repository need a token with scope 'repo'
func createRepository(ctx context.Context, name string, token *auth.Token) (int, error) {
	client, err := buildClient(defaultBaseURL, token)
	if err != nil {
		return 0, err
	}

	project, _, err := client.Projects.CreateProject(
		&gitlab.CreateProjectOptions{
			Name: gitlab.String(name),
		},
		gitlab.WithContext(ctx),
	)
	if err != nil {
		return 0, err
	}

	// fmt.Println("Project URL:", project.WebURL)

	return project.ID, nil
}

// delete repository need a token with scope 'delete_repo'
func deleteRepository(ctx context.Context, project int, token *auth.Token) error {
	client, err := buildClient(defaultBaseURL, token)
	if err != nil {
		return err
	}

	_, err = client.Projects.DeleteProject(project, gitlab.WithContext(ctx))
	return err
}
