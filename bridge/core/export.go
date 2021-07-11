package core

import (
	"fmt"

	"github.com/MichaelMure/git-bug/entity"
)

type ExportEvent int

const (
	_ ExportEvent = iota

	// Bug has been exported on the remote tracker
	ExportEventBug
	// Comment has been exported on the remote tracker
	ExportEventComment
	// Comment has been edited on the remote tracker
	ExportEventCommentEdition
	// Bug's status has been changed on on the remote tracker
	ExportEventStatusChange
	// Bug's title has been changed on the remote tracker
	ExportEventTitleEdition
	// Bug's labels have been changed on the remote tracker
	ExportEventLabelChange

	// Nothing changed on the bug
	ExportEventNothing

	// Something wrong happened during export that is worth notifying to the user
	// but not severe enough to consider the export a failure.
	ExportEventWarning

	// The export system (web API) has reached a rate limit
	ExportEventRateLimiting

	// Error happened during export
	ExportEventError
)

// ExportResult is an event that is emitted during the export process, to
// allow calling code to report on what is happening, collect metrics or
// display meaningful errors if something went wrong.
type ExportResult struct {
	Err    error
	Event  ExportEvent
	ID     entity.Id
	Reason string
}

func (er ExportResult) String() string {
	switch er.Event {
	case ExportEventBug:
		return fmt.Sprintf("new issue: %s", er.ID)
	case ExportEventComment:
		return fmt.Sprintf("new comment: %s", er.ID)
	case ExportEventCommentEdition:
		return fmt.Sprintf("updated comment: %s", er.ID)
	case ExportEventStatusChange:
		return fmt.Sprintf("changed status: %s", er.ID)
	case ExportEventTitleEdition:
		return fmt.Sprintf("changed title: %s", er.ID)
	case ExportEventLabelChange:
		return fmt.Sprintf("changed label: %s", er.ID)
	case ExportEventNothing:
		if er.ID != "" {
			return fmt.Sprintf("no actions taken for event %s: %s", er.ID, er.Reason)
		}
		return fmt.Sprintf("no actions taken: %s", er.Reason)
	case ExportEventError:
		if er.ID != "" {
			return fmt.Sprintf("export error at %s: %s", er.ID, er.Err.Error())
		}
		return fmt.Sprintf("export error: %s", er.Err.Error())
	case ExportEventWarning:
		if er.ID != "" {
			return fmt.Sprintf("warning at %s: %s", er.ID, er.Err.Error())
		}
		return fmt.Sprintf("warning: %s", er.Err.Error())
	case ExportEventRateLimiting:
		return fmt.Sprintf("rate limiting: %s", er.Reason)

	default:
		panic("unknown export result")
	}
}

func NewExportError(err error, id entity.Id) ExportResult {
	return ExportResult{
		ID:    id,
		Err:   err,
		Event: ExportEventError,
	}
}

func NewExportWarning(err error, id entity.Id) ExportResult {
	return ExportResult{
		ID:    id,
		Err:   err,
		Event: ExportEventWarning,
	}
}

func NewExportNothing(id entity.Id, reason string) ExportResult {
	return ExportResult{
		ID:     id,
		Reason: reason,
		Event:  ExportEventNothing,
	}
}

func NewExportBug(id entity.Id) ExportResult {
	return ExportResult{
		ID:    id,
		Event: ExportEventBug,
	}
}

func NewExportComment(id entity.Id) ExportResult {
	return ExportResult{
		ID:    id,
		Event: ExportEventComment,
	}
}

func NewExportCommentEdition(id entity.Id) ExportResult {
	return ExportResult{
		ID:    id,
		Event: ExportEventCommentEdition,
	}
}

func NewExportStatusChange(id entity.Id) ExportResult {
	return ExportResult{
		ID:    id,
		Event: ExportEventStatusChange,
	}
}

func NewExportLabelChange(id entity.Id) ExportResult {
	return ExportResult{
		ID:    id,
		Event: ExportEventLabelChange,
	}
}

func NewExportTitleEdition(id entity.Id) ExportResult {
	return ExportResult{
		ID:    id,
		Event: ExportEventTitleEdition,
	}
}

func NewExportRateLimiting(msg string) ExportResult {
	return ExportResult{
		Reason: msg,
		Event:  ExportEventRateLimiting,
	}
}
