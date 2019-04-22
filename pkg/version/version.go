package version

import (
	"github.com/Masterminds/semver"
)

const Devel = "devel"

// Values for these are injected by the build
var (
	version = Devel
	commit  string
)

// GetVersion returns the Drake version. This is either a semantic version
// number or else, in the case of unreleased code, the string "devel".
func GetVersion() string {
	return version
}

// GetSemver returns a semantic version object that represents the Drake
// version.
func GetSemver() (*semver.Version, error) {
	return semver.NewVersion(version)
}

// GetCommit returns the git commit SHA for the code that Drake was built from.
func GetCommit() string {
	return commit
}
