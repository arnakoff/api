// SPDX-License-Identifier: AGPL-3.0-only

package mcp_test

import (
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/mcp"
	"github.com/arnakoff/api/internal/adapter/wire"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

func TestListSourcesAnswersEverySourceAsTheAPIDoes(t *testing.T) {
	t.Parallel()

	listed := sourceLister{
		{
			Info: domain.SourceInfo{
				Name: "AMF", Publisher: "Autorité des marchés financiers", Licence: "Licence Ouverte 2.0",
				URL: "https://www.data.gouv.fr/datasets/641d5dab1d84b530f7720b08",
			},
			Synced: true,
			Sync:   app.SourceSync{Source: "AMF", LastSuccess: time.Date(2026, 11, 2, 9, 14, 42, 0, paris), Rows: 3466},
		},
		{Info: domain.SourceInfo{Name: "SIRENE", Publisher: "DINUM", Licence: "Licence Ouverte 2.0", URL: "https://sirene.example"}},
	}
	cs := connect(t, mcp.Options{Check: &checker{}, Sources: listed})

	res := call(t, cs, "list_sources", map[string]any{})

	if res.IsError {
		t.Fatalf("list_sources is an error: %s", text(t, res))
	}
	holdToContract(t, res, "SourceList")
	sameJSON(t, "sources", res.StructuredContent, wire.SourcesOf(listed))
}
