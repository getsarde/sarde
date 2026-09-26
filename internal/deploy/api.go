package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/getsarde/sarde/internal/version"
)

// APIError is a non-success response from a provider API.
type APIError struct {
	Provider  string
	Endpoint  string // "POST /sites/{id}/deploys", ids elided
	Status    int
	Code      string // provider error code, when the body carries one
	Message   string
	Retryable bool
	// Body is the raw response body, for callers that need structured
	// fields the generic decoder drops (Vercel's missing_files list).
	Body []byte
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = http.StatusText(e.Status)
	}
	return fmt.Sprintf("%s: %s returned HTTP %d: %s", e.Provider, e.Endpoint, e.Status, msg)
}

const (
	maxAttempts      = 5
	maxBackoff       = 60 * time.Second
	jsonCallTimeout  = 2 * time.Minute
	uploadTimeout    = 10 * time.Minute
	maxErrorBodySize = 2 << 10
)

// apiSleep waits d or until ctx is done. Tests replace it so retries and
// polling run instantly.
var apiSleep = func(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// bodyFunc produces a request body. It is a factory so a retry gets a fresh
// reader (a reopened file, a re-encoded payload). size is -1 when unknown.
type bodyFunc func() (r io.ReadCloser, size int64, err error)

func jsonBody(v any) bodyFunc {
	return func() (io.ReadCloser, int64, error) {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, 0, err
		}
		return io.NopCloser(bytes.NewReader(b)), int64(len(b)), nil
	}
}

func bytesBody(b []byte) bodyFunc {
	return func() (io.ReadCloser, int64, error) {
		return io.NopCloser(bytes.NewReader(b)), int64(len(b)), nil
	}
}

func fileBody(path string) bodyFunc {
	return func() (io.ReadCloser, int64, error) {
		f, err := os.Open(path)
		if err != nil {
			return nil, 0, err
		}
		info, err := f.Stat()
		if err != nil {
			f.Close()
			return nil, 0, err
		}
		return f, info.Size(), nil
	}
}

// apiRequest is one call to a provider API.
type apiRequest struct {
	Method      string
	Path        string // appended to the client base URL
	Endpoint    string // label for errors; defaults to Method + Path
	Query       url.Values
	Header      http.Header
	ContentType string
	Body        bodyFunc
	// Token overrides the client's bearer token for this call (Cloudflare's
	// upload JWT). Empty means the client token.
	Token   string
	Timeout time.Duration
}

// apiClient is a small JSON-over-HTTP client with bearer auth and retries.
type apiClient struct {
	provider string
	base     string
	token    string
	hc       *http.Client
	rep      Reporter
	// decodeError extracts a provider error code and message from a
	// failure body. Nil means the generic decoder.
	decodeError func(body []byte) (code, message string)
}

func newAPIClient(provider, defaultBase, token string, opts Options, rep Reporter) *apiClient {
	base := defaultBase
	if opts.BaseURL != "" {
		base = opts.BaseURL
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = defaultHTTPClient()
	}
	return &apiClient{
		provider: provider,
		base:     strings.TrimRight(base, "/"),
		token:    token,
		hc:       hc,
		rep:      orNop(rep),
	}
}

func defaultHTTPClient() *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.ResponseHeaderTimeout = 90 * time.Second
	tr.TLSHandshakeTimeout = 15 * time.Second
	// No client-wide Timeout: uploads can legitimately take minutes; each
	// attempt carries its own context deadline instead.
	return &http.Client{Transport: tr}
}

func userAgent() string { return "sarde/" + version.Version }

// do sends req, retrying 429, 5xx and transport failures with backoff, and
// decodes a JSON success body into out when out is non-nil.
func (c *apiClient) do(ctx context.Context, req apiRequest, out any) error {
	endpoint := req.Endpoint
	if endpoint == "" {
		endpoint = req.Method + " " + req.Path
	}
	timeout := req.Timeout
	if timeout == 0 {
		timeout = jsonCallTimeout
	}
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		status, header, body, err := c.once(ctx, req, timeout)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			lastErr = fmt.Errorf("%s: %s: %w", c.provider, endpoint, err)
			if attempt == maxAttempts {
				break
			}
			if err := apiSleep(ctx, backoff(attempt, nil)); err != nil {
				return err
			}
			continue
		}
		if status >= 200 && status < 300 {
			if out == nil || len(bytes.TrimSpace(body)) == 0 {
				return nil
			}
			if err := json.Unmarshal(body, out); err != nil {
				return fmt.Errorf("%s: %s: decoding response: %w", c.provider, endpoint, err)
			}
			return nil
		}
		apiErr := c.apiError(endpoint, status, body)
		if !apiErr.Retryable || attempt == maxAttempts {
			return apiErr
		}
		lastErr = apiErr
		wait := backoff(attempt, header)
		if status == http.StatusTooManyRequests {
			c.rep.Log(LevelWarn, fmt.Sprintf("%s rate limited the upload; waiting %s", c.provider, wait.Round(time.Second)))
		}
		if err := apiSleep(ctx, wait); err != nil {
			return err
		}
	}
	return lastErr
}

