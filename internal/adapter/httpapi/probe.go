// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"io"
	"net/http"
)

// healthzPath is the route of the liveness probe, which ProbeAlive asks.
const healthzPath = "/healthz"

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
	mux := http.NewServeMux()
	mux.Handle("GET "+healthzPath, healthz())
	mux.Handle("GET /readyz", readyz(ready))
	return mux
}
