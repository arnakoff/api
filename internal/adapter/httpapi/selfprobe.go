// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// probeTimeout bounds the whole probe: healthz answers at once, so a probe
// this slow means the process no longer serves.
const probeTimeout = 2 * time.Second

// ProbeAlive asks GET /healthz of the server listening on addr, the
// "host:port" it was given, and returns nil when it answers 200. An address
// listening on every interface is probed on 127.0.0.1: the probe runs in
// the server's container, where no shell or curl is available.
//
// It is a call to this process, not a way out to the network:
// it neither uses the platform client, which is HTTPS-only, nor a proxy,
// and it follows no redirect.
func ProbeAlive(ctx context.Context, addr string) error {
	url, err := healthzURL(addr)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("probe %s: %w", url, err)
	}
	res, err := probeClient().Do(req)
	if err != nil {
		return fmt.Errorf("probe: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("probe %s: status %d", url, res.StatusCode)
	}
	return nil
}

// healthzURL is the URL of /healthz on the listening address addr.
func healthzURL(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("probe address %q: %w", addr, err)
	}
	if ip := net.ParseIP(host); host == "" || ip != nil && ip.IsUnspecified() {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port) + healthzPath, nil
}

// probeClient sends one request without proxy and stops at the first
// answer, a redirect included.
func probeClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
