package entity

import (
	bootstrap "github.com/MichaelMure/git-bug/entity/boostrap"
)

// ErrNotFound is to be returned when an entity, item, element is
// not found.
type ErrNotFound = bootstrap.ErrNotFound

var NewErrNotFound = bootstrap.NewErrNotFound

func IsErrNotFound(err error) bool {
	_, ok := err.(*ErrNotFound)
	return ok
}

// ErrMultipleMatch is to be returned when more than one entity, item, element
// is found, where only one was expected.
type ErrMultipleMatch = bootstrap.ErrMultipleMatch

var NewErrMultipleMatch = bootstrap.NewErrMultipleMatch

func IsErrMultipleMatch(err error) bool {
	_, ok := err.(*ErrMultipleMatch)
	return ok
}

// ErrInvalidFormat is to be returned when reading on-disk data with an unexpected
// format or version.
type ErrInvalidFormat = bootstrap.ErrInvalidFormat

var NewErrInvalidFormat = bootstrap.NewErrInvalidFormat

var NewErrUnknownFormat = bootstrap.NewErrUnknownFormat
