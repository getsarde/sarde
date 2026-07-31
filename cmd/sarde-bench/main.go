// Command sarde-bench measures sarde build performance over a fixture site.
//
// It shells out to a built sarde binary, times each build as a real child
// process (the wall time a user experiences), parses the structured stats
// from `sarde build --format json`, and reports median wall time, per-phase
// medians, pages/sec, and peak RSS where the platform supports it. It can
// also benchmark a Hugo site the same way for a local head-to-head.
//
// This is a repo-maintainer and CI utility, deliberately not a subcommand of
// the shipped sarde binary. See benchmarks/README.md for usage.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	var (
		sardeBin      = flag.String("sarde-bin", "", "path to a built sarde binary (required)")
		site          = flag.String("site", "benchmarks/fixtures/sarde", "fixture directory to build")
		runs          = flag.Int("runs", 5, "number of timed runs")
		warm          = flag.Bool("warm", false, "warm mode: one untimed warm-up run, then N runs with .cache left intact (default cold: delete .cache before every run)")
		jsonPath      = flag.String("json", "", "write the full BenchResult JSON to this path")
		baselinePath  = flag.String("baseline", "", "compare against a previous --json output at this path")
		warnThreshold = flag.Float64("warn-threshold", 25, "percent regression labeled REGRESSION in the report")
		failThreshold = flag.Float64("fail-threshold", 0, "percent regression that makes the process exit 1 (0 = report-only)")
		hugoBin       = flag.String("hugo-bin", "", "path to a hugo binary; if set, also benchmarks --hugo-site")
		hugoSite      = flag.String("hugo-site", "benchmarks/fixtures/hugo", "fixture directory for hugo")
		verbose       = flag.Bool("verbose", false, "print each subprocess's stderr even on success")
	)
	flag.Parse()

	if *sardeBin == "" {
		fmt.Fprintln(os.Stderr, "sarde-bench: --sarde-bin is required")
		flag.Usage()
		os.Exit(2)
	}
	if _, err := os.Stat(*sardeBin); err != nil {
		fatalf("sarde binary not found at %s: %v", *sardeBin, err)
	}
	if *runs < 1 {
		fatalf("--runs must be at least 1, got %d", *runs)
	}
	siteDir, err := filepath.Abs(*site)
	if err != nil {
		fatalf("resolving --site: %v", err)
	}
	if _, err := os.Stat(filepath.Join(siteDir, "sarde.yaml")); err != nil {
		fatalf("fixture %s has no sarde.yaml; generate it first (see benchmarks/README.md)", siteDir)
	}

	result, err := runBenchmark(*sardeBin, siteDir, *runs, *warm, *verbose)
	if err != nil {
		fatalf("benchmarking sarde: %v", err)
	}

	var hugoResult *BenchResult
	if *hugoBin != "" {
		hugoSiteDir, err := filepath.Abs(*hugoSite)
		if err != nil {
			fatalf("resolving --hugo-site: %v", err)
		}
		hr, err := runHugoBenchmark(*hugoBin, hugoSiteDir, *runs, *verbose)
		if err != nil {
			fatalf("benchmarking hugo: %v", err)
		}
		hugoResult = &hr
	}

	printReport(os.Stdout, result, hugoResult)

	if *jsonPath != "" {
		if err := writeJSON(*jsonPath, result); err != nil {
			fatalf("writing --json: %v", err)
		}
	}

	if *baselinePath != "" {
		baseline, err := readBaseline(*baselinePath)
		if err != nil {
			fatalf("reading --baseline: %v", err)
		}
		if baseline == nil {
			fmt.Printf("\nBaseline %s is empty; skipping comparison. Seed it from a CI run (see benchmarks/README.md).\n", *baselinePath)
		} else {
			d := diffAgainstBaseline(result, *baseline, *warnThreshold)
			printDiff(os.Stdout, result, d)
			if *failThreshold > 0 && d.WallDeltaPct > *failThreshold {
				fmt.Fprintf(os.Stderr, "\nsarde-bench: wall time regressed %.1f%%, above --fail-threshold %.1f%%\n", d.WallDeltaPct, *failThreshold)
				os.Exit(1)
			}
		}
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "sarde-bench: "+format+"\n", args...)
	os.Exit(2)
}

func writeJSON(path string, result BenchResult) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// readBaseline loads a BenchResult from a prior --json output. An empty file
// or an empty JSON object ({}) returns nil, allowing the checked-in baseline
// to start as a placeholder before it is seeded from CI.
func readBaseline(path string) (*BenchResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var b BenchResult
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}
	if b.Runs == 0 && b.WallMsMedian == 0 {
		return nil, nil
	}
	return &b, nil
}
