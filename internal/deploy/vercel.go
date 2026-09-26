package deploy

import (
	"context"
	"crypto/sha1" //nolint:gosec // Vercel addresses files by SHA1; not a security use.
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/getsarde/sarde/internal/version"
)

// VercelDeployer publishes a prebuilt site through Vercel's deployments API,
// using the Build Output API layout so Vercel serves the files as they are
// and never runs a build of its own. Credentials: the VERCEL_TOKEN
// environment variable, deploy.project_id (id or name) and, for team-owned
// projects, deploy.team_id (VERCEL_ORG_ID overrides it).
//
// API reference: https://vercel.com/docs/rest-api/reference/endpoints/deployments/create-a-new-deployment
// Output layout: https://vercel.com/docs/build-output-api/configuration
type VercelDeployer struct {
	ProjectID string
	TeamID    string
	opts      Options
}

const (
	vercelAPI          = "https://api.vercel.com"
	vercelUploads      = 8
	vercelMaxFiles     = 15000
	vercelReadyTimeout = 10 * time.Minute
	vercelPollInterval = 2 * time.Second
	vercelStaticPrefix = ".vercel/output/static/"
	vercelConfigPath   = ".vercel/output/config.json"
)

func (d *VercelDeployer) Name() string { return "vercel" }

type vercelProject struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Targets map[string]struct {
		Alias []string `json:"alias"`
	} `json:"targets"`
}

type vercelDeployment struct {
	ID            string          `json:"id"`
	URL           string          `json:"url"`
	ReadyState    string          `json:"readyState"`
	AliasAssigned json.RawMessage `json:"aliasAssigned"`
	Alias         []string        `json:"alias"`
	InspectorURL  string          `json:"inspectorUrl"`
	ErrorMessage  string          `json:"errorMessage"`
}

// vercelFile is one entry of the deployment: a path in the Build Output
// layout, its digest, and where its bytes come from.
type vercelFile struct {
	Name string
	SHA  string
	Size int64
	Body bodyFunc
}

func (d *VercelDeployer) teamID() string {
	return firstNonEmpty(d.opts.env("VERCEL_ORG_ID"), d.TeamID)
}

func (d *VercelDeployer) query() url.Values {
	q := url.Values{}
	if team := d.teamID(); team != "" {
		q.Set("teamId", team)
	}
	return q
}

func (d *VercelDeployer) client(rep Reporter) (*apiClient, error) {
	token, err := d.opts.requireEnv("VERCEL_TOKEN", "Vercel deploy")
	if err != nil {
		return nil, err
	}
	if d.ProjectID == "" {
		return nil, configErrorf("vercel deploy requires deploy.project_id in sarde.yaml")
	}
	return newAPIClient(d.Name(), vercelAPI, token, d.opts, rep), nil
}

func (d *VercelDeployer) project(ctx context.Context, c *apiClient) (*vercelProject, error) {
	var p vercelProject
	err := c.do(ctx, apiRequest{
		Method:   "GET",
		Path:     "/v9/projects/" + url.PathEscape(d.ProjectID),
		Endpoint: "GET /v9/projects/{id}",
		Query:    d.query(),
	}, &p)
	if err != nil {
		var apiErr *APIError
		if d.teamID() == "" && errors.As(err, &apiErr) && (apiErr.Status == 403 || apiErr.Status == 404) {
			return nil, fmt.Errorf("%w (a project owned by a team needs deploy.team_id, the team_... id)", err)
		}
		return nil, err
	}
	return &p, nil
}

// Check verifies the token can read the configured project.
func (d *VercelDeployer) Check(ctx context.Context, rep Reporter) (*CheckResult, error) {
	rep = orNop(rep)
	c, err := d.client(rep)
	if err != nil {
		return nil, err
	}
	rep.Step(StepPrepare, "Looking up the Vercel project")
	p, err := d.project(ctx, c)
	if err != nil {
		return nil, err
	}
	res := &CheckResult{Provider: d.Name(), TargetID: p.ID, Target: p.Name, Account: d.teamID()}
	if prod, ok := p.Targets["production"]; ok && len(prod.Alias) > 0 {
		res.URL = "https://" + prod.Alias[0]
	}
	return res, nil
}

