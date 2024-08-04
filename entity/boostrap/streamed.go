package bootstrap

// TODO: type alias not possible on generics for now
// https://github.com/golang/go/issues/46477

type StreamedEntity[EntityT Entity] struct {
	Err    error
	Entity EntityT

	// CurrentEntity is the index of the current entity being streamed, to express progress.
	CurrentEntity int64
	// TotalEntities is the total count of expected entities, if known.
	TotalEntities int64
}
