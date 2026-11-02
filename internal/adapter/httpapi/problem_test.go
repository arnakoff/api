// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/arnakoff/api/api"
	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/domain"
)

// contractProblems returns the contract's Problem… examples, by code and
// detail: arnakoff/web documents one page per code, with each detail of it.
func contractProblems(t *testing.T) map[string]map[string]any {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromData(api.Spec())
	if err != nil {
		t.Fatal(err)
	}
	examples := make(map[string]map[string]any)
	for name, ref := range doc.Components.Examples {
		if !strings.HasPrefix(name, "Problem") {
			continue
		}
		b, err := json.Marshal(ref.Value.Value)
		if err != nil {
			t.Fatal(err)
		}
		var p map[string]any
		if err := json.Unmarshal(b, &p); err != nil {
			t.Fatal(err)
		}
		examples[problemKey(p)] = p
	}
	return examples
}

// problemKey names a problem body by its code and its detail.
func problemKey(p map[string]any) string {
	return fmt.Sprintf("%v: %v", p["code"], p["detail"])
}

func TestEveryProblemIsAContractExample(t *testing.T) {
	t.Parallel()

	examples := contractProblems(t)
	opts, _ := options(t, true)
	var keys []string
	for _, p := range httpapi.Catalog {
		res := serve(t, httpapi.Wrap(opts, httpapi.Answer(opts, p)), http.MethodGet, "/v1/check", nil)
		got := decode(t, res)
		key := problemKey(got)
		keys = append(keys, key)
		want, ok := examples[key]
		if !ok {
			t.Errorf("problem %s has no Problem… example in the contract", key)
			continue
		}
		want["instance"] = "tag:arnakoff.com,2026:request:GENERATED"
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("problem %s:\n got %v\nwant %v", key, got, want)
		}
		if res.Code != int(want["status"].(float64)) {
			t.Errorf("problem %s: status = %d, want %v", key, res.Code, want["status"])
		}
	}
	if documented := slices.Sorted(maps.Keys(examples)); !slices.Equal(documented, slices.Sorted(slices.Values(keys))) {
		t.Errorf("contract problems %q,\ncatalog problems %q: want the same", documented, keys)
	}
}

func TestAProblemNamesTheClientsRequestID(t *testing.T) {
	t.Parallel()

	opts, _ := options(t, true)
	res := serve(t, httpapi.NewHandler(opts), http.MethodGet, "/nowhere", http.Header{"X-Request-Id": {"ticket-42"}})
	if got := decode(t, res)["instance"]; got != "tag:arnakoff.com,2026:request:ticket-42" {
		t.Errorf("instance = %v, want the client's request ID", got)
	}
}

func TestAProblemTypeIsUnderTheConfiguredBase(t *testing.T) {
	t.Parallel()

	opts, _ := options(t, true)
	opts.ProblemBase = "https://docs.example.org/errors/"
	res := serve(t, httpapi.NewHandler(opts), http.MethodGet, "/nowhere", nil)
	if got := decode(t, res)["type"]; got != "https://docs.example.org/errors/not_found" {
		t.Errorf("type = %v, want the page of not_found under the configured base", got)
	}
}

func TestErrorsMapToProblems(t *testing.T) {
	t.Parallel()

	_, errKind := domain.ParseKind("auto")
	_, errInput := domain.ParseInput("FR76 0000", domain.KindIBAN)
	for _, tt := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"unknown kind", errKind, http.StatusBadRequest, "invalid_parameter"},
		{"unrecognised input", domain.ErrUnrecognisedInput, http.StatusUnprocessableEntity, "invalid_input"},
		{"invalid input of a forced kind", errInput, http.StatusUnprocessableEntity, "invalid_input"},
		{"wrapped", fmt.Errorf("check: %w", domain.ErrUnrecognisedInput), http.StatusUnprocessableEntity, "invalid_input"},
		{"anything else", errors.New("disk full"), http.StatusInternalServerError, "internal_error"},
	} {
		status, code := httpapi.ProblemFor(tt.err)
		if status != tt.status || code != tt.code {
			t.Errorf("%s: problem = %d %s, want %d %s", tt.name, status, code, tt.status, tt.code)
		}
	}
}
