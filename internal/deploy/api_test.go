package deploy

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testClient(t *testing.T, h http.HandlerFunc) *apiClient {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return newAPIClient("test", "", "secret-token", Options{BaseURL: srv.URL}, nil)
}

func TestAPIClient_HeadersAndDecode(t *testing.T) {
	stubSleep(t)
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Errorf("Authorization = %q", got)
		}
		if ua := r.Header.Get("User-Agent"); !strings.HasPrefix(ua, "sarde/") {
			t.Errorf("User-Agent = %q", ua)
		}
		w.Write([]byte(`{"name":"ok"}`))
	})
	var out struct{ Name string }
	if err := c.do(context.Background(), apiRequest{Method: "GET", Path: "/x"}, &out); err != nil {
		t.Fatal(err)
	}
	if out.Name != "ok" {
		t.Errorf("decoded %+v", out)
	}
}

func TestAPIClient_RetriesWithRetryAfter(t *testing.T) {
	waits := stubSleep(t)
	var calls atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"a":1}` {
			t.Errorf("attempt %d body = %q; the body factory must replay", calls.Load()+1, body)
		}
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "3")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{}`))
	})
	err := c.do(context.Background(), apiRequest{Method: "POST", Path: "/x", Body: jsonBody(map[string]int{"a": 1})}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Errorf("calls = %d, want 2", calls.Load())
	}
	if len(*waits) != 1 || (*waits)[0] != 3*time.Second {
		t.Errorf("waits = %v, want [3s]", *waits)
	}
}

func TestAPIClient_GivesUpAfterMaxAttempts(t *testing.T) {
	stubSleep(t)
	var calls atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	err := c.do(context.Background(), apiRequest{Method: "GET", Path: "/x"}, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 503 {
		t.Fatalf("err = %v, want a 503 APIError", err)
	}
	if calls.Load() != maxAttempts {
		t.Errorf("calls = %d, want %d", calls.Load(), maxAttempts)
	}
}

func TestAPIClient_NoRetryOn4xx(t *testing.T) {
	stubSleep(t)
	var calls atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"bad token"}`))
	})
	err := c.do(context.Background(), apiRequest{Method: "GET", Path: "/x", Endpoint: "GET /x"}, nil)
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", calls.Load())
	}
	if ErrorCode(err) != CodeAuth || !strings.Contains(err.Error(), "bad token") || !strings.Contains(err.Error(), "GET /x") {
		t.Errorf("err = %v (code %s)", err, ErrorCode(err))
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Error("the token leaked into an error message")
	}
}

func TestAPIClient_CancelledContextStops(t *testing.T) {
	stubSleep(t)
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := c.do(ctx, apiRequest{Method: "GET", Path: "/x"}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestBackoff(t *testing.T) {
	h := http.Header{}
	h.Set("Retry-After", "120")
	if got := backoff(1, h); got != maxBackoff {
		t.Errorf("Retry-After above the cap = %s, want %s", got, maxBackoff)
	}
	for attempt := 1; attempt <= 3; attempt++ {
		got := backoff(attempt, nil)
		base := time.Second << (attempt - 1)
		if got < base*8/10 || got > base*12/10 {
			t.Errorf("attempt %d backoff %s outside ±20%% of %s", attempt, got, base)
		}
	}
}
