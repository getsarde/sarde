// Package deploy publishes a built site to a hosting provider.
//
// Every deployer takes its credentials from environment variables, never
// from sarde.yaml, and reports progress through a Reporter instead of
// printing, so the CLI can render human lines or a JSON event stream.
package deploy

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/getsarde/sarde/internal/config"
)

// Deployer publishes a built output directory.
type Deployer interface {
	Name() string
	Deploy(ctx context.Context, distDir string, rep Reporter) (*Result, error)
}

// Checker is implemented by deployers that can verify credentials and access
// to the configured target without uploading anything.
type Checker interface {
	Check(ctx context.Context, rep Reporter) (*CheckResult, error)
}

// Result describes a finished deploy.
type Result struct {
	Provider      string `json:"provider"`
	URL           string `json:"url,omitempty"`        // live production URL
	DeployURL     string `json:"deploy_url,omitempty"` // permalink of this deploy
	DeployID      string `json:"deploy_id,omitempty"`
	AdminURL      string `json:"admin_url,omitempty"` // provider dashboard for this deploy
	FilesTotal    int    `json:"files_total"`
	FilesUploaded int    `json:"files_uploaded"`
	BytesUploaded int64  `json:"bytes_uploaded"`
}

// CheckResult describes the target a credential check reached.
type CheckResult struct {
	Provider string `json:"provider"`
	TargetID string `json:"target_id,omitempty"`
	Target   string `json:"target,omitempty"` // site or project name
	URL      string `json:"url,omitempty"`    // the target's production URL
	Account  string `json:"account,omitempty"`
}

// Options carries everything a deployer needs that is not sarde.yaml config.
type Options struct {
	// ProjectDir is the site root. The GitHub deployer reads the origin
	// remote there; empty means the process working directory.
	ProjectDir string
	// Getenv reads credentials. Nil means os.Getenv; tests inject a map.
	Getenv func(string) string
	// HTTPClient is used by the API deployers. Nil means a default client.
	HTTPClient *http.Client
	// BaseURL overrides the provider API base URL (tests point it at an
	// httptest server). Empty means the provider's public API.
	BaseURL string
}

func (o Options) env(key string) string {
	if o.Getenv != nil {
		return o.Getenv(key)
	}
	return os.Getenv(key)
}

// requireEnv reads a required credential, failing with a config error that
// names the variable when it is missing.
func (o Options) requireEnv(key, label string) (string, error) {
	val := o.env(key)
	if val == "" {
		return "", configErrorf("%s requires the %s environment variable", label, key)
	}
	return val, nil
}

// NewDeployer creates the Deployer for cfg.Provider.
func NewDeployer(cfg config.DeployConfig, opts Options) (Deployer, error) {
	switch cfg.Provider {
	case "github":
		branch := cfg.Branch
		if branch == "" {
			branch = "gh-pages"
		}
		return &GitHubPagesDeployer{Branch: branch, CNAME: cfg.CNAME, opts: opts}, nil
	case "netlify":
		return &NetlifyDeployer{SiteID: cfg.SiteID, opts: opts}, nil
	case "cloudflare":
		return &CloudflareDeployer{ProjectName: cfg.ProjectName, AccountID: cfg.AccountID, opts: opts}, nil
	case "vercel":
		return &VercelDeployer{ProjectID: cfg.ProjectID, TeamID: cfg.TeamID, opts: opts}, nil
	case "custom":
		if cfg.Command == "" {
			return nil, configErrorf("custom deploy requires deploy.command in sarde.yaml")
		}
		return &CustomDeployer{Command: cfg.Command, opts: opts}, nil
	case "":
		return nil, configErrorf("no deploy provider configured; set deploy.provider in sarde.yaml")
	default:
		return nil, configErrorf("unknown deploy provider: %q", cfg.Provider)
	}
}

// maskToken returns a masked version of a token for logging.
func maskToken(token string) string {
	if len(token) <= 4 {
		return "***"
	}
	return "***" + token[len(token)-4:]
}

// errorf prefixes a message with the provider name, the shape every API
// deployer uses so a CLI error reads "netlify: ...".
func errorf(provider, format string, args ...any) error {
	return fmt.Errorf("%s: %s", provider, fmt.Sprintf(format, args...))
}