// once performs a single attempt and returns the status, headers and body.
func (c *apiClient) once(ctx context.Context, req apiRequest, timeout time.Duration) (int, http.Header, []byte, error) {
	attemptCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	u := c.base + req.Path
	if len(req.Query) > 0 {
		u += "?" + req.Query.Encode()
	}
	var body io.ReadCloser
	size := int64(0)
	if req.Body != nil {
		r, n, err := req.Body()
		if err != nil {
			return 0, nil, nil, err
		}
		body, size = r, n
	}
	hreq, err := http.NewRequestWithContext(attemptCtx, req.Method, u, body)
	if err != nil {
		if body != nil {
			body.Close()
		}
		return 0, nil, nil, err
	}
	if body != nil && size >= 0 {
		hreq.ContentLength = size
	}
	for k, vs := range req.Header {
		for _, v := range vs {
			hreq.Header.Add(k, v)
		}
	}
	if req.ContentType != "" {
		hreq.Header.Set("Content-Type", req.ContentType)
	}
	hreq.Header.Set("User-Agent", userAgent())
	hreq.Header.Set("Accept", "application/json")
	token := c.token
	if req.Token != "" {
		token = req.Token
	}
	if token != "" {
		hreq.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.hc.Do(hreq)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return 0, nil, nil, err
	}
	return resp.StatusCode, resp.Header, respBody, nil
}

func (c *apiClient) apiError(endpoint string, status int, body []byte) *APIError {
	decode := c.decodeError
	if decode == nil {
		decode = genericErrorBody
	}
	code, msg := decode(body)
	if msg == "" {
		msg = strings.TrimSpace(string(truncate(body, maxErrorBodySize)))
	}
	return &APIError{
		Provider:  c.provider,
		Endpoint:  endpoint,
		Status:    status,
		Code:      code,
		Message:   msg,
		Retryable: status == http.StatusTooManyRequests || status >= 500,
		Body:      body,
	}
}

// genericErrorBody understands the common shapes: {"message"}, {"error":
// "..."} and {"error": {"code", "message"}}.
func genericErrorBody(body []byte) (string, string) {
	var v struct {
		Message string          `json:"message"`
		Code    json.RawMessage `json:"code"`
		Error   json.RawMessage `json:"error"`
	}
	if json.Unmarshal(body, &v) != nil {
		return "", ""
	}
	code := rawString(v.Code)
	msg := v.Message
	if len(v.Error) > 0 {
		var nested struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if json.Unmarshal(v.Error, &nested) == nil && (nested.Message != "" || nested.Code != "") {
			if nested.Code != "" {
				code = nested.Code
			}
			if nested.Message != "" {
				msg = nested.Message
			}
		} else if s := rawString(v.Error); s != "" && msg == "" {
			msg = s
		}
	}
	return code, msg
}

// rawString renders a JSON string or number as text.
func rawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		return n.String()
	}
	return ""
}

func truncate(b []byte, n int) []byte {
	if len(b) <= n {
		return b
	}
	return b[:n]
}

// backoff returns how long to wait before the next attempt: Retry-After
// (seconds or HTTP date), then X-RateLimit-Reset (epoch seconds, Netlify),
// then jittered exponential backoff. Always capped at maxBackoff.
func backoff(attempt int, h http.Header) time.Duration {
	if h != nil {
		if v := h.Get("Retry-After"); v != "" {
			if secs, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && secs >= 0 {
				return capWait(time.Duration(secs) * time.Second)
			}
			if t, err := http.ParseTime(v); err == nil {
				return capWait(time.Until(t))
			}
		}
		if v := h.Get("X-RateLimit-Reset"); v != "" {
			if epoch, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
				return capWait(time.Until(time.Unix(epoch, 0)))
			}
		}
	}
	base := time.Second << (attempt - 1)
	jitter := 0.8 + rand.Float64()*0.4 //nolint:gosec // jitter, not security
	return capWait(time.Duration(float64(base) * jitter))
}

func capWait(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}

// uploadAll runs fn for each index in [0, n) with at most concurrency calls
// in flight, stopping at the first error.
func uploadAll(ctx context.Context, n, concurrency int, fn func(ctx context.Context, i int) error) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)
	for i := range n {
		g.Go(func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return fn(ctx, i)
		})
	}
	return g.Wait()
}

// poll calls check every interval until it reports done, fails, or timeout
// elapses. It returns an error naming what was awaited on timeout.
func poll(ctx context.Context, what string, interval, timeout time.Duration, check func(ctx context.Context) (bool, error)) error {
	deadline := time.Now().Add(timeout)
	for {
		done, err := check(ctx)
		if err != nil || done {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s waiting for %s", timeout, what)
		}
		if err := apiSleep(ctx, interval); err != nil {
			return err
		}
	}
}
