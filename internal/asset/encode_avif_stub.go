//go:build !avif

package asset

import (
	"errors"
	"image"
	"io"
)

// ErrAVIFNotAvailable is returned when AVIF encoding is requested but the
// build was not compiled with the avif build tag.
var ErrAVIFNotAvailable = errors.New("AVIF encoding not available: rebuild with -tags avif")

func encodeAVIF(w io.Writer, img image.Image, quality int) error {
	return ErrAVIFNotAvailable
}
