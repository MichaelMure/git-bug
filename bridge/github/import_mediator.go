package github

import (
	"context"
	"time"

	"github.com/shurcooL/githubv4"
)

const (
	// These values influence how fast the github graphql rate limit is exhausted.
	NumIssues        = 40
	NumIssueEdits    = 100
	NumTimelineItems = 100
	NumCommentEdits  = 100

	ChanCapacity = 128
)

// importMediator provides a convenient interface to retrieve issues from the Github GraphQL API.
type importMediator struct {
	// Github graphql client
	gh *client

	// name of the repository owner on Github
	owner string

	// name of the Github repository
	project string

	// since specifies which issues to import. Issues that have been updated at or after the
	// given date should be imported.
	since time.Time

	// importEvents holds events representing issues, comments, edits, ...
	// In this channel issues are immediately followed by their issue edits and comments are
	// immediately followed by their comment edits.
	importEvents chan ImportEvent

	// Sticky error
	err error
}

type ImportEvent interface {
	isImportEvent()
}

func (RateLimitingEvent) isImportEvent() {}

type IssueEvent struct {
	issue
}

func (IssueEvent) isImportEvent() {}

type IssueEditEvent struct {
	issueId githubv4.ID
	userContentEdit
}

func (IssueEditEvent) isImportEvent() {}

type TimelineEvent struct {
	issueId githubv4.ID
	timelineItem
}

func (TimelineEvent) isImportEvent() {}

type CommentEditEvent struct {
	commentId githubv4.ID
	userContentEdit
}

func (CommentEditEvent) isImportEvent() {}

func (mm *importMediator) NextImportEvent() ImportEvent {
	return <-mm.importEvents
}

func NewImportMediator(ctx context.Context, client *client, owner, project string, since time.Time) *importMediator {
	mm := importMediator{
		gh:           client,
		owner:        owner,
		project:      project,
		since:        since,
		importEvents: make(chan ImportEvent, ChanCapacity),
		err:          nil,
	}
	go func() {
		mm.fillImportEvents(ctx)
		close(mm.importEvents)
	}()
	return &mm
}

type varmap map[string]interface{}

func newIssueVars(owner, project string, since time.Time) varmap {
	return varmap{
		"owner":             githubv4.String(owner),
		"name":              githubv4.String(project),
		"issueSince":        githubv4.DateTime{Time: since},
		"issueFirst":        githubv4.Int(NumIssues),
		"issueEditLast":     githubv4.Int(NumIssueEdits),
		"issueEditBefore":   (*githubv4.String)(nil),
		"timelineFirst":     githubv4.Int(NumTimelineItems),
		"timelineAfter":     (*githubv4.String)(nil),
		"commentEditLast":   githubv4.Int(NumCommentEdits),
		"commentEditBefore": (*githubv4.String)(nil),
	}
}

func newIssueEditVars() varmap {
	return varmap{
		"issueEditLast": githubv4.Int(NumIssueEdits),
	}
}

func newTimelineVars() varmap {
	return varmap{
		"timelineFirst":     githubv4.Int(NumTimelineItems),
		"commentEditLast":   githubv4.Int(NumCommentEdits),
		"commentEditBefore": (*githubv4.String)(nil),
	}
}

func newCommentEditVars() varmap {
	return varmap{
		"commentEditLast": githubv4.Int(NumCommentEdits),
	}
}

func (mm *importMediator) Error() error {
	return mm.err
}

func (mm *importMediator) User(ctx context.Context, loginName string) (*user, error) {
	query := userQuery{}
	vars := varmap{"login": githubv4.String(loginName)}
	if err := mm.gh.queryWithImportEvents(ctx, &query, vars, mm.importEvents); err != nil {
		return nil, err
	}
	return &query.User, nil
}

func (mm *importMediator) fillImportEvents(ctx context.Context) {
	initialCursor := githubv4.String("")
	issues, hasIssues := mm.queryIssue(ctx, initialCursor)
	for hasIssues {
		for _, node := range issues.Nodes {
			select {
			case <-ctx.Done():
				return
			case mm.importEvents <- IssueEvent{node.issue}:
			}

			// issue edit events follow the issue event
			mm.fillIssueEditEvents(ctx, &node)
			// last come the timeline events
			mm.fillTimelineEvents(ctx, &node)
		}
		if !issues.PageInfo.HasNextPage {
			break
		}
		issues, hasIssues = mm.queryIssue(ctx, issues.PageInfo.EndCursor)
	}
}

func (mm *importMediator) fillIssueEditEvents(ctx context.Context, issueNode *issueNode) {
	edits := &issueNode.UserContentEdits
	hasEdits := true
	for hasEdits {
		for edit := range reverse(edits.Nodes) {
			if edit.Diff == nil || string(*edit.Diff) == "" {
				// issueEdit.Diff == nil happen if the event is older than early
				// 2018, Github doesn't have the data before that. Best we can do is
				// to ignore the event.
				continue
			}
			select {
			case <-ctx.Done():
				return
			case mm.importEvents <- IssueEditEvent{issueId: issueNode.issue.Id, userContentEdit: edit}:
			}
		}
		if !edits.PageInfo.HasPreviousPage {
			break
		}
		edits, hasEdits = mm.queryIssueEdits(ctx, issueNode.issue.Id, edits.PageInfo.EndCursor)
	}
}

