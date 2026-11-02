// SPDX-License-Identifier: AGPL-3.0-only

// Package hosting reads the reference list of shared platforms that host
// their users' pages under a path of their own domain, shipped with the
// binary (platforms.csv, documented in docs/sources.md).
package hosting

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/csvfile"
)

// ErrFormat means a reference list departs from its format. The error
// locates the line.
var ErrFormat = errors.New("invalid hosting platforms file")

// header is the first line of a reference list.
var header = []string{"domain", "proof"}

//go:embed platforms.csv
var shipped []byte

// Load reads the reference list shipped with the binary.
func Load() (domain.Platforms, error) {
	return Parse(bytes.NewReader(shipped))
}

// Parse reads a reference list: a CSV file with header, one platform per
// row, its registrable domain and an HTTPS page of a user on it, the proof
// that it hosts pages under a path. A malformed file or proof is rejected
// with ErrFormat, a malformed domain with domain.ErrInvalidDomain.
func Parse(r io.Reader) (domain.Platforms, error) {
	cr, err := csvfile.OpenSeparated(r, ',', header, ErrFormat)
	if err != nil {
		return domain.Platforms{}, err
	}
	var domains []string
	err = cr.Each(func(rec []string) error {
		if len(rec) != len(header) {
			return fmt.Errorf("%w: %d fields, want %d", ErrFormat, len(rec), len(header))
		}
		if err := checkProof(rec[0], rec[1]); err != nil {
			return err
		}
		domains = append(domains, rec[0])
		return nil
	})
	if err != nil {
		return domain.Platforms{}, err
	}
	return domain.NewPlatforms(domains)
}

// checkProof checks that proof is an HTTPS page on the platform
// registrable, a normalised registrable domain.
func checkProof(registrable, proof string) error {
	if _, err := domain.ParseRegistrable(registrable); err != nil {
		return err
	}
	d, err := domain.ParseDomain(proof)
	if err != nil || !strings.HasPrefix(proof, "https://") || d.Registrable() != registrable {
		return fmt.Errorf("%w: proof %q is not an HTTPS page on %s", ErrFormat, proof, registrable)
	}
	return nil
}
