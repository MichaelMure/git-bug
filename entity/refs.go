package entity

import (
	bootstrap "github.com/MichaelMure/git-bug/entity/boostrap"
)

// RefsToIds parse a slice of git references and return the corresponding Entity's Id.
var RefsToIds = bootstrap.RefsToIds

// RefToId parse a git reference and return the corresponding Entity's Id.
var RefToId = bootstrap.RefToId
