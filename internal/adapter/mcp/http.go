// SPDX-License-Identifier: AGPL-3.0-only

package mcp

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// HTTP returns the remote endpoint of s, over the Streamable HTTP
// transport. It keeps no session: each POST carries one message, answered
// in JSON, and GET and DELETE are refused. A body is bounded by
// maxFrameBytes, a JSON-RPC batch and a browser's cross-origin request are
// refused, and the SDK logs nothing: a tool's arguments may be personal
// data. The caller checks the API key and spends its quota, one unit a
// request.
func HTTP(s *sdk.Server) http.Handler {
	h := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return s }, &sdk.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 true,
		MaxRequestBodyBytes:          maxFrameBytes,
		PropagateRequestCancellation: true,
	})
	return http.NewCrossOriginProtection().Handler(oneMessage(h))
}

// oneMessage answers next only for a body of at most maxFrameBytes that is
// not a JSON-RPC batch, which the SDK accepts from protocol versions before
// 2025-06-18: one request would then run many tool calls for one unit of
// quota.
func oneMessage(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxFrameBytes))
		var tooLarge *http.MaxBytesError
		switch {
		case errors.As(err, &tooLarge):
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		case err != nil:
			http.Error(w, "request body not read", http.StatusBadRequest)
			return
		case bytes.HasPrefix(bytes.TrimLeft(body, " \t\r\n"), []byte("[")):
			http.Error(w, "JSON-RPC batching is not supported", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		next.ServeHTTP(w, r)
	})
}
