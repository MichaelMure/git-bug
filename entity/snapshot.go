package entity

// Snapshot is the minimal interface that a snapshot need to implement
type Snapshot SnapshotT[Operation]

// SnapshotT is the minimal interface that a snapshot need to implement
type SnapshotT[OpT Operation] interface {
	// AllOperations returns all the operations that have been applied to that snapshot, in order
	AllOperations() []OpT
	// AppendOperation add an operation in the list
	AppendOperation(op OpT)
}

type CompileToSnapshot[OpT Operation, SnapT Snapshot] interface {
	// Compile an Entity in an easily usable snapshot
	Compile() SnapT
}
