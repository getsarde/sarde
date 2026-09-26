package deploy

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/getsarde/sarde/internal/version"
)

// NetlifyDeployer publishes through Netlify's file-digest deploy API: send a
// manifest of SHA1 digests, upload only the files Netlify does not already
// have, then wait for the deploy to go live. Credentials: the
// NETLIFY_AUTH_TOKEN environment variable and deploy.site_id.
//
// API reference: https://docs.netlify.com/api-and-cli-guides/api-guides/get-started-with-api/
type NetlifyDeployer struct {
	SiteID string
	opts   Options
}

const (
	netlifyAPI           = "https://api.netlify.com/api/v1"
	netlifyUploads       = 8
	netlifyDiffTimeout   = 5 * time.Minute
	netlifyReadyTimeout  = 20 * time.Minute
	netlifyPollInterval  = 2 * time.Second
	netlifyCancelTimeout = 5 * time.Second
)

func (d *NetlifyDeployer) Name() string { return "netlify" }

type netlifySite struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	SSLURL          string `json:"ssl_url"`
	URL             string `json:"url"`
	AdminURL        string `json:"admin_url"`
	AccountName     string `json:"account_name"`
	PublishedDeploy *struct {
		ID string `json:"id"`
	} `json:"published_deploy"`
}

type netlifyDeploy struct {
	ID           string   `json:"id"`
	State        string   `json:"state"`
	Required     []string `json:"required"`
	ErrorMessage string   `json:"error_message"`
	SSLURL       string   `json:"ssl_url"`
	DeploySSLURL string   `json:"deploy_ssl_url"`
	AdminURL     string   `json:"admin_url"`
}

func (d *NetlifyDeployer) client(rep Reporter) (*apiClient, error) {
	token, err := d.opts.requireEnv("NETLIFY_AUTH_TOKEN", "Netlify deploy")
	if err != nil {
		return nil, err
	}
	if d.SiteID == "" {
		return nil, configErrorf("netlify deploy requires deploy.site_id in sarde.yaml")
	}
	return newAPIClient(d.Name(), netlifyAPI, token, d.opts, rep), nil
}

func (d *NetlifyDeployer) site(ctx context.Context, c *apiClient) (*netlifySite, error) {
	var site netlifySite
	err := c.do(ctx, apiRequest{
		Method:   "GET",
		Path:     "/sites/" + url.PathEscape(d.SiteID),
		Endpoint: "GET /sites/{site_id}",
	}, &site)
	if err != nil {
		return nil, err
	}
	return &site, nil
}

// Check verifies the token can read the configured site.
func (d *NetlifyDeployer) Check(ctx context.Context, rep Reporter) (*CheckResult, error) {
	rep = orNop(rep)
	c, err := d.client(rep)
	if err != nil {
		return nil, err
	}
	rep.Step(StepPrepare, "Looking up the Netlify site")
	site, err := d.site(ctx, c)
	if err != nil {
		return nil, err
	}
	return &CheckResult{
		Provider: d.Name(),
		TargetID: site.ID,
		Target:   site.Name,
		URL:      firstNonEmpty(site.SSLURL, site.URL),
		Account:  site.AccountName,
	}, nil
}

