package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/getsarde/sarde/internal/deploy"
)

// deployEventVersion is sent as "v" on every JSON event. Adding fields is
// compatible; renaming or removing one bumps the version.
const deployEventVersion = 1

// jsonDeployReporter writes deploy progress as newline-delimited JSON on
// stdout, one event per line. Sarde Studio parses this stream; the event
// names and fields are documented in docs/content/docs/reference/cli-commands.md.
type jsonDeployReporter struct {
	mu       sync.Mutex
	enc      *json.Encoder
	lastSent map[deploy.Step]time.Time
	now      func() time.Time
	interval time.Duration
}

func newJSONDeployReporter(w io.Writer) *jsonDeployReporter {
	return &jsonDeployReporter{
		enc:      json.NewEncoder(w),
		lastSent: map[deploy.Step]time.Time{},
		now:      time.Now,
		interval: 100 * time.Millisecond,
	}
}

func (r *jsonDeployReporter) emit(event string, fields map[string]any) {
	fields["v"] = deployEventVersion
	fields["event"] = event
	r.mu.Lock()
	defer r.mu.Unlock()
	_ = r.enc.Encode(fields)
}

func (r *jsonDeployReporter) Start(provider string) {
	r.emit("start", map[string]any{"provider": provider})
}

func (r *jsonDeployReporter) Step(s deploy.Step, msg string) {
	r.emit("step", map[string]any{"step": string(s), "message": msg})
}

// Progress is throttled per step; the first and the final (Done == Total)
// snapshots always go out.
func (r *jsonDeployReporter) Progress(p deploy.Progress) {
	r.mu.Lock()
	now := r.now()
	last, seen := r.lastSent[p.Step]
	final := p.Total > 0 && p.Done >= p.Total
	if seen && !final && now.Sub(last) < r.interval {
		r.mu.Unlock()
		return
	}
	r.lastSent[p.Step] = now
	r.mu.Unlock()
	fields := map[string]any{"step": string(p.Step), "done": p.Done, "total": p.Total}
	if p.BytesTotal > 0 {
		fields["bytes"] = p.Bytes
		fields["bytes_total"] = p.BytesTotal
	}
	r.emit("progress", fields)
}

func (r *jsonDeployReporter) Log(level, msg string) {
	r.emit("log", map[string]any{"level": level, "message": msg})
}

func (r *jsonDeployReporter) Result(res *deploy.Result, elapsed time.Duration) {
	r.emit("result", map[string]any{
		"ok":             true,
		"provider":       res.Provider,
		"url":            res.URL,
		"deploy_url":     res.DeployURL,
		"deploy_id":      res.DeployID,
		"admin_url":      res.AdminURL,
		"files_total":    res.FilesTotal,
		"files_uploaded": res.FilesUploaded,
		"bytes_uploaded": res.BytesUploaded,
		"duration_ms":    elapsed.Milliseconds(),
	})
}

func (r *jsonDeployReporter) Check(res *deploy.CheckResult) {
	r.emit("check", map[string]any{
		"ok":        true,
		"provider":  res.Provider,
		"target_id": res.TargetID,
		"target":    res.Target,
		"url":       res.URL,
		"account":   res.Account,
	})
}

// prettyDeployReporter prints human-readable progress. On a terminal the
// upload progress redraws one line; elsewhere (CI logs) it prints at every
// quarter so a log stays short.
type prettyDeployReporter struct {
	mu       sync.Mutex
	w        io.Writer
	quiet    bool
	tty      bool
	quarter  map[deploy.Step]int
	openLine bool
}

func newPrettyDeployReporter(w io.Writer, quiet, tty bool) *prettyDeployReporter {
	return &prettyDeployReporter{w: w, quiet: quiet, tty: tty, quarter: map[deploy.Step]int{}}
}

// endLine finishes a redrawn progress line before other output.
func (r *prettyDeployReporter) endLine() {
	if r.openLine {
		fmt.Fprintln(r.w)
		r.openLine = false
	}
}

func (r *prettyDeployReporter) Start(provider string) {
	if r.quiet {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	fmt.Fprintf(r.w, "Deploying with %s...\n", provider)
}

func (r *prettyDeployReporter) Step(_ deploy.Step, msg string) {
	if r.quiet || msg == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.endLine()
	fmt.Fprintf(r.w, "  %s\n", msg)
}

func (r *prettyDeployReporter) Progress(p deploy.Progress) {
	if r.quiet || p.Step != deploy.StepUpload || p.Total == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	pct := p.Done * 100 / p.Total
	line := fmt.Sprintf("  %d/%d files (%d%%)", p.Done, p.Total, pct)
	if p.BytesTotal > 0 {
		line += fmt.Sprintf(", %s of %s", humanBytes(p.Bytes), humanBytes(p.BytesTotal))
	}
	if r.tty {
		fmt.Fprintf(r.w, "\r%s", line)
		r.openLine = true
		if p.Done >= p.Total {
			r.endLine()
		}
		return
	}
	q := pct / 25
	if q > r.quarter[p.Step] || (p.Done >= p.Total && r.quarter[p.Step] < 4) {
		r.quarter[p.Step] = q
		fmt.Fprintln(r.w, line)
	}
}

func (r *prettyDeployReporter) Log(level, msg string) {
	if r.quiet {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.endLine()
	if level == deploy.LevelWarn {
		fmt.Fprintf(r.w, "  Warning: %s\n", msg)
		return
	}
	fmt.Fprintf(r.w, "    %s\n", strings.TrimRight(msg, "\n"))
}

func (r *prettyDeployReporter) Result(res *deploy.Result, elapsed time.Duration) {
	if r.quiet {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.endLine()
	if res.URL != "" {
		fmt.Fprintf(r.w, "Deploy complete: %s (%s)\n", res.URL, elapsed.Round(time.Second))
	} else {
		fmt.Fprintf(r.w, "Deploy complete (%s).\n", elapsed.Round(time.Second))
	}
	if res.DeployID != "" && res.Provider != "github-pages" {
		fmt.Fprintf(r.w, "Deploy ID: %s\n", res.DeployID)
	}
}

func (r *prettyDeployReporter) Check(res *deploy.CheckResult) {
	if r.quiet {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	target := firstNonEmptyStr(res.Target, res.TargetID)
	fmt.Fprintf(r.w, "Credentials OK: %s can deploy to %s", res.Provider, target)
	if res.URL != "" {
		fmt.Fprintf(r.w, " (%s)", res.URL)
	}
	fmt.Fprintln(r.w)
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
