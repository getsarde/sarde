package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRequireSite(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, dir string)
		content string // --content flag value
		wantErr bool
		want    []string // substrings of the error text
	}{
		{
			name:    "empty folder",
			wantErr: true,
			want:    []string{"not a Sarde site", "no sarde.yaml and no content/ folder", "sarde test <site-dir>", "sarde new site <name>"},
		},
		{
			name:  "zero-config site",
			setup: func(t *testing.T, dir string) { mkdir(t, filepath.Join(dir, "content")) },
		},
		{
			name:  "config file only",
			setup: func(t *testing.T, dir string) { writeFile(t, filepath.Join(dir, "sarde.yaml")) },
		},
		{
			name:    "explicit --content",
			content: "elsewhere",
		},
		{
			name:    "content is a file, not a folder",
			setup:   func(t *testing.T, dir string) { writeFile(t, filepath.Join(dir, "content")) },
			wantErr: true,
		},
		{
			name:    "site in a subfolder",
			setup:   func(t *testing.T, dir string) { writeFile(t, filepath.Join(dir, "vanier", "sarde.yaml")) },
			wantErr: true,
			want:    []string{`Found a site in the subfolder "vanier". Run:`, "    cd vanier\n", "    sarde test"},
		},
		{
			name: "sites in several subfolders",
			setup: func(t *testing.T, dir string) {
				writeFile(t, filepath.Join(dir, "b-site", "sarde.yaml"))
				writeFile(t, filepath.Join(dir, "a-site", "sarde.yaml"))
				mkdir(t, filepath.Join(dir, "not-a-site"))
			},
			wantErr: true,
			want:    []string{`Found sites in the subfolders "a-site", "b-site". Run:`, "cd a-site\n"},
		},
		{
			name:    "subfolder name with a space",
			setup:   func(t *testing.T, dir string) { writeFile(t, filepath.Join(dir, "my site", "sarde.yaml")) },
			wantErr: true,
			want:    []string{`cd "my site"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.setup != nil {
				tt.setup(t, dir)
			}
			cmd := newSiteCheckCmd(t, tt.content)

			err := requireSite(cmd, dir)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("requireSite: %v", err)
				}
				if cmd.SilenceErrors {
					t.Error("SilenceErrors set without an error")
				}
				return
			}

			var nse *notSiteError
			if !errors.As(err, &nse) {
				t.Fatalf("want *notSiteError, got %v", err)
			}
			if !cmd.SilenceErrors {
				t.Error("SilenceErrors not set: Cobra would print a second, plain copy")
			}
			msg := err.Error()
			for _, w := range tt.want {
				if !strings.Contains(msg, w) {
					t.Errorf("error text missing %q:\n%s", w, msg)
				}
			}
			if strings.Contains(msg, "\x1b") {
				t.Errorf("Error() must be plain text:\n%q", msg)
			}
			if pretty := nse.Pretty(); pretty != "Error: "+msg+"\n" {
				t.Errorf("Pretty() with color off = %q, want the plain text with an Error: prefix", pretty)
			}
		})
	}
}

// A build outside a site stops before config resolution and writes nothing.
func TestRunBuild_NotASite(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() { buildCmd.SilenceErrors = false })

	rootCmd.SetArgs([]string{"build", dir})
	err := rootCmd.Execute()

	var nse *notSiteError
	if !errors.As(err, &nse) {
		t.Fatalf("want *notSiteError, got %v", err)
	}
	if !strings.Contains(err.Error(), "sarde build <site-dir>") {
		t.Errorf("hint should name the build command:\n%s", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "dist")); statErr == nil {
		t.Error("dist/ was created for a folder that is not a site")
	}
}

// newSiteCheckCmd returns a "sarde test" command carrying the flags
// requireSite reads. --config is defined locally: a parent's persistent flags
// only merge into Flags() when Cobra executes the command.
func newSiteCheckCmd(t *testing.T, content string) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "sarde"}
	cmd := &cobra.Command{Use: "test", Run: func(*cobra.Command, []string) {}}
	cmd.Flags().String("config", "sarde.yaml", "")
	cmd.Flags().String("content", "", "")
	root.AddCommand(cmd)
	if content != "" {
		if err := cmd.Flags().Set("content", content); err != nil {
			t.Fatal(err)
		}
	}
	return cmd
}

func mkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	mkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
}
