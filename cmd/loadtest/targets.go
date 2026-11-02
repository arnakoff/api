// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// writeTargets writes to w, in the vegeta http format, one GET /v1/check
// of base per case, carrying the keys in turn, or no key if there is none.
func writeTargets(w io.Writer, base string, cases []app.Case, keys []domain.APIKey) error {
	bw := bufio.NewWriter(w)
	for i, c := range cases {
		_, _ = bw.WriteString("GET " + base + "/v1/check?q=" + url.QueryEscape(c.Input.String()) + "\n")
		if len(keys) > 0 {
			_, _ = bw.WriteString("Authorization: Bearer " + keys[i%len(keys)].String() + "\n")
		}
		_, _ = bw.WriteString("\n")
	}
	return bw.Flush()
}

// readKeys returns the API keys of r, one a line, blank lines skipped.
// Each must be a well-formed key: nothing else reaches a header.
func readKeys(r io.Reader) ([]domain.APIKey, error) {
	var keys []domain.APIKey
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		k, err := domain.ParseAPIKey(line)
		if err != nil {
			return nil, fmt.Errorf("keys line %d: %w", n, err)
		}
		keys = append(keys, k)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read keys: %w", err)
	}
	if len(keys) == 0 {
		return nil, errors.New("no API key in the keys file")
	}
	return keys, nil
}
