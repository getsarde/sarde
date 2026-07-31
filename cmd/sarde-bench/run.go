package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/getsarde/sarde/internal/engine"
)

// runResult holds the measurements from a single timed build.
type runResult struct {
	WallMS    float64
	PeakRSSMB float64
	RSSOK     bool
	Build     engine.BuildResult
}

// runOnce executes one `sarde build` into a fresh temp output directory and
// returns wall time, peak RSS (where supported), and the parsed BuildResult.
func runOnce(binary, siteDir string, verbose bool) (runResult, error) {
	outDir, err := os.MkdirTemp("", "sarde-bench-out-*")
	if err != nil {
		return runResult{}, err
	}
	defer os.RemoveAll(outDir)

	cmd := exec.Command(binary, "build", siteDir, "--format", "json", "-o", outDir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	runErr := cmd.Run()
	wallMS := float64(time.Since(start).Microseconds()) / 1000.0

	if verbose && stderr.Len() > 0 {
		fmt.Fprintln(os.Stderr, stderr.String())
	}
	if runErr != nil {
		return runResult{}, fmt.Errorf("build failed: %w\nstderr:\n%s\nstdout:\n%s", runErr, stderr.String(), stdout.String())
	}

	var result engine.BuildResult
	if perr := parseLastJSONLine(stdout.Bytes(), &result); perr != nil {
		return runResult{}, fmt.Errorf("parsing build json: %w\nstdout:\n%s", perr, stdout.String())
	}

	rssMB, rssOK := peakRSSMB(cmd.ProcessState)
	return runResult{WallMS: wallMS, PeakRSSMB: rssMB, RSSOK: rssOK, Build: result}, nil
}

// parseLastJSONLine scans stdout and unmarshals the last line that is valid
// JSON into dst. Defensive against any incidental non-JSON output on the
// build path; the JSON summary is always the final line on success.
func parseLastJSONLine(out []byte, dst *engine.BuildResult) error {
	scanner := bufio.NewScanner(bytes.NewReader(out))
	// A ~1k page build result with warnings and log messages can exceed the
	// default 64KB token limit; allow lines up to 16MB.
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	var lastJSON []byte
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		if json.Valid(line) {
			lastJSON = append(lastJSON[:0], line...)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if lastJSON == nil {
		return fmt.Errorf("no JSON line found in build output")
	}
	return json.Unmarshal(lastJSON, dst)
}

// clearCache removes the fixture's incremental-build cache so a run starts cold.
func clearCache(siteDir string) error {
	return os.RemoveAll(filepath.Join(siteDir, ".cache"))
}

// runBenchmark executes N timed sarde builds and aggregates them.
// Cold mode deletes .cache before every run. Warm mode deletes it once,
// performs one untimed warm-up build, then leaves the cache intact for the
// timed runs.
func runBenchmark(binary, siteDir string, runs int, warm, verbose bool) (BenchResult, error) {
	mode := "cold"
	if warm {
		mode = "warm"
		if err := clearCache(siteDir); err != nil {
			return BenchResult{}, err
		}
		fmt.Fprintf(os.Stderr, "warm-up build...\n")
		if _, err := runOnce(binary, siteDir, verbose); err != nil {
			return BenchResult{}, fmt.Errorf("warm-up run: %w", err)
		}
	}

	results := make([]runResult, 0, runs)
	for i := 1; i <= runs; i++ {
		if !warm {
			if err := clearCache(siteDir); err != nil {
				return BenchResult{}, err
			}
		}
		fmt.Fprintf(os.Stderr, "run %d/%d...\n", i, runs)
		r, err := runOnce(binary, siteDir, verbose)
		if err != nil {
			return BenchResult{}, fmt.Errorf("run %d: %w", i, err)
		}
		results = append(results, r)
	}

	return aggregate(results, siteDir, mode), nil
}

// runHugoBenchmark times `hugo` builds of the comparison fixture the same
// way. Only wall time and peak RSS are recorded; Hugo's page accounting is
// not parsed (its stdout format is not a stable interface). Each run deletes
// resources/_gen so Hugo's asset cache does not carry over, approximating
// the cold mode used for sarde.
func runHugoBenchmark(hugoBin, siteDir string, runs int, verbose bool) (BenchResult, error) {
	if _, err := os.Stat(filepath.Join(siteDir, "hugo.yaml")); err != nil {
		return BenchResult{}, fmt.Errorf("fixture %s has no hugo.yaml; generate content and place the config/theme first (see benchmarks/README.md)", siteDir)
	}

	results := make([]runResult, 0, runs)
	for i := 1; i <= runs; i++ {
		if err := os.RemoveAll(filepath.Join(siteDir, "resources", "_gen")); err != nil {
			return BenchResult{}, err
		}
		outDir, err := os.MkdirTemp("", "sarde-bench-hugo-*")
		if err != nil {
			return BenchResult{}, err
		}

		cmd := exec.Command(hugoBin, "--source", siteDir, "--destination", outDir, "--quiet")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		start := time.Now()
		runErr := cmd.Run()
		wallMS := float64(time.Since(start).Microseconds()) / 1000.0
		if verbose && stderr.Len() > 0 {
			fmt.Fprintln(os.Stderr, stderr.String())
		}
		if runErr != nil {
			os.RemoveAll(outDir)
			return BenchResult{}, fmt.Errorf("hugo run %d: %w\nstderr:\n%s", i, runErr, stderr.String())
		}

		rssMB, rssOK := peakRSSMB(cmd.ProcessState)
		results = append(results, runResult{WallMS: wallMS, PeakRSSMB: rssMB, RSSOK: rssOK})
		os.RemoveAll(outDir)
		fmt.Fprintf(os.Stderr, "hugo run %d/%d done\n", i, runs)
	}

	return aggregate(results, siteDir, "cold"), nil
}
