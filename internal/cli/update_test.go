package cli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"syscall"
	"testing"

	"github.com/getsarde/sarde/internal/updatesign"
	"github.com/getsarde/sarde/internal/version"
)

func TestDetectPackageManager(t *testing.T) {
	tests := []struct {
		name string
		path string
		want packageManager
	}{
		{"apple silicon cellar", "/opt/homebrew/Cellar/sarde/1.2.3/bin/sarde", pmHomebrew},
		{"apple silicon symlink dir", "/opt/homebrew/bin/sarde", pmHomebrew},
		{"intel cellar (resolved symlink target)", "/usr/local/Cellar/sarde/1.2.3/bin/sarde", pmHomebrew},
		{"linuxbrew", "/home/linuxbrew/.linuxbrew/Cellar/sarde/1.2.3/bin/sarde", pmHomebrew},
		{"scoop", `C:\Users\user\scoop\apps\sarde\current\sarde.exe`, pmScoop},
		{"scoop mixed case", `C:\Users\user\Scoop\Apps\sarde\current\sarde.exe`, pmScoop},
		{"chocolatey", `C:\ProgramData\chocolatey\bin\sarde.exe`, pmChocolatey},
		{"winget", `C:\Users\user\AppData\Local\Microsoft\WinGet\Packages\getsarde.sarde\sarde.exe`, pmWinget},
		{"system usr bin", "/usr/bin/sarde", pmSystem},
		{"system usr lib", "/usr/lib/sarde/sarde", pmSystem},
		{"plain usr local bin", "/usr/local/bin/sarde", pmNone},
		{"home dir", "/home/user/bin/sarde", pmNone},
		{"windows user dir", `C:\Users\user\bin\sarde.exe`, pmNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectPackageManager(tt.path); got != tt.want {
				t.Errorf("detectPackageManager(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestUpgradeHint(t *testing.T) {
	for _, pm := range []packageManager{pmHomebrew, pmScoop, pmChocolatey, pmWinget, pmSystem} {
		if pm.upgradeHint() == "" {
			t.Errorf("upgradeHint() empty for %v", pm)
		}
	}
	if hint := pmNone.upgradeHint(); hint != "" {
		t.Errorf("upgradeHint() for pmNone = %q, want empty", hint)
	}
}

func TestTruncateNotes(t *testing.T) {
	tests := []struct {
		name     string
		notes    string
		maxLines int
		want     string
	}{
		{"empty", "", 10, ""},
		{"whitespace only", "  \n\t\n", 10, ""},
		{"short stays intact", "line1\nline2", 10, "line1\nline2"},
		{"trailing newlines trimmed", "line1\nline2\n\n", 10, "line1\nline2"},
		{"long gets cut with marker", "a\nb\nc\nd", 2, "a\nb\n  ..."},
		{"exact limit not cut", "a\nb", 2, "a\nb"},
		{"multibyte preserved", "héllo wörld 🎉\nsecond\nthird", 1, "héllo wörld 🎉\n  ..."},
		{"zero max lines", "line1", 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncateNotes(tt.notes, tt.maxLines); got != tt.want {
				t.Errorf("truncateNotes(%q, %d) = %q, want %q", tt.notes, tt.maxLines, got, tt.want)
			}
		})
	}
}

func TestPermissionHint(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantHint bool
	}{
		{"nil error", nil, false},
		{"eacces path error", &fs.PathError{Op: "rename", Path: "/usr/local/bin/sarde", Err: syscall.EACCES}, true},
		{"wrapped permission error", errors.New("boom"), false},
		{"direct permission error", fs.ErrPermission, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, got := permissionHint(tt.err)
			if got != tt.wantHint {
				t.Fatalf("permissionHint(%v) ok = %v, want %v", tt.err, got, tt.wantHint)
			}
			if got && msg == "" {
				t.Errorf("permissionHint(%v) returned empty message", tt.err)
			}
		})
	}
}

func TestReleaseValidator(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating keypair: %v", err)
	}
	origKeys := updatesign.PublicKeys
	updatesign.PublicKeys = []ed25519.PublicKey{pub}
	t.Cleanup(func() { updatesign.PublicKeys = origKeys })

	asset := []byte("fake release archive bytes")
	assetName := "sarde_linux_amd64.tar.gz"
	sum := sha256.Sum256(asset)
	checksums := []byte(fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), assetName))
	sig := updatesign.Sign(checksums, priv)

	v := newReleaseValidator()

	t.Run("checksums file resolves to sig asset", func(t *testing.T) {
		if got := v.GetValidationAssetName(checksumsFilename); got != checksumsFilename+".sig" {
			t.Errorf("GetValidationAssetName(%q) = %q, want %q", checksumsFilename, got, checksumsFilename+".sig")
		}
	})

	t.Run("asset resolves to checksums file", func(t *testing.T) {
		if got := v.GetValidationAssetName(assetName); got != checksumsFilename {
			t.Errorf("GetValidationAssetName(%q) = %q, want %q", assetName, got, checksumsFilename)
		}
	})

	t.Run("valid signature accepted", func(t *testing.T) {
		if err := v.Validate(checksumsFilename, checksums, sig); err != nil {
			t.Errorf("Validate(checksums) with valid signature failed: %v", err)
		}
	})

	t.Run("tampered checksums rejected", func(t *testing.T) {
		bad := append([]byte("evilhash  sarde_linux_amd64.tar.gz\n"), checksums...)
		if err := v.Validate(checksumsFilename, bad, sig); err == nil {
			t.Error("Validate(tampered checksums) succeeded, want error")
		}
	})

	t.Run("valid asset checksum accepted", func(t *testing.T) {
		if err := v.Validate(assetName, asset, checksums); err != nil {
			t.Errorf("Validate(asset) against checksums failed: %v", err)
		}
	})

	t.Run("tampered asset rejected", func(t *testing.T) {
		if err := v.Validate(assetName, []byte("tampered"), checksums); err == nil {
			t.Error("Validate(tampered asset) succeeded, want error")
		}
	})
}

