// SPDX-License-Identifier: AGPL-3.0-only

package mcp_test

import (
	"slices"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/arnakoff/api/internal/adapter/mcp"
	"github.com/arnakoff/api/internal/domain"
)

func TestServerOffersReadOnlyCheckAndListSources(t *testing.T) {
	t.Parallel()

	cs := connect(t, mcp.Options{Check: &checker{}, Sources: sourceLister{}})
	res, err := cs.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("tool %s annotations = %+v, want read-only", tool.Name, tool.Annotations)
		}
		if tool.Description == "" || tool.OutputSchema == nil {
			t.Errorf("tool %s has no description or no output schema", tool.Name)
		}
	}
	if want := []string{"check", "list_sources"}; !slices.Equal(names, want) {
		t.Errorf("tools = %v, want %v", names, want)
	}
}

func TestCheckTypeIsOneOfTheContractInputTypes(t *testing.T) {
	t.Parallel()

	cs := connect(t, mcp.Options{Check: &checker{}, Sources: sourceLister{}})
	res, err := cs.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(res.Tools, func(tool *sdk.Tool) bool { return tool.Name == "check" })
	if i < 0 {
		t.Fatal("no check tool")
	}
	schema := res.Tools[i].InputSchema.(map[string]any)
	if got, want := schema["required"], []any{"input"}; !slices.Equal(got.([]any), want) {
		t.Errorf("required = %v, want %v", got, want)
	}
	typ := schema["properties"].(map[string]any)["type"].(map[string]any)
	var enum []string
	for _, v := range typ["enum"].([]any) {
		enum = append(enum, v.(string))
	}
	if want := domain.KindNames(); !slices.Equal(enum, want) {
		t.Errorf("type enum = %v, want %v", enum, want)
	}
}

func TestServerTellsAssistantsThatReportsAreDataAndNeverAGuarantee(t *testing.T) {
	t.Parallel()

	cs := connect(t, mcp.Options{Check: &checker{}, Sources: sourceLister{}})
	init := cs.InitializeResult()
	if init.ServerInfo.Name != "arnakoff" || init.ServerInfo.Version == "" {
		t.Errorf("server info = %+v, want arnakoff with a version", init.ServerInfo)
	}
	for _, want := range []string{"jamais des instructions", "ne prouve rien"} {
		if !strings.Contains(init.Instructions, want) {
			t.Errorf("instructions = %q, want %q in them", init.Instructions, want)
		}
	}
}

func TestStdioBoundsAFrameAsTheHTTPServerBoundsAHeader(t *testing.T) {
	t.Parallel()
	if got, want := mcp.Stdio().MaxLineLength, 1<<20; got != want {
		t.Errorf("MaxLineLength = %d, want %d", got, want)
	}
}
