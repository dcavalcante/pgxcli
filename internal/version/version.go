package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the current version of pgxcli.
	Version = "unknown"
	// Commit is the git commit hash at build time.
	Commit = "unknown"
	// BuildTime is the date/time when the binary was built.
	BuildTime = "unknown"
)

// FullVersion returns a formatted string containing detailed version and build information.
func FullVersion() string {
	return fmt.Sprintf("pgxcli version: %s\ncommit:    %s\nbuild time: %s\ngo version: %s\nos/arch:    %s",
		Version, Commit, BuildTime, runtime.Version(), fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
}
