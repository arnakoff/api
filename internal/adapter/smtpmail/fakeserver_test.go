// SPDX-License-Identifier: AGPL-3.0-only

package smtpmail_test

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// fakeServer is an SMTP server over implicit TLS, enough for one mail per
// connection. It answers reject to the command starting with it, if any,
// and records the commands and the data it receives.
type fakeServer struct {
	addr   string
	roots  *x509.CertPool
	reject string

	mu       sync.Mutex
	commands []string
	data     string
}

// newFakeServer starts a fakeServer on the loopback address, with the
// certificate of httptest, valid for 127.0.0.1.
func newFakeServer(t *testing.T, reject string) *fakeServer {
	t.Helper()
	cert := httptest.NewUnstartedServer(nil)
	cert.StartTLS()
	cfg := &tls.Config{Certificates: cert.TLS.Certificates, MinVersion: tls.VersionTLS12}
	roots := x509.NewCertPool()
	roots.AddCert(cert.Certificate())
	cert.Close()
	ln, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	s := &fakeServer{addr: ln.Addr().String(), roots: roots, reject: reject}
	go s.serve(ln)
	return s
}

func (s *fakeServer) serve(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go s.session(conn)
	}
}

// session answers one client until QUIT.
func (s *fakeServer) session(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	r, w := bufio.NewReader(conn), bufio.NewWriter(conn)
	say := func(line string) { _, _ = w.WriteString(line + "\r\n"); _ = w.Flush() }
	say("220 fake ESMTP")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.TrimRight(line, "\r\n")
		s.record(cmd)
		if s.reject != "" && strings.HasPrefix(cmd, s.reject) {
			say(s.rejectReply())
			continue
		}
		if !s.answer(cmd, r, say) {
			return
		}
	}
}

// answer replies to cmd and reports whether the session goes on.
func (s *fakeServer) answer(cmd string, r *bufio.Reader, say func(string)) bool {
	switch {
	case strings.HasPrefix(cmd, "EHLO"):
		say("250-fake")
		say("250 AUTH PLAIN LOGIN")
	case strings.HasPrefix(cmd, "AUTH"):
		say("235 2.7.0 accepted")
	case strings.HasPrefix(cmd, "DATA"):
		say("354 go ahead")
		s.readData(r)
		say("250 2.0.0 queued")
	case strings.HasPrefix(cmd, "QUIT"):
		say("221 bye")
		return false
	default:
		say("250 ok")
	}
	return true
}

func (s *fakeServer) readData(r *bufio.Reader) {
	var b strings.Builder
	for {
		line, err := r.ReadString('\n')
		if err != nil || line == ".\r\n" {
			break
		}
		b.WriteString(strings.TrimPrefix(line, "."))
	}
	s.mu.Lock()
	s.data = b.String()
	s.mu.Unlock()
}

// rejectReply is the reply to the rejected command. It quotes the
// address, as real servers do.
func (s *fakeServer) rejectReply() string {
	return "550 5.1.1 <dev@exemple.fr>: mailbox unavailable"
}

func (s *fakeServer) record(cmd string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commands = append(s.commands, cmd)
}

func (s *fakeServer) received() ([]string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.commands...), s.data
}
