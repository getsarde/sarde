# Build-Performance Benchmarks

Everything benchmark-related lives in this directory:

- `baseline.json` (tracked): the known-good numbers CI compares against.
- `generators/sarde/` and `generators/hugo/` (tracked): deterministic, seeded
  fixture generators. Same seed, same output.
- `fixtures/` (gitignored): the generated corpora the benchmarks build.

The harness itself is `cmd/sarde-bench` at the repo root. The `testsite/`
directory is unrelated: it holds local dev-only sites and is never tracked.

## Regenerating the fixture

The sarde fixture (~1,000 markdown pages across blog, articles, news, docs,
tutorials, courses, and a French locale) is produced by the generator:

```
go -C benchmarks/generators/sarde run . -root <absolute path>/benchmarks/fixtures/sarde
```

Always pass `-root` explicitly. The generator's auto-detection uses
`os.Executable()`, which under `go run` resolves to a temporary build-cache
binary, not the source tree.

Example (Windows):

```
go -C benchmarks/generators/sarde run . -root D:\dev\my-repos\sarde\benchmarks\fixtures\sarde
```

Example (CI / Linux):

```
go -C benchmarks/generators/sarde run . -root "$GITHUB_WORKSPACE/benchmarks/fixtures/sarde"
```

## Running the harness

```
go build -o sarde ./cmd/sarde
go build -o sarde-bench ./cmd/sarde-bench
./sarde-bench --sarde-bin ./sarde --runs 5 --json results.json
```

The harness runs `sarde build --format json` N times (default 5) into a fresh
temp output directory per run and reports the median wall time, min/max,
pages/sec, per-phase medians, and peak RSS. Peak RSS is measured on Linux and
macOS; on Windows it reports n/a.

Modes:

- Cold (default): the fixture's `.cache/` is deleted before every run. This
  matches a fresh checkout and is what CI measures.
- Warm (`--warm`): the cache is cleared once, one untimed warm-up build runs,
  then N timed runs reuse the populated cache.

Useful flags: `--site` (fixture dir, default `benchmarks/fixtures/sarde`),
`--baseline <file>` (compare and print a delta table), `--warn-threshold`
(percent slower that gets labeled REGRESSION, default 25), `--fail-threshold`
(percent slower that exits 1; unset means report-only), `--verbose` (echo
subprocess stderr).

The in-process Go benchmarks in `internal/build/benchmark_test.go`
(`BenchmarkBuild_TestsiteBenchmark_*`) build the same fixture through the
engine directly and skip automatically when it has not been generated.

## Comparing against Hugo (local only)

`fixtures/hugo` holds a Hugo twin with a comparable content volume. Its
generator writes only the content; `hugo.yaml`, `themes/hugo-book/`, and
`layouts/` are hand-placed and not tracked, so this comparison is a local
maintainer workflow, not part of CI.

```
go -C benchmarks/generators/hugo run . -root <absolute path>/benchmarks/fixtures/hugo
./sarde-bench --sarde-bin ./sarde --hugo-bin hugo
```

Only wall time and peak RSS are compared; Hugo's page accounting is not parsed
because its stdout format is not a stable interface.

## The baseline

`baseline.json` is the known-good result that CI compares against. It is
seeded and updated manually, never auto-committed:

1. Trigger the Benchmark workflow (`.github/workflows/benchmark.yml`) via
   workflow_dispatch, or let it run on a push to main.
2. Download the `benchmark-results` artifact (`results.json`) from the run.
3. Review the numbers in the job summary.
4. Commit the file as `benchmarks/baseline.json`.

The baseline must come from a CI run on ubuntu-latest, not a dev machine:
hardware is not comparable across laptops and Windows cannot measure peak RSS.
The CI job is reporting-only by design. GitHub-hosted runners are too noisy to
hard-gate merges on wall time, so regressions show up as a REGRESSION label in
the job summary for a human to judge.
