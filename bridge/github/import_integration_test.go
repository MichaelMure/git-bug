package github

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/MichaelMure/git-bug/bridge/github/mocks"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
	m "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// using testify/mock and mockery

var userName = githubv4.String("marcus")
var userEmail = githubv4.String("marcus@rom.com")
var unedited = githubv4.String("unedited")
var edited = githubv4.String("edited")

func TestGithubImporterIntegration(t *testing.T) {
	// mock
	clientMock := &mocks.Client{}
	setupExpectations(t, clientMock)
	importer := githubImporter{}
	importer.client = &rateLimitHandlerClient{sc: clientMock}

	// arrange
	repo := repository.CreateGoGitTestRepo(t, false)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)
	require.NoError(t, err)

	// act
	events, err := importer.ImportAll(context.Background(), backend, time.Time{})

	// assert
	require.NoError(t, err)
	for e := range events {
		require.NoError(t, e.Err)
	}
	require.Len(t, backend.AllBugsIds(), 5)
	require.Len(t, backend.AllIdentityIds(), 2)

	b1, err := backend.ResolveBugCreateMetadata(metaKeyGithubUrl, "https://github.com/marcus/to-himself/issues/1")
	require.NoError(t, err)
	ops1 := b1.Snapshot().Operations
	require.Equal(t, "marcus", ops1[0].Author().Name())
	require.Equal(t, "title 1", ops1[0].(*bug.CreateOperation).Title)
	require.Equal(t, "body text 1", ops1[0].(*bug.CreateOperation).Message)

	b3, err := backend.ResolveBugCreateMetadata(metaKeyGithubUrl, "https://github.com/marcus/to-himself/issues/3")
	require.NoError(t, err)
	ops3 := b3.Snapshot().Operations
	require.Equal(t, "issue 3 comment 1", ops3[1].(*bug.AddCommentOperation).Message)
	require.Equal(t, "issue 3 comment 2", ops3[2].(*bug.AddCommentOperation).Message)
	require.Equal(t, []bug.Label{"bug"}, ops3[3].(*bug.LabelChangeOperation).Added)
	require.Equal(t, "title 3, edit 1", ops3[4].(*bug.SetTitleOperation).Title)

	b4, err := backend.ResolveBugCreateMetadata(metaKeyGithubUrl, "https://github.com/marcus/to-himself/issues/4")
	require.NoError(t, err)
	ops4 := b4.Snapshot().Operations
	require.Equal(t, "edited", ops4[1].(*bug.EditCommentOperation).Message)

}

func setupExpectations(t *testing.T, mock *mocks.Client) {
	rateLimitingError(mock)
	expectIssueQuery1(mock)
	expectIssueQuery2(mock)
	expectIssueQuery3(mock)
	expectUserQuery(t, mock)
}

func rateLimitingError(mock *mocks.Client) {
	mock.On("Query", m.Anything, m.AnythingOfType("*github.issueQuery"), m.Anything).Return(errors.New("API rate limit exceeded")).Once()
	mock.On("Query", m.Anything, m.AnythingOfType("*github.rateLimitQuery"), m.Anything).Return(nil).Run(
		func(args m.Arguments) {
			retVal := args.Get(1).(*rateLimitQuery)
			retVal.RateLimit.ResetAt.Time = time.Now().Add(time.Millisecond * 200)
		},
	).Once()
}

func expectIssueQuery1(mock *mocks.Client) {
	mock.On("Query", m.Anything, m.AnythingOfType("*github.issueQuery"), m.Anything).Return(nil).Run(
		func(args m.Arguments) {
			retVal := args.Get(1).(*issueQuery)
			retVal.Repository.Issues.Nodes = []issueNode{
				{
					issue: issue{
						authorEvent: authorEvent{
							Id: 1,
							Author: &actor{
								Typename: "User",
								User: userActor{
									Name:  &userName,
									Email: userEmail,
								},
							},
						},
						Title:  "title 1",
						Number: 1,
						Body:   "body text 1",
						Url: githubv4.URI{
							URL: &url.URL{
								Scheme: "https",
								Host:   "github.com",
								Path:   "marcus/to-himself/issues/1",
							},
						},
					},
					UserContentEdits: userContentEditConnection{},
					TimelineItems:    timelineItemsConnection{},
				},
				{
					issue: issue{
						authorEvent: authorEvent{
							Id: 2,
							Author: &actor{
								Typename: "User",
								User: userActor{
									Name:  &userName,
									Email: userEmail,
								},
							},
						},
						Title:  "title 2",
						Number: 2,
						Body:   "body text 2",
						Url: githubv4.URI{
							URL: &url.URL{
								Scheme: "https",
								Host:   "github.com",
								Path:   "marcus/to-himself/issues/2",
							},
						},
					},
					UserContentEdits: userContentEditConnection{},
					TimelineItems:    timelineItemsConnection{},
				},
			}
			retVal.Repository.Issues.PageInfo = pageInfo{
				EndCursor:   "end-cursor-1",
				HasNextPage: true,
			}
		},
	).Once()
}

