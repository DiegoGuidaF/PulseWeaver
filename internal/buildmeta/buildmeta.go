// Package buildmeta exposes the build identity of the running binary.
//
// The values are injected at link time with -ldflags -X (see Dockerfile and
// Makefile). The Go toolchain's own VCS stamping via runtime/debug is not
// usable here: .dockerignore excludes .git/, so a build inside the release
// image has no repository to read, and a binary built from source reports
// Main.Version as "(devel)" rather than the release tag.
//
// The name avoids "version" and "buildinfo" — both collide with stdlib
// package names (go/version, debug/buildinfo) and trip revive's var-naming.
package buildmeta

// Injected at link time. The defaults are what an un-stamped build honestly
// is — a developer build of unknown provenance — so a `go run` binary never
// claims to be a release.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = ""
)

// Info is the build identity as a single value.
type Info struct {
	Version   string
	Commit    string
	BuildTime string
}

// Get returns the build identity injected into this binary.
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
	}
}
