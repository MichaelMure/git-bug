package entity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInterleaved(t *testing.T) {
	primary := Id("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWX______________")
	secondary := Id("YZ0123456789+/________________________________________________")
	expectedId := CombinedId("aYbZc0def1ghij2klmn3opqr4stuv5wxyz6ABCD7EFGH8IJKL9MNOP+QRST/UVWX")

	interleaved := CombineIds(primary, secondary)
	require.Equal(t, expectedId, interleaved)

	// full length
	splitPrimary, splitSecondary := SeparateIds(interleaved.String())
	require.Equal(t, string(primary[:50]), splitPrimary)
	require.Equal(t, string(secondary[:14]), splitSecondary)

	// partial
	splitPrimary, splitSecondary = SeparateIds(string(expectedId[:7]))
	require.Equal(t, string(primary[:4]), splitPrimary)
	require.Equal(t, string(secondary[:3]), splitSecondary)

	// partial
	splitPrimary, splitSecondary = SeparateIds(string(expectedId[:10]))
	require.Equal(t, string(primary[:6]), splitPrimary)
	require.Equal(t, string(secondary[:4]), splitSecondary)

	// partial
	splitPrimary, splitSecondary = SeparateIds(string(expectedId[:16]))
	require.Equal(t, string(primary[:11]), splitPrimary)
	require.Equal(t, string(secondary[:5]), splitSecondary)
}
