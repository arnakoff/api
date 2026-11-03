// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/registry/replay"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/config"
)

const corpusHeader = "input,type,label,added_at,note\n"

func TestTargetsAsksEveryCorpusInput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, dir, "scam.csv", corpusHeader+"armen.ltd,domain,scam,2026-10-08,\n")
	writeFile(t, dir, "legit.csv", corpusHeader+"120027016,siren,legit,2026-10-30,\n")
	var out strings.Builder

	err := targets(options{corpus: dir, targets: "http://127.0.0.1:8100"}, &out)

	want := "GET http://127.0.0.1:8100/v1/check?q=armen.ltd\n\nGET http://127.0.0.1:8100/v1/check?q=120027016\n\n"
	if err != nil || out.String() != want {
		t.Errorf("targets = %q, %v; want %q", out.String(), err, want)
	}
}

func TestTargetsRejectsAMissingCorpus(t *testing.T) {
	t.Parallel()

	err := targets(options{corpus: filepath.Join(t.TempDir(), "missing"), targets: "http://127.0.0.1:8100"}, &strings.Builder{})
	if err == nil {
		t.Error("targets of a missing corpus: want an error")
	}
}

func TestTargetsRejectsAMissingKeysFile(t *testing.T) {
	t.Parallel()

	opts := options{corpus: defaultCorpus, targets: "http://127.0.0.1:8100", keys: filepath.Join(t.TempDir(), "missing")}
	if err := targets(opts, &strings.Builder{}); err == nil {
		t.Error("targets with a missing keys file: want an error")
	}
}

func TestRunRequiresTrustedProxiesWithLiveKeys(t *testing.T) {
	t.Parallel()

	err := run(t.Context(), options{corpus: defaultCorpus}, config.Config{APIKeyEnv: "live"})
	if !errors.Is(err, config.ErrInvalid) {
		t.Errorf("run() without ARNAKOFF_TRUSTED_PROXIES error = %v, want ErrInvalid", err)
	}
}

func TestNewServiceReplaysTheTape(t *testing.T) {
	t.Parallel()

	tapeDir := t.TempDir()
	writeFile(t, tapeDir, "rdap.json", `{"armen.ltd": {"value": {"registered": "2025-01-01T00:00:00Z"}}}`)
	tape, err := replay.Load(tapeDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{DataDir: t.TempDir()}
	svc, err := newService(t.Context(), cfg, tape.Replayed(0))
	if err != nil {
		t.Fatalf("newService: %v", err)
	}
	in, err := domain.ParseInput("armen.ltd", domain.KindDomain)
	if err != nil {
		t.Fatal(err)
	}

	states := svc.Check.Check(t.Context(), in).Sources()
	i := slices.IndexFunc(states, func(s domain.SourceState) bool { return s.Source == "RDAP" })
	if i < 0 || states[i].Status != domain.SourceOK {
		t.Errorf("sources = %+v, want RDAP answered from the tape", states)
	}
}

func TestProfilesServesTheIndex(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/debug/pprof/", nil)

	profiles().Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /debug/pprof/ = %d, want 200", rec.Code)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
