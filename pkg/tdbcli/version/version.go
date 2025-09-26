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
	return fmt.Sprintf("%s/%s (+https://github.com/cubetiqlabs/tinydb/pkg/tdbcli)", Name, value())
}

// DefaultAPIKeyDescription returns a fallback description used when creating API keys.
func DefaultAPIKeyDescription() string {
	return fmt.Sprintf("Generated via %s/%s", Name, value())
}
