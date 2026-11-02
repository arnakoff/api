// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestBatchRefusesABadBodyWithoutChecking(t *testing.T) {
	t.Parallel()

	for name, body := range map[string]string{
		"empty":                   ``,
		"not JSON":                `items=armen.ltd`,
		"an array":                `[{"q":"armen.ltd"}]`,
		"without items":           `{}`,
		"with null items":         `{"items":null}`,
		"with no item":            `{"items":[]}`,
		"with an item not object": `{"items":["armen.ltd"]}`,
		"with a q not a string":   `{"items":[{"q":42}]}`,
		"truncated":               `{"items":[{"q":"armen.ltd"}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			isRefusedAsInvalidBody(t, batchCall{body: body, refused: true})
		})
	}
}

// The contract's validator ignores data after the value: the server does
// not.
func TestBatchRefusesDataAfterTheBody(t *testing.T) {
	t.Parallel()

	isRefusedAsInvalidBody(t, batchCall{body: `{"items":[{"q":"armen.ltd"}]} {}`})
}

// isRefusedAsInvalidBody holds that call is refused as invalid_body
// without any check.
func isRefusedAsInvalidBody(t *testing.T, call batchCall) {
	t.Helper()
	res, c := call.send(t, blacklisted())
	if res.Code != http.StatusBadRequest || decode(t, res)["code"] != "invalid_body" {
		t.Errorf("answer = %d, want 400 invalid_body", res.Code)
	}
	if asked := c.inputs(); len(asked) != 0 {
		t.Errorf("checked %v, want no check", asked)
	}
}

func TestBatchAcceptsOnlyJSON(t *testing.T) {
	t.Parallel()

	body := `{"items":[{"q":"armen.ltd"}]}`
	for _, tt := range []struct {
		contentType string
		status      int
	}{
		{"application/json; charset=utf-8", http.StatusOK},
		{"Application/JSON", http.StatusOK},
		{"text/plain", http.StatusUnsupportedMediaType},
		{"application/x-www-form-urlencoded", http.StatusUnsupportedMediaType},
		{"none", http.StatusUnsupportedMediaType},
	} {
		t.Run(tt.contentType, func(t *testing.T) {
			t.Parallel()
			call := batchCall{body: body, contentType: tt.contentType, refused: tt.status != http.StatusOK}
			res, c := call.send(t, blacklisted())

			if res.Code != tt.status {
				t.Errorf("status = %d, want %d", res.Code, tt.status)
			}
			if tt.status != http.StatusOK && len(c.inputs()) != 0 {
				t.Errorf("checked %v, want no check", c.inputs())
			}
		})
	}
}

// The contract's validator reads a malformed parameter as JSON: the server
// does not.
func TestBatchRefusesAMalformedContentType(t *testing.T) {
	t.Parallel()

	res, c := batchCall{body: `{"items":[{"q":"armen.ltd"}]}`, contentType: "application/json; charset"}.
		send(t, blacklisted())

	if res.Code != http.StatusUnsupportedMediaType || len(c.inputs()) != 0 {
		t.Errorf("answer = %d after %d checks, want 415 without check", res.Code, len(c.inputs()))
	}
}

func TestBatchLimitsTheBodySize(t *testing.T) {
	t.Parallel()

	// A long q, as a pasted URL may be; each item is about 2 KiB.
	item := `{"q":"armen.ltd/` + strings.Repeat("a", 2000) + `"}`
	for _, tt := range []struct {
		name     string
		maxItems int
		items    int
		status   int
	}{
		{"at least 64 KiB", 50, 31, http.StatusOK},
		{"over 64 KiB", 50, 33, http.StatusRequestEntityTooLarge},
		{"1 KiB per item above 64", 200, 100, http.StatusOK},
		{"over 1 KiB per item", 200, 102, http.StatusRequestEntityTooLarge},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, _ := batchCall{body: items(tt.items, item), maxItems: tt.maxItems}.send(t, blacklisted())

			if res.Code != tt.status {
				t.Errorf("status = %d, want %d", res.Code, tt.status)
			}
			if tt.status != http.StatusOK && decode(t, res)["code"] != "content_too_large" {
				t.Error("want the content_too_large problem")
			}
		})
	}
}

func TestBatchAcceptsUnknownFields(t *testing.T) {
	t.Parallel()

	res, _ := batchCall{body: `{"items":[{"q":"armen.ltd","note":"x"}],"x":1}`}.send(t, blacklisted())

	if res.Code != http.StatusOK {
		t.Errorf("status = %d, want 200: clients may send fields we ignore", res.Code)
	}
}
