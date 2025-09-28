package version

import (
	"fmt"
	"strings"
)

// Name identifies the CLI in user agents and metadata.
const Name = "tdb-cli"

// Version holds the CLI version. It can be overridden at build time via -ldflags.
var Version = "dev"

// value returns a sanitized version string.
func value() string {
	v := strings.TrimSpace(Version)
	if v == "" {
		return "dev"
	}
	return v
}

// UserAgent returns the default User-Agent header for CLI HTTP requests.
func UserAgent() string {
	return fmt.Sprintf("%s/%s (+https://github.com/github.com/cubetiqlabs/tdb-cli/pkg/tdbcli)", Name, value())
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
