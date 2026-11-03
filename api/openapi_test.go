// SPDX-License-Identifier: AGPL-3.0-only

package api_test

import (
	"bytes"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/codegen"

	"github.com/arnakoff/api/api"
	"github.com/arnakoff/api/internal/domain/signal"
)

func load(t *testing.T) *openapi3.T {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromData(api.Spec())
	if err != nil {
		t.Fatalf("load the contract: %v", err)
	}
	return doc
}

func TestSpecIsAValidOpenAPI321Document(t *testing.T) {
	t.Parallel()

	doc := load(t)
	if doc.OpenAPI != "3.2.1" {
		t.Errorf("openapi = %q, want 3.2.1 (docs/standards.md)", doc.OpenAPI)
	}
	if err := doc.Validate(t.Context(), openapi3.EnableExamplesValidation()); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

// TestSpecGeneratesGoCode proves oapi-codegen, the generator pinned in
// go.mod, reads the contract: types and a net/http server interface.
func TestSpecGeneratesGoCode(t *testing.T) {
	t.Parallel()

	opts := codegen.Configuration{
		PackageName:         "gen",
		Generate:            codegen.GenerateOptions{Models: true, StdHTTPServer: true},
		AllowRemoteRefsFrom: []string{},
	}
	code, err := codegen.Generate(load(t), opts.UpdateDefaults())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for _, want := range []string{"type Report struct", "type Signal struct", "type ServerInterface interface"} {
		if !strings.Contains(code, want) {
			t.Errorf("generated code lacks %q", want)
		}
	}
}

func TestSpecReturnsACopy(t *testing.T) {
	t.Parallel()

	b := api.Spec()
	want := bytes.Clone(b)
	clear(b)
	if !bytes.Equal(api.Spec(), want) {
		t.Error("clearing the returned slice changed the contract")
	}
}

func TestSpecListsEverySignalCode(t *testing.T) {
	t.Parallel()

	var want []string
	for _, d := range signal.Catalog() {
		want = append(want, d.Code)
	}

	var got []string
	for _, e := range load(t).Components.Schemas["Signal"].Value.Properties["code"].Value.Examples {
		got = append(got, e.(string))
	}
	slices.Sort(want)
	want = slices.Compact(want)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("Signal.code examples = %v, want the codes of the catalog %v", got, want)
	}
}

// TestSpecServesOverHTTPS checks the hosts the contract declares: at least
// one, each in HTTPS, without the /v1 prefix the paths already carry.
func TestSpecServesOverHTTPS(t *testing.T) {
	t.Parallel()

	servers := load(t).Servers
	if len(servers) == 0 {
		t.Fatal("servers is empty, want the public host")
	}
	for _, s := range servers {
		u, err := url.Parse(s.URL)
		if err != nil {
			t.Errorf("server %q: %v", s.URL, err)
			continue
		}
		if u.Scheme != "https" || u.Host == "" {
			t.Errorf("server %q, want an HTTPS host", s.URL)
		}
		if strings.Trim(u.Path, "/") != "" {
			t.Errorf("server %q has path %q, want none: the paths carry /v1", s.URL, u.Path)
		}
	}
}
