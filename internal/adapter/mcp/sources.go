// SPDX-License-Identifier: AGPL-3.0-only

package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/arnakoff/api/internal/adapter/wire"
)

// listSourcesTool describes the list_sources tool, which takes no
// argument.
func listSourcesTool() *sdk.Tool {
	return &sdk.Tool{
		Name:        "list_sources",
		Title:       "Lister les sources",
		Description: "Les sources consultées, leur licence et l'état de leurs données en service.",
		Annotations: &sdk.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: new(false)},
	}
}

// listSources answers the list_sources tool: every source l lists.
func listSources(l SourceLister) sdk.ToolHandlerFor[struct{}, wire.SourceList] {
	return func(context.Context, *sdk.CallToolRequest, struct{}) (*sdk.CallToolResult, wire.SourceList, error) {
		return nil, wire.SourcesOf(l.Sources()), nil
	}
}
