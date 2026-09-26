package deploy

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/zeebo/blake3"
)

// The BLAKE3 reference test vector for the empty input pins the library
// itself (https://github.com/BLAKE3-team/BLAKE3/blob/master/test_vectors).
func TestBlake3KnownAnswer(t *testing.T) {
	sum := blake3.Sum256(nil)
	if got := hex.EncodeToString(sum[:]); got != "af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262" {
		t.Fatalf("blake3(\"\") = %s", got)
	}
}

// cloudflareHash must equal wrangler's hashFile:
// blake3hash(base64(contents) + extname(path).substring(1)).hex.slice(0, 32).
func TestCloudflareHash_MatchesWranglerFormula(t *testing.T) {
	cases := []struct {
		data, rel, input string
	}{
		{"hello", "index.html", base64.StdEncoding.EncodeToString([]byte("hello")) + "html"},
		{"", ".nojekyll", ""},
		{"x", "a/b/app.min.js", base64.StdEncoding.EncodeToString([]byte("x")) + "js"},
		{"body", "README", base64.StdEncoding.EncodeToString([]byte("body"))},
	}
	for _, c := range cases {
		want := blake3.Sum256([]byte(c.input))
		if got := cloudflareHash([]byte(c.data), c.rel); got != hex.EncodeToString(want[:])[:32] {
			t.Errorf("cloudflareHash(%q, %q) = %s", c.data, c.rel, got)
		}
	}
	// .nojekyll has no extension in Node's sense, so it hashes like an
	// empty file without an extension, not like "nojekyll".
	if cloudflareHash(nil, ".nojekyll") == cloudflareHash(nil, "x.nojekyll") {
		t.Error("dotfile extension handling differs from Node")
	}
}

func fakeJWT(maxFiles int) string {
	payload, _ := json.Marshal(map[string]any{"max_file_count_allowed": maxFiles})
	return "e30." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

type cfUploaded struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Base64   bool   `json:"base64"`
	Metadata struct {
		ContentType string `json:"contentType"`
	} `json:"metadata"`
}

type fakeCloudflare struct {
	t           *testing.T
	mu          sync.Mutex
	maxFiles    int
	tokens      int
	expireOnce  bool
	checkHashes []string
	uploaded    map[string]cfUploaded
	upsertFail  bool
	manifest    map[string]string
	form        map[string]string
	formFiles   map[string]string
	stageFail   bool
}

func (f *fakeCloudflare) ok(w http.ResponseWriter, result any) {
	b, _ := json.Marshal(map[string]any{"success": true, "errors": []any{}, "result": result})
	w.Write(b)
}

func (f *fakeCloudflare) fail(w http.ResponseWriter, status, code int, msg string) {
	w.WriteHeader(status)
	b, _ := json.Marshal(map[string]any{"success": false, "errors": []any{map[string]any{"code": code, "message": msg}}})
	w.Write(b)
}

