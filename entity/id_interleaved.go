package entity

import (
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
)

const UnsetCombinedId = CombinedId("unset")

// CombinedId is an Id holding information from both a primary Id and a secondary Id.
// While it looks like a regular Id, do not just cast from one to another.
// Instead, use CombineIds and SeparateIds to create it and split it.
type CombinedId string

// String return the identifier as a string
func (ci CombinedId) String() string {
	return string(ci)
}

// Human return the identifier, shortened for human consumption
func (ci CombinedId) Human() string {
	format := fmt.Sprintf("%%.%ds", humanIdLength)
	return fmt.Sprintf(format, ci)
}

func (ci CombinedId) HasPrefix(prefix string) bool {
	return strings.HasPrefix(string(ci), prefix)
}

// UnmarshalGQL implement the Unmarshaler interface for gqlgen
func (ci *CombinedId) UnmarshalGQL(v interface{}) error {
	_, ok := v.(string)
	if !ok {
		return fmt.Errorf("IDs must be strings")
	}

	*ci = v.(CombinedId)

	if err := ci.Validate(); err != nil {
		return errors.Wrap(err, "invalid ID")
	}

	return nil
}

// MarshalGQL implement the Marshaler interface for gqlgen
func (ci CombinedId) MarshalGQL(w io.Writer) {
	_, _ = w.Write([]byte(`"` + ci.String() + `"`))
}

// IsValid tell if the Id is valid
func (ci CombinedId) Validate() error {
	// Special case to detect outdated repo
	if len(ci) == 40 {
		return fmt.Errorf("outdated repository format, please use https://github.com/MichaelMure/git-bug-migration to upgrade")
	}
	if len(ci) != idLength {
		return fmt.Errorf("invalid length")
	}
	for _, r := range ci {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return fmt.Errorf("invalid character")
		}
	}
	return nil
}

// CombineIds compute a merged Id holding information from both the primary Id
// and the secondary Id.
//
// This allow to later find efficiently a secondary element because we can access
// the primary one directly instead of searching for a primary that has a
// secondary matching the Id.
//
// An example usage is Comment in a Bug. The interleaved Id will hold part of the
// Bug Id and part of the Comment Id.
//
// To allow the use of an arbitrary length prefix of this Id, Ids from primary
// and secondary are interleaved with this irregular pattern to give the
// best chance to find the secondary even with a 7 character prefix.
//
// Format is: PSPSPSPPPSPPPPSPPPPSPPPPSPPPPSPPPPSPPPPSPPPPSPPPPSPPPPSPPPPSPPPP
//
// A complete interleaved Id hold 50 characters for the primary and 14 for the
// secondary, which give a key space of 36^50 for the primary (~6 * 10^77) and
// 36^14 for the secondary (~6 * 10^21). This asymmetry assume a reasonable number
// of secondary within a primary Entity, while still allowing for a vast key space
// for the primary (that is, a globally merged database) with a low risk of collision.
//
// Here is the breakdown of several common prefix length:
//
// 5:    3P, 2S
// 7:    4P, 3S
// 10:   6P, 4S
// 16:  11P, 5S
func CombineIds(primary Id, secondary Id) CombinedId {
	var id strings.Builder

	for i := 0; i < idLength; i++ {
		switch {
		default:
			id.WriteByte(primary[0])
			primary = primary[1:]
		case i == 1, i == 3, i == 5, i == 9, i >= 10 && i%5 == 4:
			id.WriteByte(secondary[0])
			secondary = secondary[1:]
		}
	}

	return CombinedId(id.String())
}

// SeparateIds extract primary and secondary prefix from an arbitrary length prefix
// of an Id created with CombineIds.
func SeparateIds(prefix string) (primaryPrefix string, secondaryPrefix string) {
	var primary strings.Builder
	var secondary strings.Builder

	for i, r := range prefix {
		switch {
		default:
			primary.WriteRune(r)
		case i == 1, i == 3, i == 5, i == 9, i >= 10 && i%5 == 4:
			secondary.WriteRune(r)
		}
	}

	return primary.String(), secondary.String()
}
