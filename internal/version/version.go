package version

import (
	"fmt"
	"runtime"
)

// Build-time variables injected via ldflags.
// These are set during the build process and default to "dev" values if not set.
var (
	Version   = "dev"             // Version string (e.g., "0.1.0")
	Commit    = "none"            // Git commit hash
	Date      = "unknown"         // Build date (RFC3339 format)
	GoVersion = runtime.Version() // Go version used to build (overridable via ldflags)
)

// Short returns just the version string.
// Example: "0.1.0"
func Short() string {
	return Version
}

// Full returns version with short commit hash.
// Example: "0.1.0 (abc1234)"
func Full() string {
	commitShort := Commit
	if len(Commit) > 7 {
		commitShort = Commit[:7]
	}
	return fmt.Sprintf("%s (%s)", Version, commitShort)
}

// Platform returns the current OS and architecture.
// Example: "darwin/arm64"
func Platform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}

// Detailed returns multi-line version information including all build metadata.
// Example:
//
//	ssherpa 0.1.0
//	Commit:    abc1234def890
//	Built:     2026-02-15T10:00:00Z
//	Go:        go1.25.0
//	Platform:  darwin/arm64
func Detailed() string {
	return fmt.Sprintf(`ssherpa %s
Commit:    %s
Built:     %s
Go:        %s
Platform:  %s`,
		Version,
		Commit,
		Date,
		GoVersion,
		Platform(),
	)
}
