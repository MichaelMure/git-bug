package dag

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
)

type Snapshot entity.SnapshotT[Operation]

// Operation is an extended interface for an entity.Operation working with the dag package.
type Operation interface {
	entity.Operation

	// setId allow to set the Id, used when unmarshalling only
	setId(id entity.Id)
	// setAuthor allow to set the author, used when unmarshalling only
	setAuthor(author identity.Interface)
	// setExtraMetadataImmutable add a metadata not carried by the operation itself on the operation
	setExtraMetadataImmutable(key string, value string)
}

type OperationWithApply[SnapT Snapshot] interface {
	Operation

	// Apply the operation to a Snapshot to create the final state
	Apply(snapshot SnapT)
}

// OpBase implement the common feature that every Operation should support.
type OpBase struct {
	// Not serialized. Store the op's id in memory.
	id entity.Id
	// Not serialized
	author identity.Interface

	OperationType entity.OperationType `json:"type"`
	UnixTime      int64                `json:"timestamp"`

	// mandatory random bytes to ensure a better randomness of the data used to later generate the ID
	// len(Nonce) should be > 20 and < 64 bytes
	// It has no functional purpose and should be ignored.
	Nonce []byte `json:"nonce"`

	Metadata map[string]string `json:"metadata,omitempty"`
	// Not serialized. Store the extra metadata in memory,
	// compiled from SetMetadataOperation.
	extraMetadata map[string]string
}

func NewOpBase(opType entity.OperationType, author identity.Interface, unixTime int64) OpBase {
	return OpBase{
		OperationType: opType,
		author:        author,
		UnixTime:      unixTime,
		Nonce:         makeNonce(20),
		id:            entity.UnsetId,
	}
}

func makeNonce(len int) []byte {
	result := make([]byte, len)
	_, err := rand.Read(result)
	if err != nil {
		panic(err)
	}
	return result
}

func IdOperation(op Operation, base *OpBase) entity.Id {
	if base.id == "" {
		// something went really wrong
		panic("op's id not set")
	}
	if base.id == entity.UnsetId {
		// This means we are trying to get the op's Id *before* it has been stored, for instance when
		// adding multiple ops in one go in an OperationPack.
		// As the Id is computed based on the actual bytes written on the disk, we are going to predict
		// those and then get the Id. This is safe as it will be the exact same code writing on disk later.

		data, err := json.Marshal(op)
		if err != nil {
			panic(err)
		}

		base.id = entity.DeriveId(data)
	}
	return base.id
}

func (base *OpBase) Type() entity.OperationType {
	return base.OperationType
}

// Time return the time when the operation was added
func (base *OpBase) Time() time.Time {
	return time.Unix(base.UnixTime, 0)
}

// Validate check the OpBase for errors
func (base *OpBase) Validate(op entity.Operation, opType entity.OperationType) error {
	if base.OperationType == 0 {
		return fmt.Errorf("operation type unset")
	}
	if base.OperationType != opType {
		return fmt.Errorf("incorrect operation type (expected: %v, actual: %v)", opType, base.OperationType)
	}

	if op.Time().Unix() == 0 {
		return fmt.Errorf("time not set")
	}

	if base.author == nil {
		return fmt.Errorf("author not set")
	}

	if err := op.Author().Validate(); err != nil {
		return errors.Wrap(err, "author")
	}

	if op, ok := op.(entity.OperationWithFiles); ok {
		for _, hash := range op.GetFiles() {
			if !hash.IsValid() {
				return fmt.Errorf("file with invalid hash %v", hash)
			}
		}
	}

	if len(base.Nonce) > 64 {
		return fmt.Errorf("nonce is too big")
	}
	if len(base.Nonce) < 20 {
		return fmt.Errorf("nonce is too small")
	}

	return nil
}

// IsAuthored is a sign post method for gqlgen
func (base *OpBase) IsAuthored() {}

// Author return author identity
func (base *OpBase) Author() identity.Interface {
	return base.author
}

// IdIsSet returns true if the id has been set already
func (base *OpBase) IdIsSet() bool {
	return base.id != "" && base.id != entity.UnsetId
}

// SetMetadata store arbitrary metadata about the operation
func (base *OpBase) SetMetadata(key string, value string) {
	if base.IdIsSet() {
		panic("set metadata on an operation with already an Id")
	}

	if base.Metadata == nil {
		base.Metadata = make(map[string]string)
	}
	base.Metadata[key] = value
}

// GetMetadata retrieve arbitrary metadata about the operation
func (base *OpBase) GetMetadata(key string) (string, bool) {
	val, ok := base.Metadata[key]

	if ok {
		return val, true
	}

	// extraMetadata can't replace the original operations value if any
	val, ok = base.extraMetadata[key]

	return val, ok
}

// AllMetadata return all metadata for this operation
func (base *OpBase) AllMetadata() map[string]string {
	result := make(map[string]string)

	for key, val := range base.extraMetadata {
		result[key] = val
	}

	// Original metadata take precedence
	for key, val := range base.Metadata {
		result[key] = val
	}

	return result
}

// setId allow to set the Id, used when unmarshalling only
func (base *OpBase) setId(id entity.Id) {
	if base.id != "" && base.id != entity.UnsetId {
		panic("trying to set id again")
	}
	base.id = id
}

// setAuthor allow to set the author, used when unmarshalling only
func (base *OpBase) setAuthor(author identity.Interface) {
	base.author = author
}

func (base *OpBase) setExtraMetadataImmutable(key string, value string) {
	if base.extraMetadata == nil {
		base.extraMetadata = make(map[string]string)
	}
	if _, exist := base.extraMetadata[key]; !exist {
		base.extraMetadata[key] = value
	}
}
