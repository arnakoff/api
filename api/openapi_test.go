// SPDX-License-Identifier: AGPL-3.0-only

package api_test

import (
	"bytes"
	"os"
	"regexp"
	"slices"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/arnakoff/api/api"
)

func load(t *testing.T) *openapi3.T {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromData(api.Spec())
	if err != nil {
		t.Fatalf("load the contract: %v", err)
	}
	return doc
}

func TestSpecIsAValidOpenAPI312Document(t *testing.T) {
	t.Parallel()

	doc := load(t)
	if doc.OpenAPI != "3.1.2" {
		t.Errorf("openapi = %q, want 3.1.2 (docs/standards.md)", doc.OpenAPI)
	}
	if err := doc.Validate(t.Context(), openapi3.EnableExamplesValidation()); err != nil {
		t.Errorf("Validate: %v", err)
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

// documentedCode matches a code in the table of docs/signals.md.
var documentedCode = regexp.MustCompile("(?m)^\\| `([a-z][a-z0-9_]*)` \\|")

func TestSpecListsEverySignalCode(t *testing.T) {
	t.Parallel()

	page, err := os.ReadFile("../docs/signals.md")
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, m := range documentedCode.FindAllSubmatch(page, -1) {
		want = append(want, string(m[1]))
	}
	if len(want) == 0 {
		t.Fatal("no signal code found in docs/signals.md")
	}

	var got []string
	for _, e := range load(t).Components.Schemas["Signal"].Value.Properties["code"].Value.Examples {
		got = append(got, e.(string))
	}
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("Signal.code examples = %v, want the codes of docs/signals.md %v", got, want)
	}
}
