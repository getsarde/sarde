package main

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"
)

// BenchResult is the aggregated outcome of one benchmark (N runs of one
// site). It is the schema of --json output and of benchmarks/baseline.json.
type BenchResult struct {
	Site             string             `json:"site"`
	Mode             string             `json:"mode"` // "cold" | "warm"
	Runs             int                `json:"runs"`
	PageCount        int                `json:"pageCount"`
	WallMsMedian     float64            `json:"wallMsMedian"`
	WallMsMin        float64            `json:"wallMsMin"`
	WallMsMax        float64            `json:"wallMsMax"`
	WallMsRuns       []float64          `json:"wallMsRuns"`
	PeakRSSMBMedian  float64            `json:"peakRssMbMedian"`
	PeakRSSSupported bool               `json:"peakRssSupported"`
	PagesPerSec      float64            `json:"pagesPerSec"`
	PhaseMedianMs    map[string]float64 `json:"phaseMedianMs,omitempty"`
	GoVersion        string             `json:"goVersion"`
	GOOS             string             `json:"goos"`
	GOARCH           string             `json:"goarch"`
	Commit           string             `json:"commit,omitempty"`
	Timestamp        string             `json:"timestamp"`
}

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	mid := len(s) / 2
	if len(s)%2 == 1 {
		return s[mid]
	}
	return (s[mid-1] + s[mid]) / 2
}

func minMax(xs []float64) (min, max float64) {
	if len(xs) == 0 {
		return 0, 0
	}
	min, max = xs[0], xs[0]
	for _, x := range xs[1:] {
		if x < min {
			min = x
		}
		if x > max {
			max = x
		}
	}
	return min, max
}

// aggregate reduces N run results into one BenchResult.
func aggregate(runs []runResult, site, mode string) BenchResult {
	walls := make([]float64, len(runs))
	for i, r := range runs {
		walls[i] = r.WallMS
	}
	wallMedian := median(walls)
	wallMin, wallMax := minMax(walls)

	pageCount := 0
	if len(runs) > 0 {
		pageCount = runs[0].Build.PageCount
	}

	pagesPerSec := 0.0
	if wallMedian > 0 && pageCount > 0 {
		pagesPerSec = float64(pageCount) / (wallMedian / 1000.0)
	}

	// Per-phase medians, grouped by phase name across runs.
	phaseSamples := map[string][]float64{}
	for _, r := range runs {
		for _, pt := range r.Build.PhaseTimings {
			ms := float64(pt.Duration.Microseconds()) / 1000.0
			phaseSamples[pt.Phase] = append(phaseSamples[pt.Phase], ms)
		}
	}
	phaseMedians := make(map[string]float64, len(phaseSamples))
	for phase, samples := range phaseSamples {
		phaseMedians[phase] = median(samples)
	}

	rssSupported := len(runs) > 0
	var rssSamples []float64
	for _, r := range runs {
		if !r.RSSOK {
			rssSupported = false
			break
		}
		rssSamples = append(rssSamples, r.PeakRSSMB)
	}
	rssMedian := 0.0
	if rssSupported {
		rssMedian = median(rssSamples)
	}

	return BenchResult{
		Site:             site,
		Mode:             mode,
		Runs:             len(runs),
		PageCount:        pageCount,
		WallMsMedian:     wallMedian,
		WallMsMin:        wallMin,
		WallMsMax:        wallMax,
		WallMsRuns:       walls,
		PeakRSSMBMedian:  rssMedian,
		PeakRSSSupported: rssSupported,
		PagesPerSec:      pagesPerSec,
		PhaseMedianMs:    phaseMedians,
		GoVersion:        runtime.Version(),
		GOOS:             runtime.GOOS,
		GOARCH:           runtime.GOARCH,
		Commit:           gitCommit(),
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
	}
}

// gitCommit returns the short HEAD hash, best-effort.
func gitCommit() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// baselineDiff holds the comparison of a current result against a baseline.
type baselineDiff struct {
	WallDeltaPct  float64
	RSSDeltaPct   float64
	RSSComparable bool
	WallLabel     string
	RSSLabel      string
	Baseline      BenchResult
}

