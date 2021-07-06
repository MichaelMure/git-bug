package entity

import (
	"crypto/sha256"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
)

// sha-256
const idLength = 64
const humanIdLength = 7

const UnsetId = Id("unset")

// Id is an identifier for an entity or part of an entity
type Id string

// DeriveId generate an Id from the serialization of the object or part of the object.
func DeriveId(data []byte) Id {
	// My understanding is that sha256 is enough to prevent collision (git use that, so ...?)
	// If you read this code, I'd be happy to be schooled.

	sum := sha256.Sum256(data)
	return Id(fmt.Sprintf("%x", sum))
}

// String return the identifier as a string
func (i Id) String() string {
	return string(i)
}

// Human return the identifier, shortened for human consumption
func (i Id) Human() string {
	format := fmt.Sprintf("%%.%ds", humanIdLength)
	return fmt.Sprintf(format, i)
}

func (i Id) HasPrefix(prefix string) bool {
	return strings.HasPrefix(string(i), prefix)
}

// UnmarshalGQL implement the Unmarshaler interface for gqlgen
func (i *Id) UnmarshalGQL(v interface{}) error {
	_, ok := v.(string)
	if !ok {
		return fmt.Errorf("IDs must be strings")
	}

	*i = v.(Id)

	if err := i.Validate(); err != nil {
		return errors.Wrap(err, "invalid ID")
	}

	return nil
}

// MarshalGQL implement the Marshaler interface for gqlgen
func (i Id) MarshalGQL(w io.Writer) {
	_, _ = w.Write([]byte(`"` + i.String() + `"`))
}

// IsValid tell if the Id is valid
func (i Id) Validate() error {
	// Special case to detect outdated repo
	if len(i) == 40 {
		return fmt.Errorf("outdated repository format, please use https://github.com/MichaelMure/git-bug-migration to upgrade")
	}
	if len(i) != idLength {
		return fmt.Errorf("invalid length")
	}
	for _, r := range i {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return fmt.Errorf("invalid character")
		}
	}
	return nil
}
