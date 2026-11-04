// SPDX-License-Identifier: AGPL-3.0-only

package smtpmail

import (
	"bytes"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"strings"
	"time"
)

// senderName is the display name of the From field.
const senderName = "Arnakoff"

// header holds the fields of a mail.
type header struct {
	from, to, subject string
	date              time.Time
	id                string
}

// compose returns the RFC 5322 mail of h and body: plain text in UTF-8,
// quoted-printable, lines ended with CRLF, marked as sent by a program
// (RFC 3834) so that auto-responders stay quiet.
func compose(h header, body string) ([]byte, error) {
	var b bytes.Buffer
	fields := [][2]string{
		{"From", mime.QEncoding.Encode("utf-8", senderName) + " <" + h.from + ">"},
		{"To", "<" + h.to + ">"},
		{"Subject", mime.QEncoding.Encode("utf-8", h.subject)},
		{"Date", h.date.Format(time.RFC1123Z)},
		{"Message-ID", h.id},
		{"MIME-Version", "1.0"},
		{"Content-Type", "text/plain; charset=utf-8"},
		{"Content-Transfer-Encoding", "quoted-printable"},
		{"Auto-Submitted", "auto-generated"},
	}
	for _, f := range fields {
		fmt.Fprintf(&b, "%s: %s\r\n", f[0], f[1])
	}
	b.WriteString("\r\n")
	w := quotedprintable.NewWriter(&b)
	if _, err := w.Write([]byte(strings.ReplaceAll(body, "\n", "\r\n"))); err != nil {
		return nil, fmt.Errorf("smtp body: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("smtp body: %w", err)
	}
	return b.Bytes(), nil
}
