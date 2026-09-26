package deploy

// Step names a phase of a deploy. The CLI's JSON reporter sends them to
// consumers such as Sarde Studio verbatim, so the values are part of the
// event contract: add new ones, never rename.
type Step string

const (
	StepCollect  Step = "collect"  // walking the output directory
	StepHash     Step = "hash"     // hashing files
	StepPrepare  Step = "prepare"  // creating the deploy, fetching tokens, project lookups
	StepUpload   Step = "upload"   // sending file contents
	StepFinalize Step = "finalize" // creating the deployment from uploaded files
	StepWait     Step = "wait"     // waiting for the provider to publish
)

// Progress is a snapshot of one step's advance. Done and Total count files;
// Bytes and BytesTotal are zero when a step does not measure bytes.
type Progress struct {
	Step       Step
	Done       int
	Total      int
	Bytes      int64
	BytesTotal int64
}

// Log levels for Reporter.Log.
const (
	LevelInfo = "info"
	LevelWarn = "warn"
)

// Reporter receives a deploy's progress. Deployers never print: everything a
// user sees goes through a Reporter, so the CLI can render it as human lines
// or as a machine-readable event stream. Implementations must be safe for
// concurrent use, because uploads report from several goroutines, and should
// throttle Progress themselves.
type Reporter interface {
	Step(s Step, msg string)
	Progress(p Progress)
	Log(level, msg string)
}

// NopReporter discards everything. Used by the HTTP API path and in tests
// that do not assert on progress.
type NopReporter struct{}

func (NopReporter) Step(Step, string)  {}
func (NopReporter) Progress(Progress)  {}
func (NopReporter) Log(string, string) {}

// orNop returns rep, or a NopReporter when rep is nil.
func orNop(rep Reporter) Reporter {
	if rep == nil {
		return NopReporter{}
	}
	return rep
}
