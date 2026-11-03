// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"net/http"

	"github.com/arnakoff/api/internal/adapter/wire"
	"github.com/arnakoff/api/internal/domain"
)

// The body of a batch may be up to batchBytesPerItem a permitted item, and
// at least minBatchBytes: room for long pasted URLs, not for a flood.
const (
	batchBytesPerItem = 1 << 10
	minBatchBytes     = 64 << 10
)

// batchRequest is a request as the contract's BatchRequest schema
// describes it; unknown fields are ignored.
type batchRequest struct {
	Items []batchItem `json:"items"`
}

// batchItem is an item as the contract's BatchItem schema describes it. A
// nil Type is a type left out.
type batchItem struct {
	Q    string  `json:"q"`
	Type *string `json:"type"`
}

// batchResult is a result as the contract's BatchResult schema describes
// it: a report or an error, never both.
type batchResult struct {
	Report *wire.Report `json:"report,omitempty"`
	Error  *problemBody `json:"error,omitempty"`
}

// batch answers POST /v1/check/batch: a result per item of the body, in
// its order, each as GET /v1/check would answer it. An invalid item gets
// its problem as its result; the body itself is refused only as a whole.
func batch(c Checker, ps problems, maxItems int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items, p, ok := batchItems(w, r, maxItems)
		if !ok {
			ps.write(w, r, p)
			return
		}
		body := struct {
			Results []batchResult `json:"results"`
		}{checkItems(r.Context(), c, ps, items)}
		writeJSON(w, "application/json", http.StatusOK, body)
	})
}

// batchItems returns the items the body of r holds, or the problem
// refusing it: a body other than JSON, larger than the limit of maxItems,
// without an item or with more than maxItems.
func batchItems(w http.ResponseWriter, r *http.Request, maxItems int) ([]batchItem, problem, bool) {
	limit := max(minBatchBytes, int64(maxItems)*batchBytesPerItem)
	var req batchRequest
	if p, ok := decodeBody(w, r, limit, &req, invalidBody); !ok {
		return nil, p, false
	}
	switch {
	case len(req.Items) == 0:
		return nil, invalidBody, false
	case len(req.Items) > maxItems:
		return nil, tooManyItems(maxItems), false
	}
	return req.Items, problem{}, true
}

// checkItems returns the result of each item: the inputs it parses are
// checked together, the others answer their problem.
func checkItems(ctx context.Context, c Checker, ps problems, items []batchItem) []batchResult {
	results := make([]batchResult, len(items))
	ins := make([]domain.Input, 0, len(items))
	at := make([]int, 0, len(items))
	for i, it := range items {
		in, p, ok := itemInput(it)
		if !ok {
			results[i].Error = new(ps.body(p))
			continue
		}
		ins, at = append(ins, in), append(at, i)
	}
	for j, report := range c.CheckAll(ctx, ins) {
		results[at[j]].Report = new(wire.ReportOf(ins[j].Kind(), report))
	}
	return results
}

// itemInput returns the input it asks to check, or the problem refusing
// it, as the query of GET /v1/check would.
func itemInput(it batchItem) (domain.Input, problem, bool) {
	if it.Type == nil {
		return parseInput(it.Q, "", false)
	}
	return parseInput(it.Q, *it.Type, true)
}