func TestConfirmUpdate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		yes     bool
		isTTY   bool
		want    bool
		wantErr bool
	}{
		{"yes flag bypasses prompt on tty", "", true, true, true, false},
		{"yes flag bypasses prompt on non-tty", "", true, false, true, false},
		{"non-tty without yes errors", "y\n", false, false, false, true},
		{"answer y", "y\n", false, true, true, false},
		{"answer Y", "Y\n", false, true, true, false},
		{"answer yes", "yes\n", false, true, true, false},
		{"answer n", "n\n", false, true, false, false},
		{"empty answer declines", "\n", false, true, false, false},
		{"eof declines on tty", "", false, true, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			got, err := confirmUpdate(strings.NewReader(tt.input), &out, "1.2.3", tt.yes, tt.isTTY)
			if (err != nil) != tt.wantErr {
				t.Fatalf("confirmUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("confirmUpdate() = %v, want %v", got, tt.want)
			}
			if tt.yes && out.Len() > 0 {
				t.Errorf("confirmUpdate() with yes=true wrote a prompt: %q", out.String())
			}
			if tt.wantErr && out.Len() > 0 {
				t.Errorf("confirmUpdate() on non-tty error wrote a prompt: %q", out.String())
			}
		})
	}
}

func TestReleaseIsNewer(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{"newer", "1.2.3", "1.3.0", true},
		{"equal", "1.2.3", "1.2.3", false},
		{"older", "1.2.3", "1.1.0", false},
		{"unparsable current never blocks", "garbage", "1.3.0", true},
		{"unparsable latest never triggers", "1.2.3", "garbage", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := releaseIsNewer(tt.current, tt.latest); got != tt.want {
				t.Errorf("releaseIsNewer(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

// fakeUpdater drives runUpdate without any network access.
type fakeUpdater struct {
	release     *updateRelease
	found       bool
	detectErr   error
	applyErr    error
	applyCalled int
}

func (f *fakeUpdater) detectLatest(context.Context) (*updateRelease, bool, error) {
	return f.release, f.found, f.detectErr
}

func (f *fakeUpdater) updateTo(_ context.Context, _ *updateRelease, _ string) error {
	f.applyCalled++
	return f.applyErr
}

// withUpdateEnv installs a fake updater and a release-style version for the
// duration of one test, restoring both plus the command flags afterward.
func withUpdateEnv(t *testing.T, fake *fakeUpdater, currentVersion string) {
	t.Helper()
	origNew := newSelfUpdater
	origVersion := version.Version
	newSelfUpdater = func() (selfUpdater, error) { return fake, nil }
	version.Version = currentVersion
	t.Cleanup(func() {
		newSelfUpdater = origNew
		version.Version = origVersion
		_ = updateCmd.Flags().Set("check", "false")
		_ = updateCmd.Flags().Set("yes", "false")
	})
}

func TestRunUpdateFlow(t *testing.T) {
	rel := func(v string) *updateRelease {
		return &updateRelease{version: v, notes: "notes", url: "https://example.com/rel"}
	}

	t.Run("dev build refuses", func(t *testing.T) {
		fake := &fakeUpdater{}
		withUpdateEnv(t, fake, "dev")
		err := runUpdate(updateCmd, nil)
		if err == nil || !strings.Contains(err.Error(), "dev build") {
			t.Fatalf("runUpdate() on dev build = %v, want dev build error", err)
		}
	})

	t.Run("already up to date", func(t *testing.T) {
		fake := &fakeUpdater{release: rel("1.0.0"), found: true}
		withUpdateEnv(t, fake, "1.0.0")
		if err := runUpdate(updateCmd, nil); err != nil {
			t.Fatalf("runUpdate() = %v, want nil", err)
		}
		if fake.applyCalled != 0 {
			t.Errorf("updateTo called %d times, want 0", fake.applyCalled)
		}
	})

	t.Run("no releases found", func(t *testing.T) {
		fake := &fakeUpdater{found: false}
		withUpdateEnv(t, fake, "1.0.0")
		if err := runUpdate(updateCmd, nil); err != nil {
			t.Fatalf("runUpdate() = %v, want nil", err)
		}
	})

	t.Run("detect error surfaces", func(t *testing.T) {
		fake := &fakeUpdater{detectErr: errors.New("network down")}
		withUpdateEnv(t, fake, "1.0.0")
		err := runUpdate(updateCmd, nil)
		if err == nil || !strings.Contains(err.Error(), "network down") {
			t.Fatalf("runUpdate() = %v, want wrapped detect error", err)
		}
	})

	t.Run("check only skips apply", func(t *testing.T) {
		fake := &fakeUpdater{release: rel("2.0.0"), found: true}
		withUpdateEnv(t, fake, "1.0.0")
		_ = updateCmd.Flags().Set("check", "true")
		if err := runUpdate(updateCmd, nil); err != nil {
			t.Fatalf("runUpdate(--check) = %v, want nil", err)
		}
		if fake.applyCalled != 0 {
			t.Errorf("updateTo called %d times, want 0", fake.applyCalled)
		}
	})

	t.Run("non-tty without yes errors", func(t *testing.T) {
		// Test processes have no terminal on stdin, so the confirm step must
		// reject instead of silently cancelling.
		fake := &fakeUpdater{release: rel("2.0.0"), found: true}
		withUpdateEnv(t, fake, "1.0.0")
		err := runUpdate(updateCmd, nil)
		if err == nil || !strings.Contains(err.Error(), "--yes") {
			t.Fatalf("runUpdate() without tty = %v, want --yes guidance error", err)
		}
		if fake.applyCalled != 0 {
			t.Errorf("updateTo called %d times, want 0", fake.applyCalled)
		}
	})

	t.Run("yes applies update", func(t *testing.T) {
		fake := &fakeUpdater{release: rel("2.0.0"), found: true}
		withUpdateEnv(t, fake, "1.0.0")
		_ = updateCmd.Flags().Set("yes", "true")
		if err := runUpdate(updateCmd, nil); err != nil {
			t.Fatalf("runUpdate(--yes) = %v, want nil", err)
		}
		if fake.applyCalled != 1 {
			t.Errorf("updateTo called %d times, want 1", fake.applyCalled)
		}
	})

	t.Run("permission error surfaces", func(t *testing.T) {
		fake := &fakeUpdater{
			release:  rel("2.0.0"),
			found:    true,
			applyErr: &fs.PathError{Op: "rename", Path: "/usr/local/bin/sarde", Err: syscall.EACCES},
		}
		withUpdateEnv(t, fake, "1.0.0")
		_ = updateCmd.Flags().Set("yes", "true")
		err := runUpdate(updateCmd, nil)
		if err == nil || !strings.Contains(err.Error(), "applying update") {
			t.Fatalf("runUpdate() with EACCES = %v, want applying update error", err)
		}
	})
}
