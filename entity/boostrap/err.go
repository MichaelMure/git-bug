package bootstrap

import (
	"fmt"
	"strings"
)

// ErrNotFound is to be returned when an entity, item, element is
// not found.
type ErrNotFound struct {
	typename string
}

func NewErrNotFound(typename string) *ErrNotFound {
	return &ErrNotFound{typename: typename}
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("%s doesn't exist", e.typename)
}

func IsErrNotFound(err error) bool {
	_, ok := err.(*ErrNotFound)
	return ok
}

// ErrMultipleMatch is to be returned when more than one entity, item, element
// is found, where only one was expected.
type ErrMultipleMatch struct {
	typename string
	Matching []Id
}

func NewErrMultipleMatch(typename string, matching []Id) *ErrMultipleMatch {
	return &ErrMultipleMatch{typename: typename, Matching: matching}
}

func (e ErrMultipleMatch) Error() string {
	matching := make([]string, len(e.Matching))

	for i, match := range e.Matching {
		matching[i] = match.String()
	}

	return fmt.Sprintf("Multiple matching %s found:\n%s",
		e.typename,
		strings.Join(matching, "\n"))
}

func IsErrMultipleMatch(err error) bool {
	_, ok := err.(*ErrMultipleMatch)
	return ok
}

// ErrInvalidFormat is to be returned when reading on-disk data with an unexpected
// format or version.
type ErrInvalidFormat struct {
	version  uint
	expected uint
}

func NewErrInvalidFormat(version uint, expected uint) *ErrInvalidFormat {
	return &ErrInvalidFormat{
		version:  version,
		expected: expected,
	}
}

func NewErrUnknownFormat(expected uint) *ErrInvalidFormat {
	return &ErrInvalidFormat{
		version:  0,
		expected: expected,
	}
}

func (e ErrInvalidFormat) Error() string {
	if e.version == 0 {
		return fmt.Sprintf("unreadable data, you likely have an outdated repository format, please use https://github.com/MichaelMure/git-bug-migration to upgrade to format version %v", e.expected)
	}
	if e.version < e.expected {
		return fmt.Sprintf("outdated repository format %v, please use https://github.com/MichaelMure/git-bug-migration to upgrade to format version %v", e.version, e.expected)
	}
	return fmt.Sprintf("your version of git-bug is too old for this repository (format version %v, expected %v), please upgrade to the latest version", e.version, e.expected)
}
