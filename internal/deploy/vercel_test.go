package deploy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type vercelRef struct {
	File string `json:"file"`
	SHA  string `json:"sha"`
	Size int64  `json:"size"`
}

type fakeVercel struct {
	t          *testing.T
	mu         sync.Mutex
	creates    int
	createBody []map[string]any
	files      []vercelRef
	uploads    map[string]string // sha -> body
	queries    []string
	polls      int
	alwaysMiss bool
	failState  string
	projectErr int
}

func (f *fakeVercel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.queries = append(f.queries, r.URL.Path+"?"+r.URL.RawQuery)
	f.mu.Unlock()
	switch {
	case r.Method == "GET" && r.URL.Path == "/v9/projects/prj_1":
		if f.projectErr != 0 {
			w.WriteHeader(f.projectErr)
			w.Write([]byte(`{"error":{"code":"not_found","message":"Project not found"}}`))
			return
		}
		w.Write([]byte(`{"id":"prj_1","name":"docs","targets":{"production":{"alias":["docs.vercel.app"]}}}`))
	case r.Method == "POST" && r.URL.Path == "/v13/deployments":
		if r.URL.Query().Get("prebuilt") != "1" {
			f.t.Error("deployment created without prebuilt=1")
		}
		var body map[string]any
		raw, _ := io.ReadAll(r.Body)
		json.Unmarshal(raw, &body)
		var refs struct {
			Files []vercelRef `json:"files"`
		}
		json.Unmarshal(raw, &refs)
		f.mu.Lock()
		f.creates++
		n := f.creates
		f.createBody = append(f.createBody, body)
		f.files = refs.Files
		f.mu.Unlock()
		if n == 1 || f.alwaysMiss {
			var missing []string
			for _, ref := range refs.Files {
				if !strings.HasSuffix(ref.File, "cached.css") {
					missing = append(missing, ref.SHA)
				}
			}
			m, _ := json.Marshal(missing)
			w.WriteHeader(400)
			w.Write([]byte(`{"error":{"code":"missing_files","message":"Missing files","missing":` + string(m) + `}}`))
			return
		}
		w.Write([]byte(`{"id":"dpl_1","url":"docs-abc.vercel.app","readyState":"QUEUED"}`))
	case r.Method == "POST" && r.URL.Path == "/v2/files":
		b, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		if f.uploads == nil {
			f.uploads = map[string]string{}
		}
		f.uploads[r.Header.Get("x-vercel-digest")] = string(b)
		f.mu.Unlock()
		w.Write([]byte(`{}`))
	case r.Method == "GET" && r.URL.Path == "/v13/deployments/dpl_1":
		f.mu.Lock()
		f.polls++
		n := f.polls
		f.mu.Unlock()
		switch {
		case f.failState != "":
			w.Write([]byte(`{"id":"dpl_1","readyState":"` + f.failState + `","errorMessage":"Static files exceeded quota"}`))
		case n == 1:
			w.Write([]byte(`{"id":"dpl_1","url":"docs-abc.vercel.app","readyState":"BUILDING"}`))
		case n == 2:
			// Ready but not aliased yet: keep waiting.
			w.Write([]byte(`{"id":"dpl_1","url":"docs-abc.vercel.app","readyState":"READY","aliasAssigned":null}`))
		default:
			w.Write([]byte(`{"id":"dpl_1","url":"docs-abc.vercel.app","readyState":"READY","aliasAssigned":1712345678,"alias":["docs.vercel.app"],"inspectorUrl":"https://vercel.com/acme/docs/dpl_1"}`))
		}
	default:
		f.t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		w.WriteHeader(404)
	}
}

func vercelUnderTest(t *testing.T, f *fakeVercel, team string) *VercelDeployer {
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return &VercelDeployer{ProjectID: "prj_1", TeamID: team, opts: Options{
		BaseURL: srv.URL,
		Getenv:  fakeEnv(map[string]string{"VERCEL_TOKEN": "tok"}),
	}}
}

