package deploy

import (
	"context"
	"fmt"
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
		d, err := NewDeployer(cfg, Options{})
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
	_, err := NewDeployer(config.DeployConfig{}, Options{})
	if err == nil {
		t.Error("expected error for empty provider")
	}
}

func TestNewDeployer_UnknownProvider(t *testing.T) {
	_, err := NewDeployer(config.DeployConfig{Provider: "unknown"}, Options{})
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestNewDeployer_CustomRequiresCommand(t *testing.T) {
	_, err := NewDeployer(config.DeployConfig{Provider: "custom"}, Options{})
	if err == nil {
		t.Error("expected error for custom without command")
	}
}

func TestNewDeployer_GitHubDefaultBranch(t *testing.T) {
	d, err := NewDeployer(config.DeployConfig{Provider: "github"}, Options{})
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
	rep := &recordingReporter{}
	res, err := d.Deploy(context.Background(), distDir, rep)
	if err != nil {
		t.Fatalf("Deploy() error: %v", err)
	}
	if res.Provider != "custom" {
		t.Errorf("result provider = %q", res.Provider)
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

// A custom command that quotes a path with a space must reach the shell as
// typed, from a verbatim (`\\?\`) project directory. On Windows, Go's default
// quoting escaped the inner quotes for cmd.exe ("The filename, directory
// name, or volume label syntax is incorrect"), and cmd.exe refused the
// verbatim working directory; both found by Sarde Studio's E2E deploy test.
func TestCustomDeployer_QuotedPathWithSpace(t *testing.T) {
	distDir := t.TempDir()
	outDir := filepath.Join(t.TempDir(), "has space")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(outDir, "result.txt")
	projectDir := t.TempDir()
	var command string
	if runtime.GOOS == "windows" {
		command = `echo %DIST_DIR%> "` + outFile + `"`
		projectDir = `\\?\` + projectDir
	} else {
		command = `echo "$DIST_DIR" > "` + outFile + `"`
	}
	d := &CustomDeployer{Command: command, opts: Options{ProjectDir: projectDir}}
	if _, err := d.Deploy(context.Background(), distDir, &recordingReporter{}); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("the command did not write %s: %v", outFile, err)
	}
	if !strings.Contains(string(data), filepath.Base(distDir)) {
		t.Errorf("DIST_DIR not passed, got %q", data)
	}
}

func TestAPIDeployers_RequireCredentials(t *testing.T) {
	dist := writeDist(t, map[string]string{"index.html": "x"})
	cases := []struct {
		name string
		d    Deployer
		want string
	}{
		{"netlify token", &NetlifyDeployer{SiteID: "site", opts: Options{Getenv: fakeEnv(nil)}}, "NETLIFY_AUTH_TOKEN"},
		{"netlify site", &NetlifyDeployer{opts: Options{Getenv: fakeEnv(map[string]string{"NETLIFY_AUTH_TOKEN": "t"})}}, "deploy.site_id"},
		{"vercel token", &VercelDeployer{ProjectID: "p", opts: Options{Getenv: fakeEnv(nil)}}, "VERCEL_TOKEN"},
		{"vercel project", &VercelDeployer{opts: Options{Getenv: fakeEnv(map[string]string{"VERCEL_TOKEN": "t"})}}, "deploy.project_id"},
		{"cloudflare token", &CloudflareDeployer{ProjectName: "p", AccountID: "a", opts: Options{Getenv: fakeEnv(nil)}}, "CLOUDFLARE_API_TOKEN"},
		{"cloudflare account", &CloudflareDeployer{ProjectName: "p", opts: Options{Getenv: fakeEnv(map[string]string{"CLOUDFLARE_API_TOKEN": "t"})}}, "account_id"},
		{"cloudflare project", &CloudflareDeployer{AccountID: "a", opts: Options{Getenv: fakeEnv(map[string]string{"CLOUDFLARE_API_TOKEN": "t"})}}, "deploy.project_name"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.d.Deploy(context.Background(), dist, nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want mention of %s", err, tc.want)
			}
			if ErrorCode(err) != CodeConfig {
				t.Errorf("ErrorCode = %q, want config", ErrorCode(err))
			}
		})
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

func TestErrorCode(t *testing.T) {
	cases := map[string]error{
		CodeAuth:        &APIError{Status: 401},
		CodeNotFound:    &APIError{Status: 404},
		CodeRateLimited: &APIError{Status: 429},
		CodeProvider:    &APIError{Status: 500},
		CodeCanceled:    context.Canceled,
		CodeConfig:      configErrorf("x"),
		CodeLimit:       limitErrorf("x"),
	}
	for want, err := range cases {
		if got := ErrorCode(fmt.Errorf("wrapped: %w", err)); got != want {
			t.Errorf("ErrorCode(%v) = %q, want %q", err, got, want)
		}
	}
	if ErrorCode(nil) != "" {
		t.Error("ErrorCode(nil) should be empty")
	}
}
