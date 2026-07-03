package deploy

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/config"
)

func TestNewDeployer_ValidProviders(t *testing.T) {
	tests := []struct {
		provider string
		wantName string
	}{
		{"github", "github-pages"},
		{"netlify", "netlify"},
		{"cloudflare", "cloudflare-pages"},
		{"vercel", "vercel"},
		{"custom", "custom"},
	}

	for _, tt := range tests {
		cfg := config.DeployConfig{Provider: tt.provider}
		if tt.provider == "custom" {
			cfg.Command = "echo deploy"
		}
		d, err := NewDeployer(cfg)
		if err != nil {
			t.Errorf("NewDeployer(%q) error: %v", tt.provider, err)
			continue
		}
		if d.Name() != tt.wantName {
			t.Errorf("NewDeployer(%q).Name() = %q, want %q", tt.provider, d.Name(), tt.wantName)
		}
	}
}

func TestNewDeployer_EmptyProvider(t *testing.T) {
	_, err := NewDeployer(config.DeployConfig{})
	if err == nil {
		t.Error("expected error for empty provider")
	}
}

func TestNewDeployer_UnknownProvider(t *testing.T) {
	_, err := NewDeployer(config.DeployConfig{Provider: "unknown"})
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestNewDeployer_CustomRequiresCommand(t *testing.T) {
	_, err := NewDeployer(config.DeployConfig{Provider: "custom"})
	if err == nil {
		t.Error("expected error for custom without command")
	}
}

func TestNewDeployer_GitHubDefaultBranch(t *testing.T) {
	d, err := NewDeployer(config.DeployConfig{Provider: "github"})
	if err != nil {
		t.Fatal(err)
	}
	gh := d.(*GitHubPagesDeployer)
	if gh.Branch != "gh-pages" {
		t.Errorf("default branch = %q, want %q", gh.Branch, "gh-pages")
	}
}

func TestCustomDeployer_Execute(t *testing.T) {
	distDir := t.TempDir()
	os.WriteFile(filepath.Join(distDir, "index.html"), []byte("<h1>test</h1>"), 0o644)

	// Create a script that writes DIST_DIR to a file.
	tmpOut := t.TempDir()
	outFile := filepath.Join(tmpOut, "result.txt")

	var command string
	if runtime.GOOS == "windows" {
		// Use filepath with forward slashes for cmd compatibility.
		outFileSlash := strings.ReplaceAll(outFile, "\\", "/")
		command = `echo %DIST_DIR%> ` + outFileSlash
	} else {
		command = `echo "$DIST_DIR" > "` + outFile + `"`
	}

	d := &CustomDeployer{Command: command}
	if err := d.Deploy(distDir); err != nil {
		t.Fatalf("Deploy() error: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading result: %v", err)
	}
	output := strings.TrimSpace(string(data))
	if !strings.Contains(output, filepath.Base(distDir)) {
		t.Errorf("DIST_DIR not passed correctly, got: %q", output)
	}
}

func TestNetlifyDeployer_RequiresToken(t *testing.T) {
	os.Unsetenv("NETLIFY_AUTH_TOKEN")
	d := &NetlifyDeployer{SiteID: "test-site"}
	err := d.Deploy("/tmp/dist")
	if err == nil {
		t.Error("expected error without NETLIFY_AUTH_TOKEN")
	}
}

func TestNetlifyDeployer_RequiresSiteID(t *testing.T) {
	t.Setenv("NETLIFY_AUTH_TOKEN", "test-token")
	d := &NetlifyDeployer{SiteID: ""}
	err := d.Deploy("/tmp/dist")
	if err == nil {
		t.Error("expected error without site_id")
	}
}

func TestCloudflareDeployer_RequiresEnv(t *testing.T) {
	os.Unsetenv("CLOUDFLARE_API_TOKEN")
	os.Unsetenv("CLOUDFLARE_ACCOUNT_ID")
	d := &CloudflareDeployer{ProjectName: "my-project"}
	err := d.Deploy("/tmp/dist")
	if err == nil {
		t.Error("expected error without CLOUDFLARE_API_TOKEN")
	}
}

func TestVercelDeployer_RequiresEnv(t *testing.T) {
	os.Unsetenv("VERCEL_TOKEN")
	d := &VercelDeployer{ProjectID: "my-project"}
	err := d.Deploy("/tmp/dist")
	if err == nil {
		t.Error("expected error without VERCEL_TOKEN")
	}
}

func TestMaskToken(t *testing.T) {
	if got := maskToken("abcdefgh"); got != "***efgh" {
		t.Errorf("maskToken = %q, want %q", got, "***efgh")
	}
	if got := maskToken("abc"); got != "***" {
		t.Errorf("maskToken(short) = %q, want %q", got, "***")
	}
}

func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	os.MkdirAll(filepath.Join(src, "sub"), 0o755)
	os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0o644)
	os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("world"), 0o644)

	dst := t.TempDir()
	if err := copyDir(src, dst); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dst, "a.txt"))
	if string(data) != "hello" {
		t.Errorf("a.txt = %q", data)
	}
	data, _ = os.ReadFile(filepath.Join(dst, "sub", "b.txt"))
	if string(data) != "world" {
		t.Errorf("sub/b.txt = %q", data)
	}
}

// The unimplemented API deployers must report an error (not nil), so a CI
// pipeline running `sarde deploy` fails loudly instead of exiting 0 without
// having deployed anything.
func TestUnimplementedDeployers_ReturnError(t *testing.T) {
	t.Setenv("NETLIFY_AUTH_TOKEN", "test-token")
	t.Setenv("VERCEL_TOKEN", "test-token")
	t.Setenv("CLOUDFLARE_API_TOKEN", "test-token")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "test-account")

	deployers := []Deployer{
		&NetlifyDeployer{SiteID: "site"},
		&VercelDeployer{ProjectID: "proj"},
		&CloudflareDeployer{ProjectName: "proj"},
	}
	for _, d := range deployers {
		if err := d.Deploy("/tmp/dist"); err == nil {
			t.Errorf("%s.Deploy returned nil; unimplemented deployers must error", d.Name())
		}
	}
}
