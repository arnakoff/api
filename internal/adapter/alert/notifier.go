// SPDX-License-Identifier: AGPL-3.0-only

package alert

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/platform/httpclient"
)

// Poster sends a body to a URL; *httpclient.Client is one. Its errors must
// not quote the URL.
type Poster interface {
	Post(ctx context.Context, rawURL, contentType string, body []byte) error
}

// Notifier logs each alert and posts it to a webhook. It implements
// app.Alerter and is safe for concurrent use.
type Notifier struct {
	logger  *slog.Logger
	poster  Poster
	webhook string
}

// LogOnly returns a Notifier that only logs to logger.
func LogOnly(logger *slog.Logger) *Notifier {
	return &Notifier{logger: logger}
}

// New returns a Notifier logging to logger and posting through poster to
// webhook.
func New(logger *slog.Logger, poster Poster, webhook string) *Notifier {
	return &Notifier{logger: logger, poster: poster, webhook: webhook}
}

// ForWebhook returns a Notifier logging to logger and posting to webhook, an
// HTTPS URL, through an HTTP client allowed to reach its host only. Its
// errors never quote webhook.
func ForWebhook(logger *slog.Logger, webhook string) (*Notifier, error) {
	u, err := url.Parse(webhook)
	if err != nil {
		return nil, errors.New("alert webhook: not a URL")
	}
	client, err := httpclient.New(httpclient.Config{AllowedHosts: []string{u.Host}})
	if err != nil {
		return nil, fmt.Errorf("alert webhook: %w", err)
	}
	if u.Scheme != "https" {
		return nil, fmt.Errorf("alert webhook: %w", httpclient.ErrURLNotAllowed)
	}
	return New(logger, client, webhook), nil
}

// Alert logs a and posts it to the webhook, if any. A webhook failure is
// logged, without the URL.
func (n *Notifier) Alert(ctx context.Context, a app.Alert) {
	at := a.At.UTC().Format(time.RFC3339)
	n.logger.ErrorContext(ctx, "source update failed", "source", string(a.Source), "at", at, "error", a.Err.Error())
	if n.poster == nil {
		return
	}
	if err := n.poster.Post(ctx, n.webhook, "application/json", body(a, at)); err != nil {
		n.logger.ErrorContext(ctx, "alert webhook failed", "source", string(a.Source), "error", err.Error())
	}
}

// payload is the webhook body; see the package doc.
type payload struct {
	Text   string `json:"text"`
	Source string `json:"source"`
	At     string `json:"at"`
	Error  string `json:"error"`
}

func body(a app.Alert, at string) []byte {
	msg := a.Err.Error()
	b, _ := json.Marshal(payload{ // A struct of strings always marshals.
		Text:   "arnakoff: " + string(a.Source) + " update failed: " + msg,
		Source: string(a.Source),
		At:     at,
		Error:  msg,
	})
	return b
}