func (f *fakeCloudflare) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const project = "/accounts/acct/pages/projects/docs"
	auth := r.Header.Get("Authorization")
	switch {
	case r.Method == "GET" && r.URL.Path == project:
		f.ok(w, map[string]any{"name": "docs", "subdomain": "docs.pages.dev", "domains": []string{"docs.pages.dev", "docs.example.com"}, "production_branch": "main"})
	case r.Method == "GET" && r.URL.Path == project+"/upload-token":
		f.mu.Lock()
		f.tokens++
		n := f.tokens
		f.mu.Unlock()
		f.ok(w, map[string]string{"jwt": fakeJWT(f.maxFiles) + fmt.Sprint(n)})
	case r.URL.Path == "/pages/assets/check-missing":
		if !strings.HasPrefix(auth, "Bearer e30.") {
			f.t.Errorf("check-missing must use the upload JWT, got %q", auth)
		}
		var body struct{ Hashes []string }
		json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		f.checkHashes = body.Hashes
		f.mu.Unlock()
		f.ok(w, body.Hashes[:len(body.Hashes)-1]) // the last one is already cached
	case r.URL.Path == "/pages/assets/upload":
		f.mu.Lock()
		expire := f.expireOnce && f.tokens == 1
		f.mu.Unlock()
		if expire {
			f.fail(w, 401, 8000013, "JWT expired")
			return
		}
		var items []cfUploaded
		if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
			f.t.Errorf("upload body is not valid JSON: %v", err)
		}
		f.mu.Lock()
		if f.uploaded == nil {
			f.uploaded = map[string]cfUploaded{}
		}
		for _, it := range items {
			f.uploaded[it.Key] = it
		}
		f.mu.Unlock()
		f.ok(w, nil)
	case r.URL.Path == "/pages/assets/upsert-hashes":
		if f.upsertFail {
			f.fail(w, 400, 8000099, "nope")
			return
		}
		f.ok(w, nil)
	case r.Method == "POST" && r.URL.Path == project+"/deployments":
		_, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		mr := multipart.NewReader(r.Body, params["boundary"])
		f.form, f.formFiles = map[string]string{}, map[string]string{}
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			b, _ := io.ReadAll(p)
			if p.FileName() != "" {
				f.formFiles[p.FormName()] = string(b)
			} else {
				f.form[p.FormName()] = string(b)
			}
		}
		json.Unmarshal([]byte(f.form["manifest"]), &f.manifest)
		f.ok(w, map[string]any{"id": "dep-9", "url": "https://abc.docs.pages.dev"})
	case r.Method == "GET" && r.URL.Path == project+"/deployments/dep-9":
		status := "success"
		if f.stageFail {
			status = "failure"
		}
		f.ok(w, map[string]any{"id": "dep-9", "url": "https://abc.docs.pages.dev", "latest_stage": map[string]string{"name": "deploy", "status": status}})
	case r.Method == "GET" && r.URL.Path == project+"/deployments/dep-9/history/logs":
		f.ok(w, map[string]any{"data": []map[string]string{{"line": "Initializing"}, {"line": "Error: asset too large"}}})
	default:
		f.t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		w.WriteHeader(404)
	}
}

func cloudflareUnderTest(t *testing.T, f *fakeCloudflare) *CloudflareDeployer {
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return &CloudflareDeployer{ProjectName: "docs", AccountID: "acct", opts: Options{
		BaseURL: srv.URL,
		Getenv:  fakeEnv(map[string]string{"CLOUDFLARE_API_TOKEN": "tok"}),
	}}
}

func TestCloudflareDeploy_FullFlow(t *testing.T) {
	stubSleep(t)
	f := &fakeCloudflare{t: t, maxFiles: 100, expireOnce: true}
	d := cloudflareUnderTest(t, f)
	dist := writeDist(t, map[string]string{
		"index.html":    "<h1>home</h1>",
		"assets/app.js": "console.log(1)",
		"z.css":         "body{}",
		"_redirects":    "/old /new 301\n",
		"_headers":      "/*\n  X-Frame-Options: DENY\n",
		"a/.DS_Store":   "junk",
	})
	res, err := d.Deploy(context.Background(), dist, &recordingReporter{})
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}

	// Manifest: assets only, leading slashes, wrangler hashes.
	if len(f.manifest) != 3 {
		t.Errorf("manifest = %v", f.manifest)
	}
	if f.manifest["/index.html"] != cloudflareHash([]byte("<h1>home</h1>"), "index.html") {
		t.Errorf("index.html hash = %s", f.manifest["/index.html"])
	}
	for _, excluded := range []string{"/_redirects", "/_headers", "/a/.DS_Store", "/.sarde.lock"} {
		if _, ok := f.manifest[excluded]; ok {
			t.Errorf("%s must not be an asset", excluded)
		}
	}
	// Control files ride along as form files; the branch is production.
	if f.formFiles["_redirects"] != "/old /new 301\n" || f.formFiles["_headers"] == "" {
		t.Errorf("form files = %v", f.formFiles)
	}
	if f.form["branch"] != "main" {
		t.Errorf("branch = %q", f.form["branch"])
	}

	// Two of three hashes were missing; both uploaded as base64 with a type.
	if len(f.uploaded) != 2 {
		t.Errorf("uploaded %d assets, want 2", len(f.uploaded))
	}
	// check-missing reports the last hash as cached; every other asset
	// must have been uploaded with its bytes intact.
	cached := f.checkHashes[len(f.checkHashes)-1]
	for path, hash := range f.manifest {
		up, ok := f.uploaded[hash]
		if hash == cached {
			if ok {
				t.Errorf("%s was cached but uploaded anyway", path)
			}
			continue
		}
		if !ok {
			t.Errorf("%s (%s) was not uploaded", path, hash)
			continue
		}
		decoded, _ := base64.StdEncoding.DecodeString(up.Value)
		want, _ := os.ReadFile(filepath.Join(dist, filepath.FromSlash(strings.TrimPrefix(path, "/"))))
		if string(decoded) != string(want) || !up.Base64 || up.Metadata.ContentType != contentTypeFor(path) {
			t.Errorf("%s upload = %+v (%q)", path, up, decoded)
		}
	}
	// The expired JWT was refreshed exactly once.
	if f.tokens != 2 {
		t.Errorf("upload tokens fetched = %d, want 2", f.tokens)
	}
	if res.URL != "https://docs.example.com" || res.DeployURL != "https://abc.docs.pages.dev" || res.FilesUploaded != 2 {
		t.Errorf("result = %+v", res)
	}
}

