// SPDX-License-Identifier: AGPL-3.0-only

package postgres_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/store/postgres"
)

// testCA returns a self-signed CA certificate, in PEM, and its pool: what
// a Scaleway Managed Database instance has of its own.
func testCA(t *testing.T) ([]byte, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test database CA"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pool
}

const remote = "postgres://api:s3cret@rw-0000.rdb.fr-par.scw.cloud:5432/arnakoff?sslmode=verify-full" //nolint:gosec // G101: a fake password, to prove errors hide it.

func TestParseConfigWithCAVerifiesTheDatabaseAgainstIt(t *testing.T) {
	t.Parallel()
	ca, pool := testCA(t)

	cfg, err := postgres.ParseConfig(remote, postgres.WithCA(ca))
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	tc := cfg.ConnConfig.TLSConfig
	if tc == nil || !tc.RootCAs.Equal(pool) || tc.InsecureSkipVerify || tc.ServerName != "rw-0000.rdb.fr-par.scw.cloud" {
		t.Errorf("TLS config = %+v, want the CA as the only root and the host verified", tc)
	}
}

func TestParseConfigWithoutCAKeepsTheSystemRoots(t *testing.T) {
	t.Parallel()

	for _, opts := range [][]postgres.Option{nil, {postgres.WithCA(nil)}} {
		cfg, err := postgres.ParseConfig(remote, opts...)
		if err != nil {
			t.Fatalf("ParseConfig() error = %v", err)
		}
		if tc := cfg.ConnConfig.TLSConfig; tc == nil || tc.RootCAs != nil {
			t.Errorf("TLS config = %+v, want the system roots (nil RootCAs)", tc)
		}
	}
}

func TestParseConfigWithCARefuses(t *testing.T) {
	t.Parallel()
	ca, _ := testCA(t)

	for name, tt := range map[string]struct {
		conn string
		ca   []byte
		want error
	}{
		"not PEM":          {remote, []byte("not a certificate"), postgres.ErrBadCA},
		"PEM without cert": {remote, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte{1}}), postgres.ErrBadCA},
		"with sslrootcert": {remote + "&sslrootcert=system", ca, postgres.ErrBadCA},
		"still insecure":   {strings.Replace(remote, "verify-full", "require", 1), ca, postgres.ErrInsecure},
	} {
		_, err := postgres.ParseConfig(tt.conn, postgres.WithCA(tt.ca))
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: error = %v, want %v", name, err, tt.want)
		}
		if err != nil && strings.Contains(err.Error(), "s3cret") {
			t.Errorf("%s: error quotes the password: %v", name, err)
		}
	}
}
