package version

// Version is set to "dev" by default and overridden at build time via:
//
//	go build -ldflags "-X github.com/frostybee/sarde/internal/version.Version=1.2.3"
var Version = "dev"
