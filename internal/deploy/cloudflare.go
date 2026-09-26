package deploy

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/getsarde/sarde/internal/version"
)

// CloudflareDeployer publishes to Cloudflare Pages with direct upload: the
// same protocol `wrangler pages deploy` uses. Credentials: the
// CLOUDFLARE_API_TOKEN environment variable (Pages Write permission), the
// account id (deploy.account_id, or CLOUDFLARE_ACCOUNT_ID, which wins) and
// deploy.project_name.
//
// The project and deployment endpoints are documented
// (https://developers.cloudflare.com/api/resources/pages/). The asset upload
// endpoints are not: they are defined by wrangler's source
// (packages/wrangler/src/pages/upload.ts), so everything that touches them
// lives in cloudflare_upload.go and names the endpoint in its errors.
type CloudflareDeployer struct {
	ProjectName string
	AccountID   string
	opts        Options
}

const (
	cloudflareAPI           = "https://api.cloudflare.com/client/v4"
	cloudflareMaxAssetSize  = 25 << 20
	cloudflareDefaultMax    = 20000
	cloudflarePollAttempts  = 5
	cloudflareCreateRetries = 3
)

// cloudflareControlFiles are sent with the deployment request instead of as
// assets, as wrangler does.
var cloudflareControlFiles = map[string]bool{"_redirects": true, "_headers": true, "_routes.json": true}

func (d *CloudflareDeployer) Name() string { return "cloudflare-pages" }

type cfProject struct {
	Name             string   `json:"name"`
	Subdomain        string   `json:"subdomain"`
	Domains          []string `json:"domains"`
	ProductionBranch string   `json:"production_branch"`
}

