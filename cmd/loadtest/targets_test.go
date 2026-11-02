// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"crypto/rand"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

func TestWriteTargetsAsksTheCheckOfEachCase(t *testing.T) {
	t.Parallel()

	site, err := domain.ParseDomain("armen.ltd")
	if err != nil {
		t.Fatal(err)
	}
	email, err := domain.ParseEmail("prénom.nom@paris-majeste.com")
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder

	err = writeTargets(&out, "http://127.0.0.1:8100", []app.Case{{Input: site}, {Input: email}}, nil)

	want := "GET http://127.0.0.1:8100/v1/check?q=armen.ltd\n\n" +
		"GET http://127.0.0.1:8100/v1/check?q=pr%C3%A9nom.nom%40paris-majeste.com\n\n"
	if err != nil || out.String() != want {
		t.Errorf("writeTargets = %q, %v; want %q", out.String(), err, want)
	}
}

func TestWriteTargetsSpreadsTheKeys(t *testing.T) {
	t.Parallel()

	keys := []domain.APIKey{newKey(t), newKey(t)}
	var cases []app.Case
	for _, s := range []string{"a.fr", "b.fr", "c.fr"} {
		d, err := domain.ParseDomain(s)
		if err != nil {
			t.Fatal(err)
		}
		cases = append(cases, app.Case{Input: d})
	}
	var out strings.Builder

	err := writeTargets(&out, "http://127.0.0.1:8100", cases, keys)

	want := "GET http://127.0.0.1:8100/v1/check?q=a.fr\nAuthorization: Bearer " + keys[0].String() + "\n\n" +
		"GET http://127.0.0.1:8100/v1/check?q=b.fr\nAuthorization: Bearer " + keys[1].String() + "\n\n" +
		"GET http://127.0.0.1:8100/v1/check?q=c.fr\nAuthorization: Bearer " + keys[0].String() + "\n\n"
	if err != nil || out.String() != want {
		t.Errorf("writeTargets = %q, %v; want %q", out.String(), err, want)
	}
}

func TestReadKeys(t *testing.T) {
	t.Parallel()

	k1, k2 := newKey(t), newKey(t)
	got, err := readKeys(strings.NewReader(k1.String() + "\n\n  " + k2.String() + "  \n"))

	if err != nil || len(got) != 2 || got[0] != k1 || got[1] != k2 {
		t.Errorf("readKeys = %v, %v; want [%s %s]", got, err, k1.Hint(), k2.Hint())
	}
}

func TestReadKeysRejects(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"no key":          "\n",
		"a malformed key": "ark_test_nope\n",
		"a header":        newKey(t).String() + "\r\nX-Injected: 1\n",
	}
	for name, in := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := readKeys(strings.NewReader(in)); err == nil {
				t.Errorf("readKeys(%q): want an error", in)
			}
		})
	}
}

func newKey(t *testing.T) domain.APIKey {
	t.Helper()
	k, err := domain.NewAPIKey(domain.KeyTest, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return k
}
