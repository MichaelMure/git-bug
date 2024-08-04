package bootstrap

import "strings"

// RefsToIds parse a slice of git references and return the corresponding Entity's Id.
func RefsToIds(refs []string) []Id {
	ids := make([]Id, len(refs))

	for i, ref := range refs {
		ids[i] = RefToId(ref)
	}

	return ids
}

// RefToId parse a git reference and return the corresponding Entity's Id.
func RefToId(ref string) Id {
	split := strings.Split(ref, "/")
	return Id(split[len(split)-1])
}