type cfDeployment struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	LatestStage struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"latest_stage"`
}

// cfEnvelope is the Cloudflare v4 response wrapper.
type cfEnvelope struct {
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

func cloudflareErrorBody(body []byte) (string, string) {
	var env cfEnvelope
	if json.Unmarshal(body, &env) != nil || len(env.Errors) == 0 {
		return genericErrorBody(body)
	}
	return fmt.Sprint(env.Errors[0].Code), env.Errors[0].Message
}

func (d *CloudflareDeployer) accountID() string {
	return firstNonEmpty(d.opts.env("CLOUDFLARE_ACCOUNT_ID"), d.AccountID)
}

func (d *CloudflareDeployer) client(rep Reporter) (*apiClient, error) {
	token, err := d.opts.requireEnv("CLOUDFLARE_API_TOKEN", "Cloudflare deploy")
	if err != nil {
		return nil, err
	}
	if d.accountID() == "" {
		return nil, configErrorf("cloudflare deploy requires deploy.account_id in sarde.yaml or the CLOUDFLARE_ACCOUNT_ID environment variable")
	}
	if d.ProjectName == "" {
		return nil, configErrorf("cloudflare deploy requires deploy.project_name in sarde.yaml")
	}
	c := newAPIClient(d.Name(), cloudflareAPI, token, d.opts, rep)
	c.decodeError = cloudflareErrorBody
	return c, nil
}

func (d *CloudflareDeployer) projectPath() string {
	return "/accounts/" + url.PathEscape(d.accountID()) + "/pages/projects/" + url.PathEscape(d.ProjectName)
}

// call sends an account API request and decodes the envelope's result.
func (d *CloudflareDeployer) call(ctx context.Context, c *apiClient, req apiRequest, out any) error {
	var env cfEnvelope
	if err := c.do(ctx, req, &env); err != nil {
		return err
	}
	if !env.Success && len(env.Errors) > 0 {
		return &APIError{Provider: d.Name(), Endpoint: req.Endpoint, Status: 200,
			Code: fmt.Sprint(env.Errors[0].Code), Message: env.Errors[0].Message}
	}
	if out == nil || len(env.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(env.Result, out); err != nil {
		return fmt.Errorf("%s: %s: decoding result: %w", d.Name(), req.Endpoint, err)
	}
	return nil
}

func (d *CloudflareDeployer) project(ctx context.Context, c *apiClient) (*cfProject, error) {
	var p cfProject
	err := d.call(ctx, c, apiRequest{Method: "GET", Path: d.projectPath(),
		Endpoint: "GET /accounts/{account}/pages/projects/{project}"}, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (d *CloudflareDeployer) liveURL(p *cfProject) string {
	for _, dom := range p.Domains {
		if !strings.HasSuffix(dom, ".pages.dev") {
			return "https://" + dom
		}
	}
	if p.Subdomain != "" {
		return "https://" + p.Subdomain
	}
	return ""
}

// Check looks the project up and fetches an upload token: it proves both
// read and write access and uploads nothing.
func (d *CloudflareDeployer) Check(ctx context.Context, rep Reporter) (*CheckResult, error) {
	rep = orNop(rep)
	c, err := d.client(rep)
	if err != nil {
		return nil, err
	}
	rep.Step(StepPrepare, "Looking up the Cloudflare Pages project")
	p, err := d.project(ctx, c)
	if err != nil {
		return nil, err
	}
	if _, err := d.uploadToken(ctx, c); err != nil {
		return nil, err
	}
	return &CheckResult{Provider: d.Name(), TargetID: p.Name, Target: p.Name, URL: d.liveURL(p), Account: d.accountID()}, nil
}

func (d *CloudflareDeployer) Deploy(ctx context.Context, distDir string, rep Reporter) (*Result, error) {
	rep = orNop(rep)
	c, err := d.client(rep)
	if err != nil {
		return nil, err
	}

	rep.Step(StepPrepare, "Looking up the Cloudflare Pages project")
	project, err := d.project(ctx, c)
	if err != nil {
		return nil, err
	}

	rep.Step(StepCollect, "Collecting files")
	all, err := collectFiles(distDir, rep)
	if err != nil {
		return nil, err
	}
	var assets []File
	control := map[string]File{}
	for _, f := range all {
		switch {
		case f.Rel == "_worker.js" || strings.HasPrefix(f.Rel, "_worker.js/"):
			return nil, configErrorf("cloudflare: the output contains _worker.js; Pages Functions are not supported by sarde deploy, use provider: custom with wrangler")
		case cloudflareControlFiles[f.Rel]:
			control[f.Rel] = f
		case f.Rel == ".DS_Store" || strings.HasSuffix(f.Rel, "/.DS_Store"):
			// never published
		default:
			if f.Size > cloudflareMaxAssetSize {
				return nil, limitErrorf("cloudflare: %s is %d bytes; Pages accepts at most %d bytes per file", f.Rel, f.Size, cloudflareMaxAssetSize)
			}
			assets = append(assets, f)
		}
	}
	if _, ok := control["_routes.json"]; ok {
		rep.Log(LevelWarn, "_routes.json only matters with Pages Functions and is not uploaded")
	}

	up, err := newCloudflareUpload(ctx, d, c, rep)
	if err != nil {
		return nil, err
	}
	if len(assets) > up.maxFiles {
		return nil, limitErrorf("cloudflare: %d files exceeds this project's limit of %d", len(assets), up.maxFiles)
	}

	rep.Step(StepHash, fmt.Sprintf("Hashing %d files", len(assets)))
	hashes, err := hashAll(ctx, assets, cloudflareHashFile, rep)
	if err != nil {
		return nil, err
	}

	uploaded, bytesUp, err := up.uploadMissing(ctx, assets, hashes)
	if err != nil {
		return nil, err
	}

	manifest := make(map[string]string, len(assets))
	for i, f := range assets {
		manifest["/"+f.Rel] = hashes[i]
	}

	rep.Step(StepFinalize, "Creating the deployment")
	dep, err := d.createDeployment(ctx, c, project, manifest, control)
	if err != nil {
		return nil, err
	}

	rep.Step(StepWait, "Waiting for Cloudflare to publish")
	if err := d.waitForDeploy(ctx, c, dep, rep); err != nil {
		return nil, err
	}

	return &Result{
		Provider:      d.Name(),
		URL:           firstNonEmpty(d.liveURL(project), dep.URL),
		DeployURL:     dep.URL,
		DeployID:      dep.ID,
		AdminURL:      fmt.Sprintf("https://dash.cloudflare.com/%s/pages/view/%s/%s", d.accountID(), d.ProjectName, dep.ID),
		FilesTotal:    len(assets),
		FilesUploaded: uploaded,
		BytesUploaded: bytesUp,
	}, nil
}

func (d *CloudflareDeployer) createDeployment(ctx context.Context, c *apiClient, project *cfProject, manifest map[string]string, control map[string]File) (*cfDeployment, error) {
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("manifest", string(manifestJSON))
	if project.ProductionBranch != "" {
		_ = mw.WriteField("branch", project.ProductionBranch)
	}
	_ = mw.WriteField("commit_message", "Deployed with Sarde "+version.Version)
	for _, name := range []string{"_redirects", "_headers"} {
		f, ok := control[name]
		if !ok {
			continue
		}
		data, err := os.ReadFile(f.Abs)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", name, err)
		}
		part, err := mw.CreateFormFile(name, name)
		if err != nil {
			return nil, err
		}
		if _, err := part.Write(data); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	body := buf.Bytes()

	var dep cfDeployment
	var lastErr error
	for attempt := 0; attempt < cloudflareCreateRetries; attempt++ {
		lastErr = d.call(ctx, c, apiRequest{
			Method:      "POST",
			Path:        d.projectPath() + "/deployments",
			Endpoint:    "POST /accounts/{account}/pages/projects/{project}/deployments",
			ContentType: mw.FormDataContentType(),
			Body:        bytesBody(body),
			Timeout:     uploadTimeout,
		}, &dep)
		if lastErr == nil {
			return &dep, nil
		}
		// 8000000 is Cloudflare's generic "try again" for this endpoint.
		var apiErr *APIError
		if !errors.As(lastErr, &apiErr) || apiErr.Code != "8000000" {
			return nil, lastErr
		}
		if err := apiSleep(ctx, backoff(attempt+1, nil)); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

// waitForDeploy polls the deployment's stage with backoff. Like wrangler, it
// gives up quietly after a few attempts and reports the status as unknown
// rather than failing a deploy that may well have gone live.
func (d *CloudflareDeployer) waitForDeploy(ctx context.Context, c *apiClient, dep *cfDeployment, rep Reporter) error {
	path := d.projectPath() + "/deployments/" + url.PathEscape(dep.ID)
	for attempt := 1; attempt <= cloudflarePollAttempts; attempt++ {
		var cur cfDeployment
		err := d.call(ctx, c, apiRequest{Method: "GET", Path: path,
			Endpoint: "GET /accounts/{account}/pages/projects/{project}/deployments/{id}"}, &cur)
		if err != nil {
			return err
		}
		if cur.URL != "" {
			dep.URL = cur.URL
		}
		switch {
		case cur.LatestStage.Name == "deploy" && cur.LatestStage.Status == "success":
			return nil
		case cur.LatestStage.Status == "failure":
			return errorf(d.Name(), "deployment failed at stage %s: %s", cur.LatestStage.Name, d.lastLogLine(ctx, c, path))
		}
		if err := apiSleep(ctx, time.Second<<(attempt-1)); err != nil {
			return err
		}
	}
	rep.Log(LevelWarn, "Cloudflare accepted the deployment but its status is still pending; check the dashboard")
	return nil
}

func (d *CloudflareDeployer) lastLogLine(ctx context.Context, c *apiClient, depPath string) string {
	var logs struct {
		Data []struct {
			Line string `json:"line"`
		} `json:"data"`
	}
	err := d.call(ctx, c, apiRequest{Method: "GET", Path: depPath + "/history/logs",
		Endpoint: "GET .../deployments/{id}/history/logs", Query: url.Values{"size": {"10000000"}}}, &logs)
	if err != nil || len(logs.Data) == 0 {
		return "no log output"
	}
	return logs.Data[len(logs.Data)-1].Line
}

// uploadToken fetches the short-lived JWT the asset endpoints require.
func (d *CloudflareDeployer) uploadToken(ctx context.Context, c *apiClient) (string, error) {
	var res struct {
		JWT string `json:"jwt"`
	}
	err := d.call(ctx, c, apiRequest{Method: "GET", Path: d.projectPath() + "/upload-token",
		Endpoint: "GET /accounts/{account}/pages/projects/{project}/upload-token"}, &res)
	if err != nil {
		return "", err
	}
	if res.JWT == "" {
		return "", errorf(d.Name(), "upload-token returned no token")
	}
	return res.JWT, nil
}

// jwtMaxFiles reads the max_file_count_allowed claim from an upload token.
// The token is not verified: the claim only sets our own pre-flight limit.
func jwtMaxFiles(jwt string) int {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return cloudflareDefaultMax
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return cloudflareDefaultMax
	}
	var claims struct {
		MaxFiles int `json:"max_file_count_allowed"`
	}
	if json.Unmarshal(payload, &claims) != nil || claims.MaxFiles <= 0 {
		return cloudflareDefaultMax
	}
	return claims.MaxFiles
}