func TestVercelDeploy_PrebuiltLayoutAndMissingFiles(t *testing.T) {
	stubSleep(t)
	f := &fakeVercel{t: t}
	d := vercelUnderTest(t, f, "team_9")
	dist := writeDist(t, map[string]string{
		"index.html":  "home",
		"404.html":    "missing",
		"cached.css":  "css",
		"vercel.json": `{"redirects":[{"source":"/old/","destination":"/new/","permanent":true},{"source":"/tmp","destination":"https://x.dev","permanent":false}]}`,
	})
	res, err := d.Deploy(context.Background(), dist, &recordingReporter{})
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}

	names := map[string]vercelRef{}
	for _, ref := range f.files {
		names[ref.File] = ref
		if strings.HasPrefix(ref.File, "/") || strings.Contains(ref.File, `\`) {
			t.Errorf("bad file name %q", ref.File)
		}
	}
	for _, want := range []string{".vercel/output/static/index.html", ".vercel/output/static/404.html", ".vercel/output/config.json"} {
		if _, ok := names[want]; !ok {
			t.Errorf("missing %s in %v", want, f.files)
		}
	}
	if _, ok := names[".vercel/output/static/vercel.json"]; ok {
		t.Error("vercel.json must become routes, not a static file")
	}
	if names[".vercel/output/static/index.html"].SHA != sha1Hex("home") {
		t.Error("index.html digest mismatch")
	}

	// The generated config is uploaded and carries the translated routes.
	cfg, ok := f.uploads[names[".vercel/output/config.json"].SHA]
	if !ok {
		t.Fatalf("config.json was not uploaded; uploads = %v", f.uploads)
	}
	for _, want := range []string{`"src": "^/old/?$"`, `"status": 308`, `"Location": "/new/"`, `"status": 307`, `"handle": "filesystem"`, `"dest": "/404.html"`} {
		if !strings.Contains(cfg, want) {
			t.Errorf("config.json lacks %s:\n%s", want, cfg)
		}
	}
	if _, ok := f.uploads[names[".vercel/output/static/cached.css"].SHA]; ok {
		t.Error("a file Vercel already had was uploaded")
	}

	body := f.createBody[1]
	if body["target"] != "production" || body["name"] != "docs" || body["project"] != "prj_1" {
		t.Errorf("deployment body = %v", body)
	}
	for _, q := range f.queries {
		if !strings.Contains(q, "teamId=team_9") {
			t.Errorf("request without teamId: %s", q)
		}
	}
	if res.URL != "https://docs.vercel.app" || res.DeployURL != "https://docs-abc.vercel.app" || res.AdminURL == "" {
		t.Errorf("result = %+v", res)
	}
	if f.polls < 3 {
		t.Errorf("polls = %d; READY without an alias must keep waiting", f.polls)
	}
}

func TestVercelDeploy_SecondMissingIsFatal(t *testing.T) {
	stubSleep(t)
	f := &fakeVercel{t: t, alwaysMiss: true}
	d := vercelUnderTest(t, f, "")
	_, err := d.Deploy(context.Background(), writeDist(t, map[string]string{"index.html": "home"}), nil)
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Errorf("err = %v", err)
	}
	if f.creates != 2 {
		t.Errorf("creates = %d, want 2", f.creates)
	}
}

func TestVercelDeploy_ErrorState(t *testing.T) {
	stubSleep(t)
	f := &fakeVercel{t: t, failState: "ERROR"}
	d := vercelUnderTest(t, f, "")
	_, err := d.Deploy(context.Background(), writeDist(t, map[string]string{"index.html": "home"}), nil)
	if err == nil || !strings.Contains(err.Error(), "Static files exceeded quota") {
		t.Errorf("err = %v", err)
	}
}

func TestVercelCheck_TeamHint(t *testing.T) {
	stubSleep(t)
	f := &fakeVercel{t: t, projectErr: 404}
	d := vercelUnderTest(t, f, "")
	_, err := d.Check(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "deploy.team_id") {
		t.Errorf("err = %v, want the team_id hint", err)
	}
	if ErrorCode(err) != CodeNotFound {
		t.Errorf("code = %q", ErrorCode(err))
	}
}

func TestVercelTeamFromEnvWins(t *testing.T) {
	d := &VercelDeployer{TeamID: "team_cfg", opts: Options{Getenv: fakeEnv(map[string]string{"VERCEL_ORG_ID": "team_env"})}}
	if d.teamID() != "team_env" {
		t.Errorf("teamID = %q", d.teamID())
	}
}

func TestRedirectSourceRegex(t *testing.T) {
	cases := map[string]string{
		"/old/":     `^/old/?$`,
		"/old":      `^/old/?$`,
		"/":         `^/$`,
		"/a.b/c+d/": `^/a\.b/c\+d/?$`,
	}
	for in, want := range cases {
		if got := redirectSourceRegex(in); got != want {
			t.Errorf("redirectSourceRegex(%q) = %q, want %q", in, got, want)
		}
	}
}
