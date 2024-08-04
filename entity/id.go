package entity

import (
	bootstrap "github.com/MichaelMure/git-bug/entity/boostrap"
)

const HumanIdLength = bootstrap.HumanIdLength

const UnsetId = bootstrap.UnsetId

// Id is an identifier for an entity or part of an entity
type Id = bootstrap.Id

// DeriveId generate an Id from the serialization of the object or part of the object.
var DeriveId = bootstrap.DeriveId
