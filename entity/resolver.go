package entity

import (
	"fmt"
	"sync"
)

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

// Resolvers is a collection of Resolver, for different type of Entity
type Resolvers map[Resolved]Resolver

// Resolve use the appropriate sub-resolver for the given type and find the Entity matching the Id.
func Resolve[T Resolved](rs Resolvers, id Id) (T, error) {
	var zero T
	for t, resolver := range rs {
		if _, ok := t.(T); ok {
			val, err := resolver.(Resolver).Resolve(id)
			if err != nil {
				return zero, err
			}
			return val.(T), nil
		}
	}
	return zero, fmt.Errorf("unknown type to resolve")
}

var _ Resolver = &CachedResolver{}

// CachedResolver is a resolver ensuring that loading is done only once through another Resolver.
type CachedResolver struct {
	resolver Resolver
	mu       sync.RWMutex
	entities map[Id]Resolved
}

func NewCachedResolver(resolver Resolver) *CachedResolver {
	return &CachedResolver{
		resolver: resolver,
		entities: make(map[Id]Resolved),
	}
}

func (c *CachedResolver) Resolve(id Id) (Resolved, error) {
	c.mu.RLock()
	if i, ok := c.entities[id]; ok {
		c.mu.RUnlock()
		return i, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	i, err := c.resolver.Resolve(id)
	if err != nil {
		return nil, err
	}
	c.entities[id] = i
	return i, nil
}

var _ Resolver = ResolverFunc[Resolved](nil)

// ResolverFunc is a helper to morph a function resolver into a Resolver
type ResolverFunc[EntityT Resolved] func(id Id) (EntityT, error)

func (fn ResolverFunc[EntityT]) Resolve(id Id) (Resolved, error) {
	return fn(id)
}

// MakeResolver create a resolver able to return the given entities.
func MakeResolver(entities ...Resolved) Resolver {
	return ResolverFunc[Resolved](func(id Id) (Resolved, error) {
		for _, entity := range entities {
			if entity.Id() == id {
				return entity, nil
			}
		}
		return nil, fmt.Errorf("entity not found")
	})
}
