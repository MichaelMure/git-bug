package identity

import (
	"encoding/json"
	"fmt"
)

// Custom unmarshaling function to allow package user to delegate
// the decoding of an Identity and distinguish between an Identity
// and a Bare.
//
// If the given message has a "id" field, it's considered being a proper Identity.
func UnmarshalJSON(raw json.RawMessage) (Interface, error) {
	aux := &IdentityStub{}

	// First try to decode and load as a normal Identity
	err := json.Unmarshal(raw, &aux)
	if err == nil && aux.Id() != "" {
		return aux, nil
	}

	// abort if we have an error other than the wrong type
	if _, ok := err.(*json.UnmarshalTypeError); err != nil && !ok {
		return nil, err
	}

	return nil, fmt.Errorf("unknown identity type")
}
