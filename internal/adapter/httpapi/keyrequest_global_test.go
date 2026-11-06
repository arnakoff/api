// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/httpapi"
)

func TestKeyRequestStopsMailingOverTheGlobalLimit(t *testing.T) {
	t.Parallel()
	opts, logs := options(t, true)
	q := &requester{}
	opts.Keys = httpapi.KeyRequestOptions{
		Requests: q, Origin: siteOrigin, Verifier: &verifier{},
		Limit: frozenLimiter(1 << 30), AddressLimit: frozenLimiter(1), GlobalLimit: frozenLimiter(2),
	}

	for i, addr := range []string{"dev@exemple.fr", "autre@exemple.fr"} {
		if res := linkRequest(addr).send(t, opts); res.Code != http.StatusAccepted {
			t.Fatalf("request %d: status = %d, want 202", i, res.Code)
		}
	}
	// Over the cap, an address that already had its share is answered as
	// a new one: the answer does not tell which addresses asked for links.
	for _, addr := range []string{"troisieme@exemple.fr", "dev@exemple.fr"} {
		res := linkRequest(addr).send(t, opts)
		if res.Code != http.StatusServiceUnavailable || decode(t, res)["code"] != "key_request_unavailable" {
			t.Fatalf("%s over the cap: status = %d, body %s; want 503 key_request_unavailable", addr, res.Code, res.Body)
		}
	}
	if got := q.addresses(); len(got) != 2 {
		t.Errorf("mailed %v, want the first two links only", got)
	}
	var capped int
	for _, line := range logs.Lines(t) {
		if line["msg"] == "login link cap reached" && line["level"] == "WARN" {
			capped++
		}
	}
	if capped != 2 || strings.Contains(logs.String(), "exemple.fr") {
		t.Errorf("logs = %s, want two warnings without the address", logs)
	}
}

func TestKeyRequestSpendsTheGlobalLimitOnMailedLinksOnly(t *testing.T) {
	t.Parallel()
	opts, q, _ := keyOptions(t)
	opts.Keys.AddressLimit = frozenLimiter(1)
	opts.Keys.GlobalLimit = frozenLimiter(2)

	for i, addr := range []string{"dev@exemple.fr", "DEV@exemple.fr", "autre@exemple.fr"} {
		if res := linkRequest(addr).send(t, opts); res.Code != http.StatusAccepted {
			t.Fatalf("request %d: status = %d, want 202", i, res.Code)
		}
	}
	if got := q.addresses(); len(got) != 2 {
		t.Errorf("mailed %v, want two links: the one held back gave its share back", got)
	}
}
