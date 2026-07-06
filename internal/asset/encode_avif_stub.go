//go:build !avif

package asset

import (
	"image"
	"io"
)

func encodeAVIF(w io.Writer, img image.Image, quality int) error {
	return ErrAVIFNotAvailable
}