func (mm *importMediator) queryIssueEdits(ctx context.Context, nid githubv4.ID, cursor githubv4.String) (*userContentEditConnection, bool) {
	vars := newIssueEditVars()
	vars["gqlNodeId"] = nid
	if cursor == "" {
		vars["issueEditBefore"] = (*githubv4.String)(nil)
	} else {
		vars["issueEditBefore"] = cursor
	}
	query := issueEditQuery{}
	if err := mm.gh.queryWithImportEvents(ctx, &query, vars, mm.importEvents); err != nil {
		mm.err = err
		return nil, false
	}
	connection := &query.Node.Issue.UserContentEdits
	if len(connection.Nodes) <= 0 {
		return nil, false
	}
	return connection, true
}

func (mm *importMediator) fillTimelineEvents(ctx context.Context, issueNode *issueNode) {
	items := &issueNode.TimelineItems
	hasItems := true
	for hasItems {
		for _, item := range items.Nodes {
			select {
			case <-ctx.Done():
				return
			case mm.importEvents <- TimelineEvent{issueId: issueNode.issue.Id, timelineItem: item}:
			}
			if item.Typename == "IssueComment" {
				// Issue comments are different than other timeline items in that
				// they may have associated user content edits.
				// Right after the comment we send the comment edits.
				mm.fillCommentEdits(ctx, &item)
			}
		}
		if !items.PageInfo.HasNextPage {
			break
		}
		items, hasItems = mm.queryTimeline(ctx, issueNode.issue.Id, items.PageInfo.EndCursor)
	}
}

func (mm *importMediator) queryTimeline(ctx context.Context, nid githubv4.ID, cursor githubv4.String) (*timelineItemsConnection, bool) {
	vars := newTimelineVars()
	vars["gqlNodeId"] = nid
	if cursor == "" {
		vars["timelineAfter"] = (*githubv4.String)(nil)
	} else {
		vars["timelineAfter"] = cursor
	}
	query := timelineQuery{}
	if err := mm.gh.queryWithImportEvents(ctx, &query, vars, mm.importEvents); err != nil {
		mm.err = err
		return nil, false
	}
	connection := &query.Node.Issue.TimelineItems
	if len(connection.Nodes) <= 0 {
		return nil, false
	}
	return connection, true
}

func (mm *importMediator) fillCommentEdits(ctx context.Context, item *timelineItem) {
	// Here we are only concerned with timeline items of type issueComment.
	if item.Typename != "IssueComment" {
		return
	}
	// First: setup message handling while submitting GraphQL queries.
	comment := &item.IssueComment
	edits := &comment.UserContentEdits
	hasEdits := true
	for hasEdits {
		for edit := range reverse(edits.Nodes) {
			if edit.Diff == nil || string(*edit.Diff) == "" {
				// issueEdit.Diff == nil happen if the event is older than early
				// 2018, Github doesn't have the data before that. Best we can do is
				// to ignore the event.
				continue
			}
			select {
			case <-ctx.Done():
				return
			case mm.importEvents <- CommentEditEvent{commentId: comment.Id, userContentEdit: edit}:
			}
		}
		if !edits.PageInfo.HasPreviousPage {
			break
		}
		edits, hasEdits = mm.queryCommentEdits(ctx, comment.Id, edits.PageInfo.EndCursor)
	}
}

func (mm *importMediator) queryCommentEdits(ctx context.Context, nid githubv4.ID, cursor githubv4.String) (*userContentEditConnection, bool) {
	vars := newCommentEditVars()
	vars["gqlNodeId"] = nid
	if cursor == "" {
		vars["commentEditBefore"] = (*githubv4.String)(nil)
	} else {
		vars["commentEditBefore"] = cursor
	}
	query := commentEditQuery{}
	if err := mm.gh.queryWithImportEvents(ctx, &query, vars, mm.importEvents); err != nil {
		mm.err = err
		return nil, false
	}
	connection := &query.Node.IssueComment.UserContentEdits
	if len(connection.Nodes) <= 0 {
		return nil, false
	}
	return connection, true
}

func (mm *importMediator) queryIssue(ctx context.Context, cursor githubv4.String) (*issueConnection, bool) {
	vars := newIssueVars(mm.owner, mm.project, mm.since)
	if cursor == "" {
		vars["issueAfter"] = (*githubv4.String)(nil)
	} else {
		vars["issueAfter"] = cursor
	}
	query := issueQuery{}
	if err := mm.gh.queryWithImportEvents(ctx, &query, vars, mm.importEvents); err != nil {
		mm.err = err
		return nil, false
	}
	connection := &query.Repository.Issues
	if len(connection.Nodes) <= 0 {
		return nil, false
	}
	return connection, true
}

func reverse(eds []userContentEdit) chan userContentEdit {
	ret := make(chan userContentEdit)
	go func() {
		for i := range eds {
			ret <- eds[len(eds)-1-i]
		}
		close(ret)
	}()
	return ret
}
