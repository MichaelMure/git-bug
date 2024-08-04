package bootstrap

type Entity interface {
	// Id return the Entity identifier
	//
	// This Id need to be immutable without having to store the entity somewhere (ie, an entity only in memory
	// should have a valid Id, and it should not change if further edit are done on this entity).
	// How to achieve that is up to the entity itself. A common way would be to take a hash of an immutable data at
	// the root of the entity.
	// It is acceptable to use such a hash and keep mutating that data as long as Id() is not called.
	Id() Id

	// Validate check if the Entity data is valid
	Validate() error
}
