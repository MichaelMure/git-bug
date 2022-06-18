package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

const (
	testRepoBaseName = "git-bug-test-github-exporter"
)

type testCase struct {
	name    string
	bug     *cache.BugCache
	numOrOp int // number of original operations
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

	_, _, err = bugLabelChange.ChangeLabels([]string{"InVaLiD"}, nil)
	require.NoError(t, err)

	_, _, err = bugLabelChange.ChangeLabels([]string{"bUG"}, nil)
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
		{
			name:    "simple bug",
			bug:     simpleBug,
			numOrOp: 1,
		},
		{
			name:    "bug with comments",
			bug:     bugWithComments,
			numOrOp: 2,
		},
		{
			name:    "bug label change",
			bug:     bugLabelChange,
			numOrOp: 6,
		},
		{
			name:    "bug with comment editions",
			bug:     bugWithCommentEditions,
			numOrOp: 4,
		},
		{
			name:    "bug changed status",
			bug:     bugStatusChanged,
			numOrOp: 3,
		},
		{
			name:    "bug title edited",
			bug:     bugTitleEdited,
			numOrOp: 2,
		},
	}
}

func TestGithubPushPull(t *testing.T) {
	// repo owner
	envUser := os.Getenv("GITHUB_TEST_USER")

	// token must have 'repo' and 'delete_repo' scopes
	envToken := os.Getenv("GITHUB_TOKEN_ADMIN")
	if envToken == "" {
		t.Skip("Env var GITHUB_TOKEN_ADMIN missing")
	}

	// create repo backend
	repo := repository.CreateGoGitTestRepo(t, false)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	// set author identity
	login := "identity-test"
	author, err := backend.NewIdentity("test identity", "test@test.org")
	require.NoError(t, err)
	author.SetMetadata(metaKeyGithubLogin, login)
	err = author.Commit()
	require.NoError(t, err)

	err = backend.SetUserIdentity(author)
	require.NoError(t, err)

	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	token := auth.NewToken(target, envToken)
	token.SetMetadata(auth.MetaKeyLogin, login)
	err = auth.Store(repo, token)
	require.NoError(t, err)

	tests := testCases(t, backend)

	// generate project name
	projectName := generateRepoName()

	// create target Github repository
	err = createRepository(projectName, envToken)
	require.NoError(t, err)

	fmt.Println("created repository", projectName)

	// Let Github handle the repo creation and update all their internal caches.
	// Avoid HTTP error 404 retrieving repository node id
	time.Sleep(10 * time.Second)

	// Make sure to remove the Github repository when the test end
	defer func(t *testing.T) {
		if err := deleteRepository(projectName, envUser, envToken); err != nil {
			t.Fatal(err)
		}
		fmt.Println("deleted repository:", projectName)
	}(t)

	interrupt.RegisterCleaner(func() error {
		return deleteRepository(projectName, envUser, envToken)
	})

	ctx := context.Background()

	// initialize exporter
	exporter := &githubExporter{}
	err = exporter.Init(ctx, backend, core.Configuration{
		confKeyOwner:        envUser,
		confKeyProject:      projectName,
		confKeyDefaultLogin: login,
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

	importer := &githubImporter{}
	err = importer.Init(ctx, backend, core.Configuration{
		confKeyOwner:        envUser,
		confKeyProject:      projectName,
		confKeyDefaultLogin: login,
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
			require.Len(t, tt.bug.Snapshot().Operations, tt.numOrOp*2)

			// verify operation have correct metadata
			for _, op := range tt.bug.Snapshot().Operations {
				// Check if the originals operations (*not* SetMetadata) are tagged properly
				if _, ok := op.(*bug.SetMetadataOperation); !ok {
					_, haveIDMetadata := op.GetMetadata(metaKeyGithubId)
					require.True(t, haveIDMetadata)

					_, haveURLMetada := op.GetMetadata(metaKeyGithubUrl)
					require.True(t, haveURLMetada)
				}
			}

			// get bug github ID
			bugGithubID, ok := tt.bug.Snapshot().GetCreateMetadata(metaKeyGithubId)
			require.True(t, ok)

			// retrieve bug from backendTwo
			importedBug, err := backendTwo.ResolveBugCreateMetadata(metaKeyGithubId, bugGithubID)
			require.NoError(t, err)

			// verify bug have same number of original operations
			require.Len(t, importedBug.Snapshot().Operations, tt.numOrOp)

			// verify bugs are tagged with origin=github
			issueOrigin, ok := importedBug.Snapshot().GetCreateMetadata(core.MetaKeyOrigin)
			require.True(t, ok)
			require.Equal(t, issueOrigin, target)

			// TODO: maybe more tests to ensure bug final state
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
func createRepository(project, token string) error {
	// This function use the V3 Github API because repository creation is not supported yet on the V4 API.
	url := fmt.Sprintf("%s/user/repos", githubV3Url)

	params := struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Private     bool   `json:"private"`
		HasIssues   bool   `json:"has_issues"`
	}{
		Name:        project,
		Description: "git-bug exporter temporary test repository",
		Private:     true,
		HasIssues:   true,
	}

	data, err := json.Marshal(params)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	// need the token for private repositories
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	client := &http.Client{
		Timeout: defaultTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// delete repository need a token with scope 'delete_repo'
func deleteRepository(project, owner, token string) error {
	// This function use the V3 Github API because repository removal is not supported yet on the V4 API.
	url := fmt.Sprintf("%s/repos/%s/%s", githubV3Url, owner, project)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	// need the token for private repositories
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	client := &http.Client{
		Timeout: defaultTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("error deleting repository")
	}

	return nil
}
