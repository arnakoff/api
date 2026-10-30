// SPDX-License-Identifier: AGPL-3.0-only

package alert_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/alert"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/platform/httpclient"
	"github.com/arnakoff/api/internal/platform/httpclient/httpclienttest"
)

var failed = app.Alert{
	Source: "AMF",
	At:     time.Date(2026, 10, 30, 6, 0, 0, 0, time.UTC),
	Err:    errors.New("sync AMF: snapshot AMF: shrunk by more than 20 %"),
}

// logLines returns a logger writing JSON lines and a function decoding them.
func logLines(t *testing.T) (*slog.Logger, func() []map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	return logger, func() []map[string]any {
		var lines []map[string]any
		for line := range strings.Lines(buf.String()) {
			var m map[string]any
			if err := json.Unmarshal([]byte(line), &m); err != nil {
				t.Fatalf("log line %q: %v", line, err)
			}
			lines = append(lines, m)
		}
		return lines
	}
}

func TestNotifierLogsAnErrorLine(t *testing.T) {
	t.Parallel()
	logger, lines := logLines(t)

	alert.LogOnly(logger).Alert(t.Context(), failed)

	got := lines()
	if len(got) != 1 {
		t.Fatalf("log = %v, want one line", got)
	}
	want := map[string]any{"level": "ERROR", "msg": "source update failed", "source": "AMF", "at": "2026-10-30T06:00:00Z", "error": failed.Err.Error()}
	for k, v := range want {
		if got[0][k] != v {
			t.Errorf("log %s = %v, want %v", k, got[0][k], v)
		}
	}
}

func TestNotifierPostsToTheWebhook(t *testing.T) {
	t.Parallel()
	type received struct {
		contentType string
		body        []byte
	}
	got := make(chan received, 1)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got <- received{r.Header.Get("Content-Type"), b}
	}))
	t.Cleanup(srv.Close)
	logger, lines := logLines(t)
	client := httpclienttest.NewClient(t, srv, httpclient.Config{})

	alert.New(logger, client, srv.URL+"/hook").Alert(t.Context(), failed)

	r := <-got
	if r.contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", r.contentType)
	}
	var body map[string]string
	if err := json.Unmarshal(r.body, &body); err != nil {
		t.Fatalf("body %s: %v", r.body, err)
	}
	want := map[string]string{
		"text":   "arnakoff: AMF update failed: " + failed.Err.Error(),
		"source": "AMF",
		"at":     "2026-10-30T06:00:00Z",
		"error":  failed.Err.Error(),
	}
	for k, v := range want {
		if body[k] != v {
			t.Errorf("body %s = %q, want %q", k, body[k], v)
		}
	}
	if n := len(lines()); n != 1 {
		t.Errorf("logged %d lines, want only the alert", n)
	}
}

func TestNotifierLogsAWebhookFailureWithoutItsURL(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	logger, lines := logLines(t)
	client := httpclienttest.NewClient(t, srv, httpclient.Config{})

	alert.New(logger, client, srv.URL+"/hook/T0K3N").Alert(t.Context(), failed)

	got := lines()
	if len(got) != 2 || got[1]["msg"] != "alert webhook failed" || got[1]["level"] != "ERROR" {
		t.Fatalf("log = %v, want the alert then the webhook failure", got)
	}
	if s := got[1]["error"].(string); !strings.Contains(s, "404") || strings.Contains(s, "T0K3N") {
		t.Errorf("webhook failure error = %q, want the status and not the URL", s)
	}
}

func TestForWebhookChecksTheURL(t *testing.T) {
	t.Parallel()
	if _, err := alert.ForWebhook(slog.Default(), "https://hooks.example.net/T0K3N"); err != nil {
		t.Errorf("ForWebhook(valid) = %v", err)
	}
	for _, raw := range []string{"http://hooks.example.net/T0K3N", "https:///T0K3N", "%zzT0K3N"} {
		_, err := alert.ForWebhook(slog.Default(), raw)
		if err == nil {
			t.Errorf("ForWebhook(%q) succeeded", raw)
			continue
		}
		if strings.Contains(err.Error(), "T0K3N") {
			t.Errorf("ForWebhook error %q quotes the URL", err)
		}
	}
}
