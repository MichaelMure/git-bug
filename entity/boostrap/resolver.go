package bootstrap

// Resolved is a minimal interface on which Resolver operates on.
// Notably, this operates on Entity and Excerpt in the cache.
type Resolved interface {
	// Id returns the object identifier.
	Id() Id
}

// Resolver is an interface to find an Entity from its Id
type Resolver interface {
	Resolve(id Id) (Resolved, error)
}