func TestCloudflareDeploy_FileCountFromToken(t *testing.T) {
	stubSleep(t)
	f := &fakeCloudflare{t: t, maxFiles: 2}
	d := cloudflareUnderTest(t, f)
	_, err := d.Deploy(context.Background(), writeDist(t, map[string]string{"a.html": "a", "b.html": "b", "c.html": "c"}), nil)
	if ErrorCode(err) != CodeLimit {
		t.Errorf("err = %v, want a limit error", err)
	}
}

func TestCloudflareDeploy_RejectsWorker(t *testing.T) {
	stubSleep(t)
	f := &fakeCloudflare{t: t, maxFiles: 100}
	d := cloudflareUnderTest(t, f)
	_, err := d.Deploy(context.Background(), writeDist(t, map[string]string{"_worker.js": "export default {}"}), nil)
	if err == nil || !strings.Contains(err.Error(), "Pages Functions") {
		t.Errorf("err = %v", err)
	}
}

func TestCloudflareDeploy_StageFailureUsesLogs(t *testing.T) {
	stubSleep(t)
	f := &fakeCloudflare{t: t, maxFiles: 100, stageFail: true}
	d := cloudflareUnderTest(t, f)
	_, err := d.Deploy(context.Background(), writeDist(t, map[string]string{"index.html": "x"}), nil)
	if err == nil || !strings.Contains(err.Error(), "asset too large") {
		t.Errorf("err = %v", err)
	}
}

func TestCloudflareDeploy_UpsertFailureOnlyWarns(t *testing.T) {
	stubSleep(t)
	f := &fakeCloudflare{t: t, maxFiles: 100, upsertFail: true}
	d := cloudflareUnderTest(t, f)
	rep := &recordingReporter{}
	if _, err := d.Deploy(context.Background(), writeDist(t, map[string]string{"index.html": "x", "b.html": "y"}), rep); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	found := false
	for _, l := range rep.logs {
		if strings.Contains(l, "register") {
			found = true
		}
	}
	if !found {
		t.Errorf("logs = %v, want an upsert warning", rep.logs)
	}
}

func TestCloudflareAccountEnvWins(t *testing.T) {
	d := &CloudflareDeployer{AccountID: "cfg", opts: Options{Getenv: fakeEnv(map[string]string{"CLOUDFLARE_ACCOUNT_ID": "env"})}}
	if d.accountID() != "env" {
		t.Errorf("accountID = %q", d.accountID())
	}
}

func TestBucketAssets(t *testing.T) {
	var assets []cfAsset
	for i := range 5 {
		assets = append(assets, cfAsset{hash: fmt.Sprint(i), file: File{Size: int64(i+1) * (15 << 20)}})
	}
	buckets := bucketAssets(assets)
	for _, b := range buckets {
		var total int64
		for _, a := range b {
			total += a.file.Size
		}
		if total > cfBucketBytes && len(b) > 1 {
			t.Errorf("bucket over the byte cap: %d", total)
		}
	}
	if buckets[0][0].file.Size != 75<<20 {
		t.Error("largest file should go first")
	}
}

func TestJWTMaxFiles(t *testing.T) {
	if got := jwtMaxFiles(fakeJWT(123)); got != 123 {
		t.Errorf("jwtMaxFiles = %d", got)
	}
	if got := jwtMaxFiles("not-a-jwt"); got != cloudflareDefaultMax {
		t.Errorf("fallback = %d", got)
	}
}