// diffAgainstBaseline computes percent deltas (positive = current is slower
// or larger) and labels each metric "ok" or "REGRESSION" against the warn
// threshold.
func diffAgainstBaseline(current, baseline BenchResult, warnThreshold float64) baselineDiff {
	d := baselineDiff{Baseline: baseline}

	if baseline.WallMsMedian > 0 {
		d.WallDeltaPct = (current.WallMsMedian - baseline.WallMsMedian) / baseline.WallMsMedian * 100
	}
	d.WallLabel = "ok"
	if d.WallDeltaPct > warnThreshold {
		d.WallLabel = "REGRESSION"
	}

	d.RSSComparable = current.PeakRSSSupported && baseline.PeakRSSSupported && baseline.PeakRSSMBMedian > 0
	if d.RSSComparable {
		d.RSSDeltaPct = (current.PeakRSSMBMedian - baseline.PeakRSSMBMedian) / baseline.PeakRSSMBMedian * 100
		d.RSSLabel = "ok"
		if d.RSSDeltaPct > warnThreshold {
			d.RSSLabel = "REGRESSION"
		}
	}
	return d
}

// printReport renders the primary results as GitHub-flavored markdown. The
// output is readable in a terminal and renders as tables when appended to
// $GITHUB_STEP_SUMMARY.
func printReport(w io.Writer, result BenchResult, hugo *BenchResult) {
	fmt.Fprintf(w, "## sarde build benchmark\n\n")
	fmt.Fprintf(w, "Site: `%s` | mode: %s | runs: %d | %s %s/%s", result.Site, result.Mode, result.Runs, result.GoVersion, result.GOOS, result.GOARCH)
	if result.Commit != "" {
		fmt.Fprintf(w, " | commit %s", result.Commit)
	}
	fmt.Fprintf(w, "\n\n")

	fmt.Fprintf(w, "| Metric | Value |\n|---|---|\n")
	fmt.Fprintf(w, "| Pages | %d |\n", result.PageCount)
	fmt.Fprintf(w, "| Wall time (median) | %.0f ms |\n", result.WallMsMedian)
	fmt.Fprintf(w, "| Wall time (min / max) | %.0f / %.0f ms |\n", result.WallMsMin, result.WallMsMax)
	fmt.Fprintf(w, "| Pages/sec | %.0f |\n", result.PagesPerSec)
	if result.PeakRSSSupported {
		fmt.Fprintf(w, "| Peak RSS (median) | %.0f MB |\n", result.PeakRSSMBMedian)
	} else {
		fmt.Fprintf(w, "| Peak RSS (median) | n/a (unsupported on %s) |\n", result.GOOS)
	}

	if len(result.PhaseMedianMs) > 0 {
		fmt.Fprintf(w, "\n<details><summary>Per-phase medians</summary>\n\n")
		fmt.Fprintf(w, "| Phase | Median |\n|---|---|\n")
		// Sort phases by descending median so the expensive ones lead.
		type phaseRow struct {
			name string
			ms   float64
		}
		rows := make([]phaseRow, 0, len(result.PhaseMedianMs))
		for name, ms := range result.PhaseMedianMs {
			rows = append(rows, phaseRow{name, ms})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].ms > rows[j].ms })
		for _, r := range rows {
			fmt.Fprintf(w, "| %s | %.1f ms |\n", r.name, r.ms)
		}
		fmt.Fprintf(w, "\n</details>\n")
	}

	if hugo != nil {
		fmt.Fprintf(w, "\n### Hugo comparison (`%s`)\n\n", hugo.Site)
		fmt.Fprintf(w, "| Metric | sarde | hugo |\n|---|---|---|\n")
		fmt.Fprintf(w, "| Wall time (median) | %.0f ms | %.0f ms |\n", result.WallMsMedian, hugo.WallMsMedian)
		if result.PeakRSSSupported && hugo.PeakRSSSupported {
			fmt.Fprintf(w, "| Peak RSS (median) | %.0f MB | %.0f MB |\n", result.PeakRSSMBMedian, hugo.PeakRSSMBMedian)
		}
		fmt.Fprintf(w, "\nHugo page counts are not parsed; the fixtures hold a comparable content volume by construction.\n")
	}
}

// printDiff renders the vs-baseline table.
func printDiff(w io.Writer, current BenchResult, d baselineDiff) {
	fmt.Fprintf(w, "\n### vs baseline")
	if d.Baseline.Commit != "" {
		fmt.Fprintf(w, " (commit %s, %s)", d.Baseline.Commit, d.Baseline.Timestamp)
	}
	fmt.Fprintf(w, "\n\n| Metric | Baseline | Current | Delta | Status |\n|---|---|---|---|---|\n")
	fmt.Fprintf(w, "| Wall time (median) | %.0f ms | %.0f ms | %+.1f%% | %s |\n", d.Baseline.WallMsMedian, current.WallMsMedian, d.WallDeltaPct, d.WallLabel)
	if d.RSSComparable {
		fmt.Fprintf(w, "| Peak RSS (median) | %.0f MB | %.0f MB | %+.1f%% | %s |\n", d.Baseline.PeakRSSMBMedian, current.PeakRSSMBMedian, d.RSSDeltaPct, d.RSSLabel)
	}
}
