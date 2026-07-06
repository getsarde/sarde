package asset

import "errors"

// ErrAVIFNotAvailable is returned when AVIF encoding is requested but the
// build was not compiled with the avif build tag. Declared in an untagged
// file so both build modes compile (image.go references it unconditionally).
var ErrAVIFNotAvailable = errors.New("AVIF encoding not available: rebuild with -tags avif")
