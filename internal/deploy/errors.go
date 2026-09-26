package deploy

import (
	"context"
	"errors"
	"fmt"
	"net"
)

// Error codes reported by ErrorCode. They reach Sarde Studio through the
// JSON error envelope, so the values are part of the CLI contract.
const (
	CodeAuth        = "auth"         // token missing scopes, rejected or expired
	CodeNotFound    = "not_found"    // site or project does not exist
	CodeRateLimited = "rate_limited" // provider throttled us past the retry budget
	CodeLimit       = "limit"        // file count or size over a provider limit
	CodeNetwork     = "network"      // could not reach the provider
	CodeCanceled    = "canceled"     // the caller cancelled
	CodeConfig      = "config"       // missing or invalid local configuration
	CodeProvider    = "provider"     // any other provider-side failure
)

// codedError attaches an error code to an error without changing its text.
type codedError struct {
	code string
	err  error
}

func (e *codedError) Error() string { return e.err.Error() }
func (e *codedError) Unwrap() error { return e.err }

func withCode(code string, err error) error {
	if err == nil {
		return nil
	}
	return &codedError{code: code, err: err}
}

func configErrorf(format string, args ...any) error {
	return withCode(CodeConfig, fmt.Errorf(format, args...))
}

func limitErrorf(format string, args ...any) error {
	return withCode(CodeLimit, fmt.Errorf(format, args...))
}

// ErrorCode classifies a deploy error for machine consumers. It returns ""
// for nil.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	}
	var coded *codedError
	if errors.As(err, &coded) {
		return coded.code
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return CodeCanceled
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.Status == 401 || apiErr.Status == 403:
			return CodeAuth
		case apiErr.Status == 404:
			return CodeNotFound
		case apiErr.Status == 429:
			return CodeRateLimited
		case apiErr.Status == 413:
			return CodeLimit
		}
		return CodeProvider
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return CodeNetwork
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return CodeNetwork
	}
	return CodeProvider
}
