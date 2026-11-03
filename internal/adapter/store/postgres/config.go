// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrInsecure means a connection string lets a connection to a remote host
// go without TLS, or with TLS that does not verify the certificate and the
// host name: sslmode=verify-full is required beyond the local machine.
var ErrInsecure = errors.New("database connection without verified TLS")

// connectTimeout bounds the opening of a connection when the connection
// string sets no connect_timeout.
const connectTimeout = 5 * time.Second

// ParseConfig parses a PostgreSQL connection string, URL or key=value, and
// requires verified TLS (sslmode=verify-full) for every host but the local
// machine: a loopback address, "localhost" or a Unix socket. Its errors
// never quote the connection string, which may hold a password.
func ParseConfig(conn string, opts ...Option) (*pgxpool.Config, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	cfg, err := pgxpool.ParseConfig(conn)
	if err != nil {
		// pgx redacts the password on a best-effort basis only.
		return nil, errors.New("database connection string: cannot be parsed")
	}
	cc := cfg.ConnConfig
	endpoints := append([]*pgconn.FallbackConfig{{Host: cc.Host, TLSConfig: cc.TLSConfig}}, cc.Fallbacks...)
	for _, e := range endpoints {
		if !secure(e.Host, e.TLSConfig) {
			return nil, fmt.Errorf("%w: host %s", ErrInsecure, e.Host)
		}
	}
	if err := applyCA(o.ca, endpoints); err != nil {
		return nil, err
	}
	if cc.ConnectTimeout == 0 {
		cc.ConnectTimeout = connectTimeout
	}
	return cfg, nil
}

// secure reports whether a connection to host with tc is local or uses TLS
// verifying the certificate chain and the host name.
func secure(host string, tc *tls.Config) bool {
	if local(host) {
		return true
	}
	return tc != nil && !tc.InsecureSkipVerify && tc.ServerName != ""
}

// local reports whether host is a Unix socket directory, "localhost" or a
// loopback address.
func local(host string) bool {
	if strings.HasPrefix(host, "/") || host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
