// SPDX-License-Identifier: AGPL-3.0-only

package mcp_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/arnakoff/api/internal/adapter/mcp"
)

// initialize is the first message of a session, as a client sends it.
const initialize = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25",` +
	`"capabilities":{},"clientInfo":{"name":"test","version":"v0.0.0"}}}`

// post sends body to h as an MCP client would, with header.
func post(t *testing.T, h http.Handler, body string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, vs := range header {
		req.Header[k] = vs
	}
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}

func TestHTTPServesTheTools(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(mcp.HTTP(mcp.NewServer(mcp.Options{Sources: sourceLister{}})))
	t.Cleanup(srv.Close)

	transport := &sdk.StreamableClientTransport{Endpoint: srv.URL, DisableStandaloneSSE: true, MaxRetries: -1}
	cs, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "v0.0.0"}, nil).Connect(t.Context(), transport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	res, err := cs.CallTool(t.Context(), &sdk.CallToolParams{Name: "list_sources", Arguments: map[string]any{}})
	if err != nil || res.IsError {
		t.Fatalf("list_sources = %+v, %v; want an answer", res, err)
	}
}

func TestHTTPAnswersInJSONWithoutSession(t *testing.T) {
	t.Parallel()
	res := post(t, mcp.HTTP(mcp.NewServer(mcp.Options{})), initialize, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("initialize: status %d, want 200; body %s", res.Code, res.Body)
	}
	if ct := res.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if id := res.Header().Get("Mcp-Session-Id"); id != "" {
		t.Errorf("Mcp-Session-Id = %q, want none: the server keeps no session", id)
	}
}

func TestHTTPRefusesAnOversizedBody(t *testing.T) {
	t.Parallel()
	body := initialize[:len(initialize)-1] + `,"pad":"` + strings.Repeat("a", 1<<20) + `"}`
	if res := post(t, mcp.HTTP(mcp.NewServer(mcp.Options{})), body, nil); res.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status %d, want 413", res.Code)
	}
}

func TestHTTPRefusesACrossOriginBrowserRequest(t *testing.T) {
	t.Parallel()
	header := http.Header{"Sec-Fetch-Site": {"cross-site"}, "Origin": {"https://evil.example"}}
	if res := post(t, mcp.HTTP(mcp.NewServer(mcp.Options{})), initialize, header); res.Code != http.StatusForbidden {
		t.Errorf("status %d, want 403", res.Code)
	}
}

// TestHTTPRefusesABatch keeps one POST to one message: a JSON-RPC batch,
// which protocol versions before 2025-06-18 allow, would spend one unit of
// the key's quota on many checks.
func TestHTTPRefusesABatch(t *testing.T) {
	t.Parallel()
	old := strings.Replace(initialize, "2025-11-25", "2025-03-26", 1)
	header := http.Header{"Mcp-Protocol-Version": {"2025-03-26"}}
	for _, body := range []string{"[" + old + "]", " \n[" + old + "," + strings.Replace(old, `"id":1`, `"id":2`, 1) + "]"} {
		if res := post(t, mcp.HTTP(mcp.NewServer(mcp.Options{})), body, header); res.Code != http.StatusBadRequest {
			t.Errorf("batch %.20q: status %d, want 400", body, res.Code)
		}
	}
}