func (d *VercelDeployer) Deploy(ctx context.Context, distDir string, rep Reporter) (*Result, error) {
	rep = orNop(rep)
	c, err := d.client(rep)
	if err != nil {
		return nil, err
	}

	rep.Step(StepPrepare, "Looking up the Vercel project")
	project, err := d.project(ctx, c)
	if err != nil {
		return nil, err
	}

	rep.Step(StepCollect, "Collecting files")
	files, err := collectFiles(distDir, rep)
	if err != nil {
		return nil, err
	}
	if len(files) > vercelMaxFiles {
		return nil, limitErrorf("vercel: %d files exceeds the %d file limit per deployment", len(files), vercelMaxFiles)
	}

	var redirectsFile *File
	var has404 bool
	static := make([]File, 0, len(files))
	for i, f := range files {
		switch f.Rel {
		case "vercel.json":
			// Translated into Build Output routes below; a vercel.json at
			// the root of a prebuilt upload would be served as a file.
			redirectsFile = &files[i]
			continue
		case "404.html":
			has404 = true
		}
		static = append(static, f)
	}

	config, err := buildVercelConfig(redirectsFile, has404)
	if err != nil {
		return nil, err
	}

	rep.Step(StepHash, fmt.Sprintf("Hashing %d files", len(static)))
	sums, err := hashAll(ctx, static, func(f File) (string, error) { return sha1File(f.Abs) }, rep)
	if err != nil {
		return nil, err
	}
	entries := make([]vercelFile, 0, len(static)+1)
	for i, f := range static {
		entries = append(entries, vercelFile{Name: vercelStaticPrefix + f.Rel, SHA: sums[i], Size: f.Size, Body: fileBody(f.Abs)})
	}
	configSum := sha1.Sum(config) //nolint:gosec // content address
	entries = append(entries, vercelFile{
		Name: vercelConfigPath,
		SHA:  hex.EncodeToString(configSum[:]),
		Size: int64(len(config)),
		Body: bytesBody(config),
	})

	rep.Step(StepFinalize, "Creating the deployment")
	dep, uploaded, bytesUp, err := d.createWithUploads(ctx, c, project, entries, rep)
	if err != nil {
		return nil, err
	}

	rep.Step(StepWait, "Waiting for Vercel to publish")
	err = poll(ctx, "vercel to publish", vercelPollInterval, vercelReadyTimeout, func(ctx context.Context) (bool, error) {
		err := c.do(ctx, apiRequest{
			Method:   "GET",
			Path:     "/v13/deployments/" + url.PathEscape(dep.ID),
			Endpoint: "GET /v13/deployments/{id}",
			Query:    d.query(),
		}, dep)
		if err != nil {
			return false, err
		}
		switch {
		case dep.ReadyState == "ERROR" || dep.ReadyState == "CANCELED" || strings.HasSuffix(dep.ReadyState, "_ERROR"):
			return false, errorf(d.Name(), "deployment %s: %s", strings.ToLower(dep.ReadyState), firstNonEmpty(dep.ErrorMessage, "no details"))
		case dep.ReadyState == "READY" && truthy(dep.AliasAssigned):
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	res := &Result{
		Provider:      d.Name(),
		DeployID:      dep.ID,
		AdminURL:      dep.InspectorURL,
		FilesTotal:    len(entries),
		FilesUploaded: uploaded,
		BytesUploaded: bytesUp,
	}
	if dep.URL != "" {
		res.DeployURL = "https://" + dep.URL
	}
	// The unique deploy URL may sit behind Deployment Protection; the
	// production alias is what visitors see.
	if len(dep.Alias) > 0 {
		res.URL = "https://" + dep.Alias[0]
	} else {
		res.URL = res.DeployURL
	}
	return res, nil
}

// createWithUploads posts the deployment, uploads whatever Vercel reports
// missing, and posts again. Vercel keeps previously uploaded files, so a
// redeploy only sends what changed.
func (d *VercelDeployer) createWithUploads(ctx context.Context, c *apiClient, project *vercelProject, entries []vercelFile, rep Reporter) (*vercelDeployment, int, int64, error) {
	uploaded := 0
	var bytesUp int64
	for attempt := 0; attempt < 2; attempt++ {
		dep, missing, err := d.create(ctx, c, project, entries)
		if err != nil {
			return nil, 0, 0, err
		}
		if dep != nil {
			return dep, uploaded, bytesUp, nil
		}
		if attempt == 1 {
			return nil, 0, 0, errorf(d.Name(), "deployment still reports %d missing files after upload", len(missing))
		}
		n, b, err := d.upload(ctx, c, entries, missing, rep)
		if err != nil {
			return nil, 0, 0, err
		}
		uploaded, bytesUp = n, b
		rep.Step(StepFinalize, "Creating the deployment")
	}
	return nil, 0, 0, errorf(d.Name(), "could not create the deployment")
}

// create posts the deployment. It returns the deployment, or the digests
// Vercel does not have yet (error code missing_files).
func (d *VercelDeployer) create(ctx context.Context, c *apiClient, project *vercelProject, entries []vercelFile) (*vercelDeployment, []string, error) {
	type fileRef struct {
		File string `json:"file"`
		SHA  string `json:"sha"`
		Size int64  `json:"size"`
	}
	refs := make([]fileRef, len(entries))
	for i, e := range entries {
		refs[i] = fileRef{File: e.Name, SHA: e.SHA, Size: e.Size}
	}
	q := d.query()
	q.Set("skipAutoDetectionConfirmation", "1")
	q.Set("prebuilt", "1")
	var dep vercelDeployment
	err := c.do(ctx, apiRequest{
		Method:   "POST",
		Path:     "/v13/deployments",
		Endpoint: "POST /v13/deployments",
		Query:    q,
		Body: jsonBody(map[string]any{
			"name":    project.Name,
			"project": project.ID,
			"target":  "production",
			"files":   refs,
			"meta":    map[string]string{"sardeVersion": version.Version},
		}),
		ContentType: "application/json",
	}, &dep)
	if err == nil {
		return &dep, nil, nil
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.Code == "missing_files" {
		var body struct {
			Error struct {
				Missing []string `json:"missing"`
			} `json:"error"`
		}
		if json.Unmarshal(apiErr.Body, &body) == nil && len(body.Error.Missing) > 0 {
			return nil, body.Error.Missing, nil
		}
	}
	return nil, nil, err
}

func (d *VercelDeployer) upload(ctx context.Context, c *apiClient, entries []vercelFile, missing []string, rep Reporter) (int, int64, error) {
	bySHA := make(map[string]vercelFile, len(entries))
	for _, e := range entries {
		bySHA[e.SHA] = e
	}
	var todo []vercelFile
	var bytesTotal int64
	for _, sha := range missing {
		e, ok := bySHA[sha]
		if !ok {
			return 0, 0, errorf(d.Name(), "deployment asked for unknown file %s", sha)
		}
		todo = append(todo, e)
		bytesTotal += e.Size
	}
	rep.Step(StepUpload, fmt.Sprintf("Uploading %d of %d files", len(todo), len(entries)))
	rep.Progress(Progress{Step: StepUpload, Total: len(todo), BytesTotal: bytesTotal})
	var done, sent atomic.Int64
	err := uploadAll(ctx, len(todo), vercelUploads, func(ctx context.Context, i int) error {
		e := todo[i]
		err := c.do(ctx, apiRequest{
			Method:      "POST",
			Path:        "/v2/files",
			Endpoint:    "POST /v2/files",
			Query:       d.query(),
			Header:      map[string][]string{"x-vercel-digest": {e.SHA}},
			ContentType: "application/octet-stream",
			Body:        e.Body,
			Timeout:     uploadTimeout,
		}, nil)
		if err != nil {
			return err
		}
		rep.Progress(Progress{Step: StepUpload, Done: int(done.Add(1)), Total: len(todo), Bytes: sent.Add(e.Size), BytesTotal: bytesTotal})
		return nil
	})
	return len(todo), bytesTotal, err
}

// buildVercelConfig renders .vercel/output/config.json: the redirects from
// the build's vercel.json as routes, then the filesystem, then a 404 page
// when the site has one.
func buildVercelConfig(redirects *File, has404 bool) ([]byte, error) {
	routes := []map[string]any{}
	if redirects != nil {
		raw, err := os.ReadFile(redirects.Abs)
		if err != nil {
			return nil, fmt.Errorf("reading vercel.json: %w", err)
		}
		var vj struct {
			Redirects []struct {
				Source      string `json:"source"`
				Destination string `json:"destination"`
				Permanent   *bool  `json:"permanent"`
				StatusCode  int    `json:"statusCode"`
			} `json:"redirects"`
		}
		if err := json.Unmarshal(raw, &vj); err != nil {
			return nil, configErrorf("vercel: the build's vercel.json is not valid JSON: %v", err)
		}
		for _, r := range vj.Redirects {
			if r.Source == "" || r.Destination == "" {
				continue
			}
			status := 307
			switch {
			case r.StatusCode != 0:
				status = r.StatusCode
			case r.Permanent == nil || *r.Permanent:
				status = 308
			}
			routes = append(routes, map[string]any{
				"src":     redirectSourceRegex(r.Source),
				"status":  status,
				"headers": map[string]string{"Location": r.Destination},
			})
		}
	}
	routes = append(routes, map[string]any{"handle": "filesystem"})
	if has404 {
		routes = append(routes,
			map[string]any{"handle": "error"},
			map[string]any{"src": "^/.*$", "status": 404, "dest": "/404.html"},
		)
	}
	return json.MarshalIndent(map[string]any{"version": 3, "routes": routes}, "", "  ")
}

// redirectSourceRegex turns a literal redirect source path into a Build
// Output route pattern that matches it with or without a trailing slash.
func redirectSourceRegex(source string) string {
	trimmed := strings.TrimRight(source, "/")
	if trimmed == "" {
		return "^/$"
	}
	return "^" + regexp.QuoteMeta(trimmed) + "/?$"
}

// truthy reports whether a JSON value is set: true, a non-zero number or a
// non-empty string (Vercel has sent aliasAssigned as each over time).
func truthy(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	switch s {
	case "", "null", "false", "0", `""`:
		return false
	}
	return true
}
