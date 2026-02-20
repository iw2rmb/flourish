package flourish

import (
	_ "embed"
	"regexp"
	"strings"
)

var semverRE = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)

//go:embed VERSION
var embeddedVersion string

// Version returns the library version string in SemVer format (without `v`).
func Version() string {
	return strings.TrimSpace(embeddedVersion)
}

// VersionTag returns the git tag form of Version (with leading `v`).
func VersionTag() string {
	return "v" + Version()
}

// IsSemver reports whether v matches SemVer 2.0.0.
func IsSemver(v string) bool {
	return semverRE.MatchString(strings.TrimSpace(v))
}

// VersionIsSemver reports whether the embedded Version is valid SemVer.
func VersionIsSemver() bool {
	return IsSemver(Version())
}
