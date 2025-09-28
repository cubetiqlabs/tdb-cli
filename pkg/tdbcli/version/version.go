package version

import (
	"fmt"
	"strings"
)

// Name identifies the CLI in user agents and metadata.
const Name = "tdb-cli"

// IssuesURL points to the canonical tracker for bugs and feedback.
const IssuesURL = "https://github.com/cubetiqlabs/tdb-cli/issues"

// Version holds the CLI version. It can be overridden at build time via -ldflags.
var Version = "dev"

// Commit holds the git commit hash for the build (overridden via -ldflags).
var Commit = ""

// BuildTime holds the build timestamp (overridden via -ldflags).
var BuildTime = ""

// value returns a sanitized version string.
func value() string {
	v := strings.TrimSpace(Version)
	if v == "" {
		return "dev"
	}
	return v
}

func commitValue() string {
	c := strings.TrimSpace(Commit)
	if c == "" {
		return "unknown"
	}
	if len(c) > 40 {
		c = c[:40]
	}
	return c
}

func builtAtValue() string {
	t := strings.TrimSpace(BuildTime)
	if t == "" {
		return "unknown"
	}
	return t
}

// UserAgent returns the default User-Agent header for CLI HTTP requests.
func UserAgent() string {
	return fmt.Sprintf("%s/%s (+https://github.com/cubetiqlabs/tdb-cli)", Name, value())
}

// DefaultAPIKeyDescription returns a fallback description used when creating API keys.
func DefaultAPIKeyDescription() string {
	return fmt.Sprintf("Generated via %s/%s", Name, value())
}

// DefaultApplicationDescription returns a fallback description used when creating applications.
func DefaultApplicationDescription() string {
	return fmt.Sprintf("Created via %s/%s", Name, value())
}

// Number returns the sanitized version string.
func Number() string {
	return value()
}

// Display returns the combined CLI name and version.
func Display() string {
	return fmt.Sprintf("%s/%s", Name, value())
}

// CommitHash returns the sanitized git commit hash associated with the build.
func CommitHash() string {
	return commitValue()
}

// BuiltAt returns the sanitized build timestamp (as provided at build time).
func BuiltAt() string {
	return builtAtValue()
}

// Info aggregates key build metadata for display purposes.
func Info() map[string]string {
	return map[string]string{
		"name":     Name,
		"version":  value(),
		"commit":   commitValue(),
		"built_at": builtAtValue(),
		"issues":   IssuesURL,
	}
}
