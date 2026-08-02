package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/getsarde/sarde/internal/version"
)

func envFrom(m map[string]string) func(string) string {
	return func(key string) string { return m[key] }
}

func TestShouldSkipUpdateCheck(t *testing.T) {
	tests := []struct {
		name  string
		ver   string
		env   map[string]string
		quiet bool
		isTTY bool
		want  bool
	}{
		{"all clear runs", "1.2.3", nil, false, true, false},
		{"dev build skips", "dev", nil, false, true, true},
		{"CI skips", "1.2.3", map[string]string{"CI": "true"}, false, true, true},
		{"opt-out env skips", "1.2.3", map[string]string{"SARDE_NO_UPDATE_CHECK": "1"}, false, true, true},
		{"quiet skips", "1.2.3", nil, true, true, true},
		{"non-tty skips", "1.2.3", nil, false, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSkipUpdateCheck(tt.ver, envFrom(tt.env), tt.quiet, tt.isTTY)
			if got != tt.want {
				t.Errorf("shouldSkipUpdateCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateCacheRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "update-check.json")

	if _, err := loadUpdateCache(path); err == nil {
		t.Fatal("loadUpdateCache() on missing file succeeded, want error")
	}

	in := &updateCheckState{
		LastChecked:   time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC),
		LatestVersion: "1.4.0",
		Notified:      true,
	}
	if err := saveUpdateCache(path, in); err != nil {
		t.Fatalf("saveUpdateCache() failed: %v", err)
	}
	out, err := loadUpdateCache(path)
	if err != nil {
		t.Fatalf("loadUpdateCache() failed: %v", err)
	}
	if !out.LastChecked.Equal(in.LastChecked) || out.LatestVersion != in.LatestVersion || out.Notified != in.Notified {
		t.Errorf("round trip mismatch: got %+v, want %+v", out, in)
	}
}

func TestUpdateCacheStaleness(t *testing.T) {
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		checked time.Time
		want    bool
	}{
		{"just checked", now.Add(-time.Minute), false},
		{"under a day", now.Add(-23 * time.Hour), false},
		{"over a day", now.Add(-25 * time.Hour), true},
		{"zero value", time.Time{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &updateCheckState{LastChecked: tt.checked}
			if got := s.isStale(now); got != tt.want {
				t.Errorf("isStale() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateNotice(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{"newer version notifies", "1.2.3", "1.3.0", true},
		{"same version silent", "1.2.3", "1.2.3", false},
		{"older cached version silent", "1.2.3", "1.1.0", false},
		{"empty cache silent", "1.2.3", "", false},
		{"unparsable current silent", "dev", "1.3.0", false},
		{"unparsable latest silent", "1.2.3", "not-a-version", false},
		{"v-prefixed latest parses", "1.2.3", "v1.3.0", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, got := updateNotice(tt.current, tt.latest)
			if got != tt.want {
				t.Fatalf("updateNotice(%q, %q) ok = %v, want %v", tt.current, tt.latest, got, tt.want)
			}
			if got && !strings.Contains(msg, "sarde update") {
				t.Errorf("notice %q does not mention `sarde update`", msg)
			}
		})
	}
}

// withPassiveFetch swaps the network seam and the running version for one
// test, restoring both afterward.
func withPassiveFetch(t *testing.T, currentVersion string, fetch func() (string, bool)) {
	t.Helper()
	origFetch := fetchLatestVersion
	origVersion := version.Version
	fetchLatestVersion = fetch
	version.Version = currentVersion
	t.Cleanup(func() {
		fetchLatestVersion = origFetch
		version.Version = origVersion
	})
}

func TestPendingUpdateCheckFinish(t *testing.T) {
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	staleState := updateCheckState{LastChecked: now.Add(-48 * time.Hour)}

	t.Run("fast lookup notifies in the same run", func(t *testing.T) {
		withPassiveFetch(t, "1.0.0", func() (string, bool) { return "2.0.0", true })
		path := filepath.Join(t.TempDir(), "update-check.json")

		p := newPendingUpdateCheck(path, staleState, now)
		if p.result == nil {
			t.Fatal("stale cache did not start a lookup")
		}
		var out bytes.Buffer
		p.finish(&out, time.Second, now)

		if !strings.Contains(out.String(), "v2.0.0") {
			t.Errorf("no notice printed, got %q", out.String())
		}
		saved, err := loadUpdateCache(path)
		if err != nil {
			t.Fatalf("cache not written: %v", err)
		}
		if saved.LatestVersion != "2.0.0" || !saved.Notified || !saved.LastChecked.Equal(now) {
			t.Errorf("saved state = %+v, want 2.0.0/notified/now", saved)
		}
	})

	t.Run("slow lookup records the attempt without a notice", func(t *testing.T) {
		release := make(chan struct{})
		withPassiveFetch(t, "1.0.0", func() (string, bool) {
			<-release
			return "2.0.0", true
		})
		t.Cleanup(func() { close(release) })
		path := filepath.Join(t.TempDir(), "update-check.json")

		p := newPendingUpdateCheck(path, staleState, now)
		var out bytes.Buffer
		p.finish(&out, 10*time.Millisecond, now)

		if out.Len() > 0 {
			t.Errorf("notice printed despite timeout: %q", out.String())
		}
		saved, err := loadUpdateCache(path)
		if err != nil {
			t.Fatalf("cache not written on timeout: %v", err)
		}
		if !saved.LastChecked.Equal(now) {
			t.Errorf("LastChecked = %v, want %v (attempt must be recorded)", saved.LastChecked, now)
		}
		if saved.LatestVersion != "" {
			t.Errorf("LatestVersion = %q, want empty", saved.LatestVersion)
		}
	})

	t.Run("fresh cache starts no lookup and notifies from cached state", func(t *testing.T) {
		withPassiveFetch(t, "1.0.0", func() (string, bool) {
			t.Error("fetchLatestVersion called despite fresh cache")
			return "", false
		})
		path := filepath.Join(t.TempDir(), "update-check.json")
		fresh := updateCheckState{LastChecked: now.Add(-time.Hour), LatestVersion: "1.5.0"}

		p := newPendingUpdateCheck(path, fresh, now)
		if p.result != nil {
			t.Fatal("fresh cache started a lookup")
		}
		var out bytes.Buffer
		p.finish(&out, time.Second, now)

		if !strings.Contains(out.String(), "v1.5.0") {
			t.Errorf("no notice from cached state, got %q", out.String())
		}
		saved, err := loadUpdateCache(path)
		if err != nil {
			t.Fatalf("cache not written: %v", err)
		}
		if !saved.Notified {
			t.Error("Notified not persisted")
		}
	})

	t.Run("already notified version stays silent and writes nothing", func(t *testing.T) {
		withPassiveFetch(t, "1.0.0", func() (string, bool) { return "", false })
		path := filepath.Join(t.TempDir(), "update-check.json")
		fresh := updateCheckState{LastChecked: now.Add(-time.Hour), LatestVersion: "1.5.0", Notified: true}

		p := newPendingUpdateCheck(path, fresh, now)
		var out bytes.Buffer
		p.finish(&out, time.Second, now)

		if out.Len() > 0 {
			t.Errorf("notice printed twice: %q", out.String())
		}
		if _, err := loadUpdateCache(path); err == nil {
			t.Error("cache written although nothing changed")
		}
	})

	t.Run("newer version than the notified one announces again", func(t *testing.T) {
		withPassiveFetch(t, "1.0.0", func() (string, bool) { return "3.0.0", true })
		path := filepath.Join(t.TempDir(), "update-check.json")
		prev := updateCheckState{LastChecked: now.Add(-48 * time.Hour), LatestVersion: "2.0.0", Notified: true}

		p := newPendingUpdateCheck(path, prev, now)
		var out bytes.Buffer
		p.finish(&out, time.Second, now)

		if !strings.Contains(out.String(), "v3.0.0") {
			t.Errorf("newer version not announced, got %q", out.String())
		}
		saved, err := loadUpdateCache(path)
		if err != nil {
			t.Fatalf("cache not written: %v", err)
		}
		if saved.LatestVersion != "3.0.0" || !saved.Notified {
			t.Errorf("saved state = %+v, want 3.0.0 notified", saved)
		}
	})

	t.Run("failed lookup keeps cached version and records attempt", func(t *testing.T) {
		withPassiveFetch(t, "1.0.0", func() (string, bool) { return "", false })
		path := filepath.Join(t.TempDir(), "update-check.json")
		prev := updateCheckState{LastChecked: now.Add(-48 * time.Hour), LatestVersion: "2.0.0", Notified: true}

		p := newPendingUpdateCheck(path, prev, now)
		var out bytes.Buffer
		p.finish(&out, time.Second, now)

		saved, err := loadUpdateCache(path)
		if err != nil {
			t.Fatalf("cache not written: %v", err)
		}
		if saved.LatestVersion != "2.0.0" || !saved.Notified || !saved.LastChecked.Equal(now) {
			t.Errorf("saved state = %+v, want 2.0.0 notified with attempt recorded", saved)
		}
	})

	t.Run("nil receiver is a no-op", func(t *testing.T) {
		var p *pendingUpdateCheck
		p.finishAndNotify()
	})
}
