package cli

import (
	"cmp"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/version"
)

// buildReporter prints the human-readable `sarde build` output: a header
// before the build, live phase progress while it runs, then the summary.
// Everything the final line counts is printed above it.
type buildReporter struct {
	w          io.Writer // summary destination (stdout); live lines go through devlog (stderr)
	verbose    bool
	quiet      bool
	projectDir string   // absolute
	contentDir string   // absolute; resolves content-relative warning paths
	phases     []string // ordered full-build phase names

	resolved map[string]string // warning File -> display path
}

func newBuildReporter(w io.Writer, projectDir string, verbose, quiet bool, phases []string) *buildReporter {
	if abs, err := filepath.Abs(projectDir); err == nil {
		projectDir = abs
	}
	return &buildReporter{
		w:          w,
		verbose:    verbose && !quiet,
		quiet:      quiet,
		projectDir: projectDir,
		phases:     phases,
		resolved:   make(map[string]string),
	}
}

func (r *buildReporter) header() {
	if r.quiet {
		return
	}
	fmt.Fprintf(r.w, "Building site with sarde v%s (%s/%s)\n", version.Version, runtime.GOOS, runtime.GOARCH)
}

// onPhase runs as each build phase finishes: verbose output gets a permanent
// timing line, and the in-place progress line moves on to the next phase.
func (r *buildReporter) onPhase(pt engine.PhaseTiming) {
	if r.verbose {
		devlog.Log("build", "%-26s %6d ms", pt.Phase, pt.Duration.Milliseconds())
	}
	if next := r.nextPhase(pt.Phase); next != "" {
		devlog.SetProgress("build", "%s...", next)
	} else {
		devlog.ClearProgress()
	}
}

func (r *buildReporter) nextPhase(done string) string {
	if i := slices.Index(r.phases, done); i >= 0 && i+1 < len(r.phases) {
		return r.phases[i+1]
	}
	return ""
}

// summary prints everything after the build. consoleWarns is the number of
// warnings already printed to the log during the build (devlog.Warn), which
// the final line counts alongside result.Warnings.
func (r *buildReporter) summary(result *engine.BuildResult, cfg *config.SiteConfig, consoleWarns int) {
	lines, listed := r.prepareWarnings(result.Warnings)
	if r.quiet {
		r.printWarnings(lines, listed)
		return
	}

	if r.verbose {
		r.printPlugins(result.LogMessages)
		r.printSlowestPages(result.SlowestPages)
	}

	fmt.Fprintln(r.w)
	printStatsTable(r.w, result)
	r.printWarnings(lines, listed)

	total := len(result.Warnings) + consoleWarns
	fmt.Fprintf(r.w, "\n%s\n", verdictLine(result.PageCount, result.Duration, total, total-listed))
	fmt.Fprintf(r.w, "  Output: %s\n", result.OutputDir)

	if r.verbose {
		fmt.Fprintf(r.w, "  Theme: %s\n", cfg.Theme.Name)
		fmt.Fprintf(r.w, "  Base path: %q\n", cfg.Build.BasePath)
		fmt.Fprintf(r.w, "  Content dir: %s\n", cfg.Content.Dir)
	}
}

// verdictLine is the last summary line: "Built 48 pages in 312 ms", plus the
// warning total and how many of those were printed in the log rather than
// in the list.
func verdictLine(pages int, dur time.Duration, total, inLog int) string {
	line := fmt.Sprintf("Built %s in %d ms", plural(pages, "page", "pages"), dur.Milliseconds())
	if total > 0 {
		line += ", " + plural(total, "warning", "warnings")
		if inLog > 0 {
			line += fmt.Sprintf(" (%d in the log above)", inLog)
		}
	}
	return line
}

func (r *buildReporter) printWarnings(lines []string, listed int) {
	if listed == 0 {
		return
	}
	fmt.Fprintf(r.w, "\n%s:\n", plural(listed, "warning", "warnings"))
	for _, l := range lines {
		fmt.Fprintln(r.w, l)
	}
}

func (r *buildReporter) printPlugins(msgs []engine.BuildLogEntry) {
	if len(msgs) == 0 {
		return
	}
	// BuildDone hooks run in parallel, so arrival order is random.
	sorted := slices.Clone(msgs)
	slices.SortStableFunc(sorted, func(a, b engine.BuildLogEntry) int {
		return cmp.Compare(a.Source, b.Source)
	})
	fmt.Fprintln(r.w, "\nPlugins")
	for _, m := range sorted {
		fmt.Fprintf(r.w, "  [%s] %s\n", m.Source, m.Message)
	}
}

