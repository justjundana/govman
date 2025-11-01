package version

import (
	"fmt"
	"runtime"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
	BuildBy = "unknown"
)

type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	BuildBy   string `json:"buildBy"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
}

// Get aggregates build-time and runtime information into an Info struct.
// No parameters. Returns Info containing version, commit, date, builder, Go version, and platform.
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		BuildBy:   BuildBy,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// BuildVersion returns the display version string.
// For development builds, it formats as "dev-<commit>"; otherwise returns the version as-is.
func BuildVersion() string {
	info := Get()
	if info.Version == "dev" {
		return fmt.Sprintf("%s-%s", info.Version, info.Commit)
	}

	return info.Version
}
