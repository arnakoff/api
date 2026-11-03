// SPDX-License-Identifier: AGPL-3.0-only

package mcp_test

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/arnakoff/api/internal/adapter/contracttest"
	"github.com/arnakoff/api/internal/adapter/mcp"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// paris is a zone other than UTC: times are written in UTC.
var paris = time.FixedZone("CET", 3600)

// contract is the public contract, loaded once: it is read-only.
var contract = sync.OnceValues(contracttest.Load)

// checker is a fake check answering report and keeping the inputs it was
// given.
type checker struct {
	mu     sync.Mutex
	report domain.Report
	inputs []domain.Input
}

func (c *checker) Check(_ context.Context, in domain.Input) domain.Report {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inputs = append(c.inputs, in)
	return c.report
}

func (c *checker) checked() []domain.Input {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.inputs
}

// sourceLister is a fake list of sources.
type sourceLister []app.ListedSource

func (l sourceLister) Sources() []app.ListedSource { return l }

// connect returns a client session to a server of opts, over the SDK's
// in-memory transport; the session is closed with the test.
func connect(t *testing.T, opts mcp.Options) *sdk.ClientSession {
	t.Helper()
	server, client := sdk.NewInMemoryTransports()
	ss, err := mcp.NewServer(opts).Connect(t.Context(), server, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	cs, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "v0.0.0"}, nil).Connect(t.Context(), client, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// call calls the tool name with args and fails the test on a protocol
// error; a tool error is in the result.
func call(t *testing.T, cs *sdk.ClientSession, name string, args any) *sdk.CallToolResult {
	t.Helper()
	res, err := cs.CallTool(t.Context(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	return res
}

// text returns the text of res, which must be a single text content.
func text(t *testing.T, res *sdk.CallToolResult) string {
	t.Helper()
	if len(res.Content) != 1 {
		t.Fatalf("content = %d items, want 1", len(res.Content))
	}
	tc, ok := res.Content[0].(*sdk.TextContent)
	if !ok {
		t.Fatalf("content is %T, want text", res.Content[0])
	}
	return tc.Text
}

// holdToContract fails the test unless the structured content of res
// follows the contract's schema name and its text is the same JSON.
func holdToContract(t *testing.T, res *sdk.CallToolResult, name string) {
	t.Helper()
	ct, err := contract()
	if err != nil {
		t.Fatal(err)
	}
	if err := ct.CheckSchema(name, res.StructuredContent); err != nil {
		t.Errorf("structured content breaks the contract: %v", err)
	}
	var got any
	if err := json.Unmarshal([]byte(text(t, res)), &got); err != nil {
		t.Fatalf("text is not JSON: %v", err)
	}
	sameJSON(t, "text", got, res.StructuredContent)
}

// sameJSON fails the test unless got and want write the same JSON value,
// whatever the order of their object members.
func sameJSON(t *testing.T, what string, got, want any) {
	t.Helper()
	g, w := decoded(t, got), decoded(t, want)
	if !reflect.DeepEqual(g, w) {
		t.Errorf("%s = %v, want %v", what, g, w)
	}
}

// decoded returns v written in JSON and read back as a generic value.
func decoded(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}