func (r *buildReporter) printSlowestPages(pages []engine.PageTiming) {
	if len(pages) == 0 {
		return
	}
	fmt.Fprintln(r.w, "\nSlowest pages")
	for _, p := range pages {
		path := p.Path
		if !strings.HasPrefix(path, "/") {
			path = devlog.DisplayPath(filepath.Join(r.projectDir, filepath.FromSlash(path)))
		}
		ms := float64(p.Duration.Microseconds()) / 1000
		fmt.Fprintf(r.w, "  %7.1f ms  %s\n", ms, path)
	}
}

// warningEntry is one distinct warning line; count > 1 renders as "(xN)".
type warningEntry struct {
	file, field, msg, level string
	line, col, count        int
}

// prepareWarnings formats the warnings list: paths made clickable, sorted by
// location, identical warnings collapsed. Link warnings are skipped because
// the link report already printed them during the build. listed is how many
// warnings (counting duplicates) the lines cover.
func (r *buildReporter) prepareWarnings(ws []engine.ValidationWarning) (lines []string, listed int) {
	var entries []*warningEntry
	index := make(map[warningEntry]*warningEntry)
	for _, w := range ws {
		if w.Field == "link" {
			continue
		}
		listed++
		key := warningEntry{
			file:  r.resolveWarningFile(w.File),
			field: w.Field,
			msg:   stripLinePrefix(w.Message, w.Line),
			level: w.Level,
			line:  w.Line,
			col:   w.Col,
		}
		if e, ok := index[key]; ok {
			e.count++
			continue
		}
		e := key
		e.count = 1
		index[key] = &e
		entries = append(entries, &e)
	}

	slices.SortStableFunc(entries, func(a, b *warningEntry) int {
		return cmp.Or(
			cmp.Compare(a.file, b.file),
			cmp.Compare(a.line, b.line),
			cmp.Compare(a.col, b.col),
			cmp.Compare(a.field, b.field),
			cmp.Compare(a.msg, b.msg),
		)
	})

	for _, e := range entries {
		label := " WARN"
		if e.level == "error" {
			label = "ERROR"
		}
		text := e.msg
		if e.field != "" {
			text = "[" + e.field + "] " + text
		}
		l := fmt.Sprintf("  %s  %s  %s", label, warningLocation(e.file, e.line, e.col), text)
		if e.count > 1 {
			l += fmt.Sprintf(" (x%d)", e.count)
		}
		lines = append(lines, l)
	}
	return lines, listed
}

// resolveWarningFile turns a warning's File into a path relative to the
// working directory. Warnings carry absolute paths, project-relative paths,
// content-relative paths (frontmatter checks), or config references such as
// "sarde.yaml: plugins.config.x", which are kept as they are.
func (r *buildReporter) resolveWarningFile(f string) string {
	if f == "" || strings.Contains(f, ": ") {
		return f
	}
	if p, ok := r.resolved[f]; ok {
		return p
	}
	p := filepath.ToSlash(f)
	if filepath.IsAbs(f) {
		p = devlog.DisplayPath(f)
	} else {
		for _, base := range []string{r.projectDir, r.contentDir} {
			if base == "" {
				continue
			}
			candidate := filepath.Join(base, f)
			if _, err := os.Stat(candidate); err == nil {
				p = devlog.DisplayPath(candidate)
				break
			}
		}
	}
	r.resolved[f] = p
	return p
}

// warningLocation renders file[:line[:col]], the form terminals and editors
// open at the right position.
func warningLocation(file string, line, col int) string {
	if file == "" || line <= 0 {
		return file
	}
	loc := file + ":" + strconv.Itoa(line)
	if col > 0 {
		loc += ":" + strconv.Itoa(col)
	}
	return loc
}

// stripLinePrefix drops a leading "line N: " from msg when N matches the
// warning's Line, which the location already shows. content_lint keeps the
// prefix in Message for consumers that do not read Line yet.
func stripLinePrefix(msg string, line int) string {
	if line <= 0 {
		return msg
	}
	return strings.TrimPrefix(msg, "line "+strconv.Itoa(line)+": ")
}

// plural formats a count with the matching noun form: "1 page", "2 pages".
func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}

func printStatsTable(w io.Writer, result *engine.BuildResult) {
	type row struct {
		label string
		value int
	}
	rows := []row{
		{"Pages", result.PageCount},
		{"Paginator pages", result.PaginatorPages},
		{"Collections", result.Collections},
		{"Bundle assets", result.BundleAssets},
		{"Public files", result.PublicFiles},
		{"Processed images", result.ProcessedImages},
		{"Aliases", result.AliasCount},
		{"Sitemaps", result.SitemapCount},
	}

	fmt.Fprintf(w, "%19s | Total\n", "")
	fmt.Fprintf(w, "-------------------+-------\n")
	for _, r := range rows {
		fmt.Fprintf(w, "  %-17s|%5d\n", r.label, r.value)
	}
}