func expectIssueQuery2(mock *mocks.Client) {
	mock.On("Query", m.Anything, m.AnythingOfType("*github.issueQuery"), m.Anything).Return(nil).Run(
		func(args m.Arguments) {
			retVal := args.Get(1).(*issueQuery)
			retVal.Repository.Issues.Nodes = []issueNode{
				{
					issue: issue{
						authorEvent: authorEvent{
							Id: 3,
							Author: &actor{
								Typename: "User",
								User: userActor{
									Name:  &userName,
									Email: userEmail,
								},
							},
						},
						Title:  "title 3",
						Number: 3,
						Body:   "body text 3",
						Url: githubv4.URI{
							URL: &url.URL{
								Scheme: "https",
								Host:   "github.com",
								Path:   "marcus/to-himself/issues/3",
							},
						},
					},
					UserContentEdits: userContentEditConnection{},
					TimelineItems: timelineItemsConnection{
						Nodes: []timelineItem{
							{
								Typename: "IssueComment",
								IssueComment: issueComment{
									authorEvent: authorEvent{
										Id: 301,
										Author: &actor{
											Typename: "User",
											User: userActor{
												Name:  &userName,
												Email: userEmail,
											},
										},
									},
									Body: "issue 3 comment 1",
									Url: githubv4.URI{
										URL: &url.URL{
											Scheme: "https",
											Host:   "github.com",
											Path:   "marcus/to-himself/issues/3#issuecomment-1",
										},
									},
									UserContentEdits: userContentEditConnection{},
								},
							},
							{
								Typename: "IssueComment",
								IssueComment: issueComment{
									authorEvent: authorEvent{
										Id: 302,
										Author: &actor{
											Typename: "User",
											User: userActor{
												Name:  &userName,
												Email: userEmail,
											},
										},
									},
									Body: "issue 3 comment 2",
									Url: githubv4.URI{
										URL: &url.URL{
											Scheme: "https",
											Host:   "github.com",
											Path:   "marcus/to-himself/issues/3#issuecomment-2",
										},
									},
									UserContentEdits: userContentEditConnection{},
								},
							},
							{
								Typename: "LabeledEvent",
								LabeledEvent: labeledEvent{
									actorEvent: actorEvent{
										Id: 303,
										Actor: &actor{
											Typename: "User",
											User: userActor{
												Name:  &userName,
												Email: userEmail,
											},
										},
									},
									Label: label{
										Name: "bug",
									},
								},
							},
							{
								Typename: "RenamedTitleEvent",
								RenamedTitleEvent: renamedTitleEvent{
									actorEvent: actorEvent{
										Id: 304,
										Actor: &actor{
											Typename: "User",
											User: userActor{
												Name:  &userName,
												Email: userEmail,
											},
										},
									},
									CurrentTitle: "title 3, edit 1",
								},
							},
						},
						PageInfo: pageInfo{},
					},
				},
				{
					issue: issue{
						authorEvent: authorEvent{
							Id: 4,
							Author: &actor{
								Typename: "User",
								User: userActor{
									Name:  &userName,
									Email: userEmail,
								},
							},
						},
						Title:  "title 4",
						Number: 4,
						Body:   unedited,
						Url: githubv4.URI{
							URL: &url.URL{
								Scheme: "https",
								Host:   "github.com",
								Path:   "marcus/to-himself/issues/4",
							},
						},
					},
					UserContentEdits: userContentEditConnection{
						Nodes: []userContentEdit{
							// Github is weird: here the order is reversed chronological
							{
								Id: 402,
								Editor: &actor{
									Typename: "User",
									User: userActor{
										Name:  &userName,
										Email: userEmail,
									},
								},
								Diff: &edited,
							},
							{
								Id: 401,
								Editor: &actor{
									Typename: "User",
									User: userActor{
										Name:  &userName,
										Email: userEmail,
									},
								},
								// Github is weird: whenever an issue has issue edits, then the first item
								// (issue edit) holds the original (unedited) content and the second item
								// (issue edit) holds the (first) edited content.
								Diff: &unedited,
							},
						},
						PageInfo: pageInfo{},
					},
					TimelineItems: timelineItemsConnection{},
				},
			}
			retVal.Repository.Issues.PageInfo = pageInfo{
				EndCursor:   "end-cursor-2",
				HasNextPage: true,
			}
		},
	).Once()
}

func expectIssueQuery3(mock *mocks.Client) {
	mock.On("Query", m.Anything, m.AnythingOfType("*github.issueQuery"), m.Anything).Return(nil).Run(
		func(args m.Arguments) {
			retVal := args.Get(1).(*issueQuery)
			retVal.Repository.Issues.Nodes = []issueNode{
				{
					issue: issue{
						authorEvent: authorEvent{
							Author: nil,
						},
						Title:  "title 5",
						Number: 5,
						Body:   "body text 5",
						Url: githubv4.URI{
							URL: &url.URL{
								Scheme: "https",
								Host:   "github.com",
								Path:   "marcus/to-himself/issues/5",
							},
						},
					},
					UserContentEdits: userContentEditConnection{},
					TimelineItems:    timelineItemsConnection{},
				},
			}
			retVal.Repository.Issues.PageInfo = pageInfo{}
		},
	).Once()
}

func expectUserQuery(t *testing.T, mock *mocks.Client) {
	mock.On("Query", m.Anything, m.AnythingOfType("*github.userQuery"), m.AnythingOfType("map[string]interface {}")).Return(nil).Run(
		func(args m.Arguments) {
			vars := args.Get(2).(map[string]interface{})
			ghost := githubv4.String("ghost")
			require.Equal(t, ghost, vars["login"])

			retVal := args.Get(1).(*userQuery)
			retVal.User.Name = &ghost
			retVal.User.Login = "ghost-login"
		},
	).Once()
}
