package deploy

import (
	"fmt"
	"os"

	"github.com/getsarde/sarde/internal/config"
)

// Deployer is the interface for deploying a built site.
type Deployer interface {
	Deploy(distDir string) error
	Name() string
}

// NewDeployer creates a Deployer based on the deploy configuration.
func NewDeployer(cfg config.DeployConfig) (Deployer, error) {
	switch cfg.Provider {
	case "github":
		branch := cfg.Branch
		if branch == "" {
			branch = "gh-pages"
		}
		return &GitHubPagesDeployer{Branch: branch}, nil
	case "netlify":
		return &NetlifyDeployer{SiteID: cfg.SiteID}, nil
	case "cloudflare":
		return &CloudflareDeployer{ProjectName: cfg.ProjectName}, nil
	case "vercel":
		return &VercelDeployer{ProjectID: cfg.ProjectID}, nil
	case "custom":
		if cfg.Command == "" {
			return nil, fmt.Errorf("custom deploy requires a command")
		}
		return &CustomDeployer{Command: cfg.Command}, nil
	case "":
		return nil, fmt.Errorf("no deploy provider configured; set deploy.provider in sarde.yaml")
	default:
		return nil, fmt.Errorf("unknown deploy provider: %q", cfg.Provider)
	}
}

// requireEnv reads a required environment variable, returning an error if missing.
func requireEnv(key, label string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("%s requires %s environment variable", label, key)
	}
	return val, nil
}

// maskToken returns a masked version of a token for logging.
func maskToken(token string) string {
	if len(token) <= 4 {
		return "***"
	}
	return "***" + token[len(token)-4:]
}
