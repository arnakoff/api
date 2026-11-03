// SPDX-License-Identifier: AGPL-3.0-only

// Package buildinfo tells the version of the running binary.
//
// The container image injects it at link time:
//
//	go build -ldflags "-X github.com/arnakoff/api/internal/platform/buildinfo.version=0.48.0"
//
// Without it, the version of the main module recorded by the Go toolchain
// is used, as `go install …@v0.48.0` sets it; a local build is "dev".
package buildinfo

import "runtime/debug"

// version is set by the linker only, never at run time.
var version string

// devel is the module version the Go toolchain records for a build from a
// working tree it cannot version.
const devel = "(devel)"

// Version returns the version of the running binary, never empty.
func Version() string {
	return resolve(version, debug.ReadBuildInfo)
}

// resolve returns injected when set, else the version of the main module
// read from the build information, else "dev".
func resolve(injected string, read func() (*debug.BuildInfo, bool)) string {
	if injected != "" {
		return injected
	}
	if info, ok := read(); ok && info.Main.Version != "" && info.Main.Version != devel {
		return info.Main.Version
	}
	return "dev"
}
