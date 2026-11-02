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

// contractProblems returns the contract's Problem… examples, by code: the
// list of codes arnakoff/web documents, one page each.
func contractProblems(t *testing.T) map[string]map[string]any {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromData(api.Spec())
	if err != nil {
		t.Fatal(err)
	}
	byCode := make(map[string]map[string]any)
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
		byCode[p["code"].(string)] = p
	}
	return byCode
}

func TestEveryProblemIsAContractExample(t *testing.T) {
	t.Parallel()

	examples := contractProblems(t)
	opts, _ := options(t, true)
	var codes []string
	for _, p := range httpapi.Catalog {
		res := serve(t, httpapi.Wrap(opts, httpapi.Answer(opts, p)), http.MethodGet, "/v1/check", nil)
		got := decode(t, res)
		code, _ := got["code"].(string)
		codes = append(codes, code)
		want, ok := examples[code]
		if !ok {
			t.Errorf("problem %s has no Problem… example in the contract", code)
			continue
		}
		want["instance"] = "tag:arnakoff.com,2026:request:GENERATED"
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("problem %s:\n got %v\nwant %v", code, got, want)
		}
		if res.Code != int(want["status"].(float64)) {
			t.Errorf("problem %s: status = %d, want %v", code, res.Code, want["status"])
		}
	}
	if missing := slices.Sorted(maps.Keys(examples)); !slices.Equal(missing, slices.Sorted(slices.Values(codes))) {
		t.Errorf("contract codes %v, catalog codes %v: want the same", missing, codes)
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
