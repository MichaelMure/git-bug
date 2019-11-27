package markdown

import "github.com/eliukblau/pixterm/ansimage"

type Options func(r *renderer)

// DitheringMode type is used for image scale dithering mode constants.
type DitheringMode uint8

const (
	NoDithering = DitheringMode(iota)
	DitheringWithBlocks
	DitheringWithChars
)

// Dithering mode for ansimage
// Default is fine directly through a terminal
// DitheringWithBlocks is recommended if a terminal UI library is used
func WithImageDithering(mode DitheringMode) Options {
	return func(r *renderer) {
		r.imageDithering = ansimage.DitheringMode(mode)
	}
}
