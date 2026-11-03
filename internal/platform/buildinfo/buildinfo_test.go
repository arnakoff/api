// SPDX-License-Identifier: AGPL-3.0-only

package buildinfo_test

import (
	"runtime/debug"
	"testing"

	"github.com/arnakoff/api/internal/platform/buildinfo"
)

// built returns a reader of build information whose main module has
// version v, or no information at all when ok is false.
func built(v string, ok bool) func() (*debug.BuildInfo, bool) {
	return func() (*debug.BuildInfo, bool) {
		if !ok {
			return nil, false
		}
		return &debug.BuildInfo{Main: debug.Module{Version: v}}, true
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		injected string
		read     func() (*debug.BuildInfo, bool)
		want     string
	}{
		{"injected at link time wins", "0.48.0", built("v0.47.0", true), "0.48.0"},
		{"module version without injection", "", built("v0.47.0", true), "v0.47.0"},
		{"local build", "", built("(devel)", true), "dev"},
		{"empty module version", "", built("", true), "dev"},
		{"no build information", "", built("", false), "dev"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := buildinfo.Resolve(tt.injected, tt.read); got != tt.want {
				t.Errorf("resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVersionIsNeverEmpty(t *testing.T) {
	t.Parallel()
	if buildinfo.Version() == "" {
		t.Error("Version() = \"\", want a version")
	}
}
