// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"crypto/x509"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrBadCA means the certificate authority given with WithCA holds no
// certificate, or comes with an sslrootcert of the connection string.
var ErrBadCA = errors.New("database certificate authority not usable")

// Option changes how ParseConfig reads a connection string.
type Option func(*options)

// options are what the Options set.
type options struct {
	ca []byte
}

// WithCA verifies the database against the certificates of pem only,
// instead of the system's: the authority of a managed database instance,
// which a container cannot mount as the file sslrootcert names. Empty, it
// changes nothing.
func WithCA(pem []byte) Option {
	return func(o *options) { o.ca = pem }
}

// applyCA makes the certificates of pem the only roots of every TLS
// endpoint. It refuses pem without a certificate, and an endpoint whose
// roots sslrootcert already set: two sources would leave one ignored.
func applyCA(pem []byte, endpoints []*pgconn.FallbackConfig) error {
	if len(pem) == 0 {
		return nil
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return errors.Join(ErrBadCA, errors.New("no PEM certificate"))
	}
	for _, e := range endpoints {
		if e.TLSConfig == nil {
			continue
		}
		if e.TLSConfig.RootCAs != nil {
			return errors.Join(ErrBadCA, errors.New("sslrootcert is also set, keep one"))
		}
		e.TLSConfig.RootCAs = pool
	}
	return nil
}
