//go:build avif

package asset

import (
	"image"
	"io"

	"github.com/gen2brain/avif"
)

func encodeAVIF(w io.Writer, img image.Image, quality int) error {
	return avif.Encode(w, img, avif.Options{Quality: quality})
}
