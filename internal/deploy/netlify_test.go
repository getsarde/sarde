package deploy

import (
	"context"
	"crypto/sha1" //nolint:gosec // mirrors the content address
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
)

// fakeNetlify is a state machine over the endpoints the deployer uses.
type fakeNetlify struct {
	t        *testing.T
	mu       sync.Mutex
	requests []string
	manifest map[string]string
	uploads  map[string]string // path -> body
	polls    int
	cancel   bool
	failSite int // status for GET /sites/{id}; 0 = ok
	errState bool
	block    chan struct{} // when set, the first poll blocks until closed
}

func sha1Hex(s string) string {
	sum := sha1.Sum([]byte(s)) //nolint:gosec
	return hex.EncodeToString(sum[:])
}

func (f *fakeNetlify) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.requests = append(f.requests, r.Method+" "+r.URL.EscapedPath())
	f.mu.Unlock()
	switch {
	case r.Method == "GET" && r.URL.Path == "/sites/site-1":
		if f.failSite != 0 {
			w.WriteHeader(f.failSite)
			w.Write([]byte(`{"message":"Access denied"}`))
			return
		}
		w.Write([]byte(`{"id":"site-1","name":"docs","ssl_url":"https://docs.example.com","admin_url":"https://app.netlify.com/sites/docs","published_deploy":{"id":"dep-1"}}`))
	case r.Method == "POST" && r.URL.Path == "/sites/site-1/deploys":
		var body struct {
			Files map[string]string `json:"files"`
			Draft bool              `json:"draft"`
			Async bool              `json:"async"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.Draft || !body.Async {
			f.t.Errorf("deploy body draft=%v async=%v", body.Draft, body.Async)
		}
		f.mu.Lock()
		f.manifest = body.Files
		f.mu.Unlock()
		w.Write([]byte(`{"id":"dep-1","state":"preparing"}`))
	case r.Method == "GET" && r.URL.Path == "/deploys/dep-1":
		f.mu.Lock()
		f.polls++
		n := f.polls
		block := f.block
		f.mu.Unlock()
		if block != nil && n == 1 {
			select {
			case <-block:
			case <-r.Context().Done():
			}
			w.WriteHeader(503)
			return
		}
		switch {
		case f.errState:
			w.Write([]byte(`{"id":"dep-1","state":"error","error_message":"build image unavailable"}`))
		case n == 1:
			w.Write([]byte(`{"id":"dep-1","state":"preparing"}`))
		case n == 2:
			// "shared" is required once but carried by two files.
			req, _ := json.Marshal([]string{sha1Hex("home"), sha1Hex("shared")})
			w.Write([]byte(`{"id":"dep-1","state":"prepared","required":` + string(req) + `}`))
		case n == 3:
			w.Write([]byte(`{"id":"dep-1","state":"uploaded"}`))
		default:
			w.Write([]byte(`{"id":"dep-1","state":"ready","deploy_ssl_url":"https://dep-1--docs.netlify.app"}`))
		}
	case r.Method == "PUT" && strings.HasPrefix(r.URL.Path, "/deploys/dep-1/files/"):
		if ct := r.Header.Get("Content-Type"); ct != "application/octet-stream" {
			f.t.Errorf("upload content type %q", ct)
		}
		b, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		if f.uploads == nil {
			f.uploads = map[string]string{}
		}
		f.uploads[strings.TrimPrefix(r.URL.EscapedPath(), "/deploys/dep-1/files/")] = string(b)
		f.mu.Unlock()
		w.Write([]byte(`{}`))
	case r.Method == "POST" && r.URL.Path == "/deploys/dep-1/cancel":
		f.mu.Lock()
		f.cancel = true
		f.mu.Unlock()
		w.Write([]byte(`{}`))
	default:
		f.t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(404)
	}
}

func netlifyUnderTest(t *testing.T, f *fakeNetlify) *NetlifyDeployer {
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return &NetlifyDeployer{SiteID: "site-1", opts: Options{
		BaseURL: srv.URL,
		Getenv:  fakeEnv(map[string]string{"NETLIFY_AUTH_TOKEN": "tok"}),
	}}
}

func TestNetlifyDeploy_UploadsOnlyRequiredFiles(t *testing.T) {
	stubSleep(t)
	f := &fakeNetlify{t: t}
	d := netlifyUnderTest(t, f)
	dist := writeDist(t, map[string]string{
		"index.html":     "home",
		"a b/copy.txt":   "shared",
		"other/copy.txt": "shared",
		"cached.css":     "already on netlify",
	})
	rep := &recordingReporter{}
	res, err := d.Deploy(context.Background(), dist, rep)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}

	// Manifest: leading slashes, SHA1 digests, no lock file.
	if len(f.manifest) != 4 {
		t.Errorf("manifest has %d entries: %v", len(f.manifest), f.manifest)
	}
	if f.manifest["/index.html"] != sha1Hex("home") || f.manifest["/a b/copy.txt"] != sha1Hex("shared") {
		t.Errorf("manifest = %v", f.manifest)
	}
	for p := range f.manifest {
		if strings.Contains(p, "sarde.lock") {
			t.Errorf("lock file in manifest: %s", p)
		}
	}

	// Uploads: every file carrying a required digest, paths escaped.
	var uploaded []string
	for p := range f.uploads {
		uploaded = append(uploaded, p)
	}
	sort.Strings(uploaded)
	want := []string{"a%20b/copy.txt", "index.html", "other/copy.txt"}
	if strings.Join(uploaded, ",") != strings.Join(want, ",") {
		t.Errorf("uploads = %v, want %v", uploaded, want)
	}

	if res.URL != "https://docs.example.com" || res.DeployURL != "https://dep-1--docs.netlify.app" || res.DeployID != "dep-1" {
		t.Errorf("result = %+v", res)
	}
	if res.FilesTotal != 4 || res.FilesUploaded != 3 {
		t.Errorf("counts = %d/%d", res.FilesUploaded, res.FilesTotal)
	}
	if f.cancel {
		t.Error("a successful deploy must not be cancelled")
	}
	last := rep.progress[len(rep.progress)-1]
	if last.Step != StepUpload || last.Done != 3 || last.Total != 3 {
		t.Errorf("last progress = %+v", last)
	}
}

func TestNetlifyDeploy_ErrorState(t *testing.T) {
	stubSleep(t)
	f := &fakeNetlify{t: t, errState: true}
	d := netlifyUnderTest(t, f)
	_, err := d.Deploy(context.Background(), writeDist(t, map[string]string{"index.html": "home"}), nil)
	if err == nil || !strings.Contains(err.Error(), "build image unavailable") {
		t.Errorf("err = %v", err)
	}
}

func TestNetlifyDeploy_AuthFailure(t *testing.T) {
	stubSleep(t)
	f := &fakeNetlify{t: t, failSite: 401}
	d := netlifyUnderTest(t, f)
	_, err := d.Deploy(context.Background(), writeDist(t, map[string]string{"index.html": "home"}), nil)
	if ErrorCode(err) != CodeAuth {
		t.Errorf("err = %v, code %q, want auth", err, ErrorCode(err))
	}
	if len(f.requests) != 1 {
		t.Errorf("requests = %v, want only the site lookup", f.requests)
	}
}

func TestNetlifyDeploy_RejectsHashInPath(t *testing.T) {
	stubSleep(t)
	f := &fakeNetlify{t: t}
	d := netlifyUnderTest(t, f)
	_, err := d.Deploy(context.Background(), writeDist(t, map[string]string{"c#.html": "x"}), nil)
	if ErrorCode(err) != CodeConfig {
		t.Errorf("err = %v", err)
	}
	for _, r := range f.requests {
		if strings.HasPrefix(r, "POST") {
			t.Errorf("no deploy may be created, saw %v", f.requests)
		}
	}
}

func TestNetlifyDeploy_CancelCancelsRemoteDeploy(t *testing.T) {
	stubSleep(t)
	f := &fakeNetlify{t: t, block: make(chan struct{})}
	defer close(f.block)
	d := netlifyUnderTest(t, f)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := d.Deploy(ctx, writeDist(t, map[string]string{"index.html": "home"}), nil)
		done <- err
	}()
	// Wait until the deployer is polling, then cancel.
	for {
		f.mu.Lock()
		n := f.polls
		f.mu.Unlock()
		if n > 0 {
			break
		}
	}
	cancel()
	err := <-done
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.cancel {
		t.Error("the remote deploy was not cancelled")
	}
}

func TestNetlifyCheck(t *testing.T) {
	stubSleep(t)
	f := &fakeNetlify{t: t}
	d := netlifyUnderTest(t, f)
	res, err := d.Check(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Target != "docs" || res.URL != "https://docs.example.com" {
		t.Errorf("check = %+v", res)
	}
	if len(f.requests) != 1 {
		t.Errorf("check made %v, want one GET", f.requests)
	}
}
