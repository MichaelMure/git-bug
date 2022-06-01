package github

import (
	"context"
	"fmt"
	"time"

	"github.com/shurcooL/githubv4"
)

const (
	// These values influence how fast the github graphql rate limit is exhausted.
	NumIssues        = 100
	NumIssueEdits    = 50
	NumTimelineItems = 50
	NumCommentEdits  = 50

	ChanCapacity = 128
)

// importMediator provides a convenient interface to retrieve issues from the Github GraphQL API.
type importMediator struct {
	// Github graphql client
	gh *rateLimitHandlerClient

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

type ErrorEvent struct {
	issueId githubv4.ID
	err     error
}

func (ErrorEvent) isImportEvent() {}

func (mm *importMediator) NextImportEvent() ImportEvent {
	return <-mm.importEvents
}

func NewImportMediator(ctx context.Context, client *rateLimitHandlerClient, owner, project string, since time.Time) *importMediator {
	mm := importMediator{
		gh:           client,
		owner:        owner,
		project:      project,
		since:        since,
		importEvents: make(chan ImportEvent, ChanCapacity),
		err:          nil,
	}

	// 1. Prepare import state
	state := newImportState()
	state.tryLoadFromFile()
	if !state.isLoaded() {
		issues := mm.getIssueIds(ctx)
		state.addNewIssues(issues)
		state.writeToFileSystem()
	}

	// 2. Process import state
	go func() {
		mm.generateImportEvents(ctx, state)
		close(mm.importEvents)
		state.writeToFileSystem()
	}()
	return &mm
}

type varmap map[string]interface{}

func newIssueIdsVars(owner, project string, since time.Time) varmap {
	return varmap{
		"owner":      githubv4.String(owner),
		"name":       githubv4.String(project),
		"issueSince": githubv4.DateTime{Time: since},
		"issueFirst": githubv4.Int(NumIssues),
	}
}

func newIssueVars(owner, project string, issueId githubv4.Int) varmap {
	return varmap{
		"owner":             githubv4.String(owner),
		"name":              githubv4.String(project),
		"issueNumber":       issueId,
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

func (mm *importMediator) getIssueIds(ctx context.Context) []githubv4.Int {
	result := make([]githubv4.Int, 0)
	initialCursor := githubv4.String("")
	issueConnection, err := mm.queryIssueIds(ctx, initialCursor)
	if err != nil {
		mm.err = err
	}
	for issueConnection != nil {
		for _, issueNode := range issueConnection.Nodes {
			result = append(result, issueNode.Number)
		}
		if !issueConnection.PageInfo.HasNextPage {
			break
		}
		issueConnection, err = mm.queryIssueIds(ctx, issueConnection.PageInfo.EndCursor)
		if err != nil {
			mm.err = err
		}
	}
	return result
}

func (mm *importMediator) generateImportEvents(ctx context.Context, importState importState) {
	for _, issueId := range importState.issuesToImport() {
		node, err := mm.queryIssue(ctx, issueId)
		if err != nil {
			// TODO(as)
			importState.setImportError(issueId)
			continue
		}
		select {
		case <-ctx.Done():
			return
		case mm.importEvents <- IssueEvent{node.issue}:
		}

		var err1, err2 error
		// issue edit events follow the issue event
		err1 = mm.generateIssueEditEvents(ctx, node)
		// last come the timeline events
		err2 = mm.generateTimelineEvents(ctx, node)

		if err1 == nil && err2 == nil {
			importState.setImportSuccess(issueId)
		} else {
			importState.setImportError(issueId)
		}
	}
}

func (mm *importMediator) generateIssueEditEvents(ctx context.Context, issueNode *issueNode) error {
	edits := &issueNode.UserContentEdits
	for edits != nil {
		var err error
		for edit := range reverse(edits.Nodes) {
			if edit.Diff == nil || string(*edit.Diff) == "" {
				// issueEdit.Diff == nil happen if the event is older than early
				// 2018, Github doesn't have the data before that. Best we can do is
				// to ignore the event.
				continue
			}
			select {
			case <-ctx.Done():
				mm.err = fmt.Errorf("canceled")
				return mm.err
			case mm.importEvents <- IssueEditEvent{issueId: issueNode.issue.Id, userContentEdit: edit}:
			}
		}
		if !edits.PageInfo.HasPreviousPage {
			break
		}
		edits, err = mm.queryIssueEdits(ctx, issueNode.issue.Id, edits.PageInfo.EndCursor)
		if err != nil {
			mm.err = err
			return err
		}
	}
	return nil
}

func (mm *importMediator) queryIssueEdits(ctx context.Context, nid githubv4.ID, cursor githubv4.String) (*userContentEditConnection, error) {
	vars := newIssueEditVars()
	vars["gqlNodeId"] = nid
	if cursor == "" {
		vars["issueEditBefore"] = (*githubv4.String)(nil)
	} else {
		vars["issueEditBefore"] = cursor
	}
	query := issueEditQuery{}
	if err := mm.gh.queryWithImportEvents(ctx, &query, vars, mm.importEvents); err != nil {
		// TODO(as): remove that line?
		mm.err = err
		return nil, err
	}
	connection := &query.Node.Issue.UserContentEdits
	if len(connection.Nodes) <= 0 {
		return nil, nil
	}
	return connection, nil
}

func (mm *importMediator) generateTimelineEvents(ctx context.Context, issueNode *issueNode) error {
	items := &issueNode.TimelineItems
	for items != nil {
		var err error
		for _, item := range items.Nodes {
			select {
			case <-ctx.Done():
				mm.err = fmt.Errorf("canceled")
				return mm.err
			case mm.importEvents <- TimelineEvent{issueId: issueNode.issue.Id, timelineItem: item}:
			}
			if item.Typename == "IssueComment" {
				// Issue comments are different than other timeline items in that
				// they may have associated user content edits.
				// Right after the comment we send the comment edits.
				if err := mm.fillCommentEdits(ctx, &item); err != nil {
					mm.err = err
					return err
				}
			}
		}
		if !items.PageInfo.HasNextPage {
			break
		}
		items, err = mm.queryTimeline(ctx, issueNode.issue.Id, items.PageInfo.EndCursor)
		if err != nil {
			mm.err = err
			return err
		}
	}
	return nil
}

func (mm *importMediator) queryTimeline(ctx context.Context, nid githubv4.ID, cursor githubv4.String) (*timelineItemsConnection, error) {
	vars := newTimelineVars()
	vars["gqlNodeId"] = nid
	if cursor == "" {
		vars["timelineAfter"] = (*githubv4.String)(nil)
	} else {
		vars["timelineAfter"] = cursor
	}
	query := timelineQuery{}
	if err := mm.gh.queryWithImportEvents(ctx, &query, vars, mm.importEvents); err != nil {
		return nil, err
	}
	connection := &query.Node.Issue.TimelineItems
	if len(connection.Nodes) <= 0 {
		return nil, nil
	}
	return connection, nil
}

func (mm *importMediator) fillCommentEdits(ctx context.Context, item *timelineItem) error {
	// Here we are only concerned with timeline items of type issueComment.
	if item.Typename != "IssueComment" {
		return nil
	}
	var err error
	// First: setup message handling while submitting GraphQL queries.
	comment := &item.IssueComment
	edits := &comment.UserContentEdits
	for edits != nil {
		for edit := range reverse(edits.Nodes) {
			if edit.Diff == nil || string(*edit.Diff) == "" {
				// issueEdit.Diff == nil happen if the event is older than early
				// 2018, Github doesn't have the data before that. Best we can do is
				// to ignore the event.
				continue
			}
			select {
			case <-ctx.Done():
				return fmt.Errorf("canceled")
			case mm.importEvents <- CommentEditEvent{commentId: comment.Id, userContentEdit: edit}:
			}
		}
		if !edits.PageInfo.HasPreviousPage {
			break
		}
		edits, err = mm.queryCommentEdits(ctx, comment.Id, edits.PageInfo.EndCursor)
		if err != nil {
			return err
		}
	}
	return nil
}

func (mm *importMediator) queryCommentEdits(ctx context.Context, nid githubv4.ID, cursor githubv4.String) (*userContentEditConnection, error) {
	vars := newCommentEditVars()
	vars["gqlNodeId"] = nid
	if cursor == "" {
		vars["commentEditBefore"] = (*githubv4.String)(nil)
	} else {
		vars["commentEditBefore"] = cursor
	}
	query := commentEditQuery{}
	if err := mm.gh.queryWithImportEvents(ctx, &query, vars, mm.importEvents); err != nil {
		return nil, err
	}
	connection := &query.Node.IssueComment.UserContentEdits
	if len(connection.Nodes) <= 0 {
		return nil, nil
	}
	return connection, nil
}

func (mm *importMediator) queryIssueIds(ctx context.Context, cursor githubv4.String) (*issueIdsConnection, error) {
	vars := newIssueIdsVars(mm.owner, mm.project, mm.since)
	if cursor == "" {
		vars["issueAfter"] = (*githubv4.String)(nil)
	} else {
		vars["issueAfter"] = cursor
	}
	query := issueIdsQuery{}
	if err := mm.gh.queryWithImportEvents(ctx, &query, vars, mm.importEvents); err != nil {
		return nil, err
	}
	issueConnection := &query.Repository.Issues
	if len(issueConnection.Nodes) <= 0 {
		return nil, nil
	}
	return issueConnection, nil
}

func (mm *importMediator) queryIssue(ctx context.Context, issueId githubv4.Int) (*issueNode, error) {
	// TODO(as)
	vars := newIssueVars(mm.owner, mm.project, issueId)
	query := issueQuery{}
	if err := mm.gh.queryWithImportEvents(ctx, &query, vars, mm.importEvents); err != nil {
		mm.err = err
		return nil, err
	}
	return &query.Repository.Issue, nil
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