func (d *NetlifyDeployer) Deploy(ctx context.Context, distDir string, rep Reporter) (*Result, error) {
	rep = orNop(rep)
	c, err := d.client(rep)
	if err != nil {
		return nil, err
	}

	rep.Step(StepPrepare, "Looking up the Netlify site")
	site, err := d.site(ctx, c)
	if err != nil {
		return nil, err
	}

	rep.Step(StepCollect, "Collecting files")
	files, err := collectFiles(distDir, rep)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		// Netlify cannot address these characters in a file path.
		if strings.ContainsAny(f.Rel, "#?") {
			return nil, configErrorf("netlify: file name %q contains # or ?, which Netlify cannot serve", f.Rel)
		}
	}

	rep.Step(StepHash, fmt.Sprintf("Hashing %d files", len(files)))
	sums, err := hashAll(ctx, files, func(f File) (string, error) { return sha1File(f.Abs) }, rep)
	if err != nil {
		return nil, err
	}
	manifest := make(map[string]string, len(files))
	bySum := make(map[string][]int)
	for i, f := range files {
		manifest["/"+f.Rel] = sums[i]
		bySum[sums[i]] = append(bySum[sums[i]], i)
	}

	rep.Step(StepPrepare, "Creating the deploy")
	var dep netlifyDeploy
	err = c.do(ctx, apiRequest{
		Method:   "POST",
		Path:     "/sites/" + url.PathEscape(site.ID) + "/deploys",
		Endpoint: "POST /sites/{site_id}/deploys",
		Query:    url.Values{"title": {"Deployed with Sarde " + version.Version}},
		Body: jsonBody(map[string]any{
			"files": manifest,
			"draft": false,
			"async": true,
		}),
		ContentType: "application/json",
	}, &dep)
	if err != nil {
		return nil, err
	}
	deployID := dep.ID

	// From here on a cancelled deploy is cancelled on Netlify's side too, so
	// it never lingers in "uploading".
	published := false
	defer func() {
		if published || ctx.Err() == nil {
			return
		}
		cctx, cancel := context.WithTimeout(context.Background(), netlifyCancelTimeout)
		defer cancel()
		_ = c.do(cctx, apiRequest{
			Method:   "POST",
			Path:     "/deploys/" + url.PathEscape(deployID) + "/cancel",
			Endpoint: "POST /deploys/{id}/cancel",
		}, nil)
	}()

	// An async deploy computes its required list in the background.
	rep.Step(StepPrepare, "Waiting for Netlify to compare files")
	err = poll(ctx, "netlify to compare files", netlifyPollInterval, netlifyDiffTimeout, func(ctx context.Context) (bool, error) {
		if err := d.getDeploy(ctx, c, deployID, &dep); err != nil {
			return false, err
		}
		switch dep.State {
		case "error":
			return false, errorf(d.Name(), "deploy failed: %s", firstNonEmpty(dep.ErrorMessage, "unknown error"))
		case "prepared", "uploading", "uploaded", "ready":
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	// Every file carrying a required digest is uploaded, as netlify-cli does.
	var uploads []File
	for _, sum := range dep.Required {
		for _, i := range bySum[sum] {
			uploads = append(uploads, files[i])
		}
	}
	bytesTotal := totalSize(uploads)
	rep.Step(StepUpload, fmt.Sprintf("Uploading %d of %d files", len(uploads), len(files)))
	var done atomic.Int64
	var sent atomic.Int64
	rep.Progress(Progress{Step: StepUpload, Done: 0, Total: len(uploads), BytesTotal: bytesTotal})
	err = uploadAll(ctx, len(uploads), netlifyUploads, func(ctx context.Context, i int) error {
		f := uploads[i]
		err := c.do(ctx, apiRequest{
			Method:      "PUT",
			Path:        "/deploys/" + url.PathEscape(deployID) + "/files/" + escapePath(f.Rel),
			Endpoint:    "PUT /deploys/{id}/files/{path}",
			Body:        fileBody(f.Abs),
			ContentType: "application/octet-stream",
			Timeout:     uploadTimeout,
		}, nil)
		if err != nil {
			return err
		}
		rep.Progress(Progress{
			Step:       StepUpload,
			Done:       int(done.Add(1)),
			Total:      len(uploads),
			Bytes:      sent.Add(f.Size),
			BytesTotal: bytesTotal,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	rep.Step(StepWait, "Waiting for Netlify to publish")
	err = poll(ctx, "netlify to publish", netlifyPollInterval, netlifyReadyTimeout, func(ctx context.Context) (bool, error) {
		if err := d.getDeploy(ctx, c, deployID, &dep); err != nil {
			return false, err
		}
		switch dep.State {
		case "error":
			return false, errorf(d.Name(), "deploy failed: %s", firstNonEmpty(dep.ErrorMessage, "unknown error"))
		case "ready":
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	published = true

	// A site with auto publishing locked keeps serving the previous deploy.
	if after, err := d.site(ctx, c); err == nil && after.PublishedDeploy != nil && after.PublishedDeploy.ID != deployID {
		rep.Log(LevelWarn, "Netlify built the deploy but did not publish it (auto publishing is locked); publish it from the Netlify dashboard")
	}

	return &Result{
		Provider:      d.Name(),
		URL:           firstNonEmpty(site.SSLURL, dep.SSLURL, site.URL),
		DeployURL:     dep.DeploySSLURL,
		DeployID:      deployID,
		AdminURL:      firstNonEmpty(dep.AdminURL, site.AdminURL),
		FilesTotal:    len(files),
		FilesUploaded: len(uploads),
		BytesUploaded: bytesTotal,
	}, nil
}

func (d *NetlifyDeployer) getDeploy(ctx context.Context, c *apiClient, id string, out *netlifyDeploy) error {
	return c.do(ctx, apiRequest{
		Method:   "GET",
		Path:     "/deploys/" + url.PathEscape(id),
		Endpoint: "GET /deploys/{id}",
	}, out)
}

// escapePath escapes each segment of a forward-slash path for a URL.
func escapePath(rel string) string {
	parts := strings.Split(rel, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
