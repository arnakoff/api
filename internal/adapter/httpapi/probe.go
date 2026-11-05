// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"io"
	"net/http"
)

// Paths of the probes: liveness, which ProbeAlive asks, and readiness.
const (
	healthzPath = "/healthz"
	readyzPath  = "/readyz"
)

// passingProbe reports whether route is a probe and status says it passed:
// the platform probes every few hundred milliseconds, so such a request is
// the normal state and is not logged (#266). A failing probe is.
func passingProbe(route string, status int) bool {
	isProbe := route == "GET "+healthzPath || route == "GET "+readyzPath
	return isProbe && status >= 200 && status < 300
}

// Probe bodies: the probes are outside the public contract.
const (
	probeOK          = `{"status":"ok"}`
	probeUnavailable = `{"status":"indisponible"}`
)

// healthz answers while the process serves requests.
func healthz() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeProbe(w, http.StatusOK, probeOK)
	})
}

// readyz answers 200 once ready reports every snapshot loaded, 503 before.
func readyz(ready Readiness) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if !ready.Ready() {
			writeProbe(w, http.StatusServiceUnavailable, probeUnavailable)
			return
		}
		writeProbe(w, http.StatusOK, probeOK)
	})
}

func writeProbe(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

// NewProbeHandler serves only the liveness and readiness probes, for a
// process that answers nothing else, such as `admin migrate -listen`.
func NewProbeHandler(ready Readiness) http.Handler {
	return probeMux(ready)
}

// probeMux routes the liveness and readiness probes, for a process to add
// its own routes to.
func probeMux(ready Readiness) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET "+healthzPath, healthz())
	mux.Handle("GET "+readyzPath, readyz(ready))
	return mux
}
