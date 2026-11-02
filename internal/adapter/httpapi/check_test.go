// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/adapter/httpapi/contracttest"
	"github.com/arnakoff/api/internal/domain"
)

// contract is the public contract, loaded once for every test.
var contract = sync.OnceValues(contracttest.Load)

// paris is a zone other than UTC, to prove times are written in UTC.
var paris = time.FixedZone("CET", 3600)

// blacklisted is the report of the contract's ReportBlacklisted example,
// with times in another zone than UTC.
func blacklisted() domain.Report {
	return domain.NewReport("armen.ltd", time.Date(2026, 11, 2, 9, 25, 53, 0, paris), []domain.Signal{{
		Code:       "amf_blacklist",
		Level:      domain.LevelAlert,
		Message:    "Figure sur la liste noire de l'AMF (catégorie Usurpation) depuis le 08/10/2026.",
		Source:     "AMF",
		SourceURL:  "https://www.amf-france.org/fr/espace-epargnants/proteger-son-epargne/listes-noires-et-mises-en-garde",
		ObservedAt: time.Date(2026, 11, 2, 9, 14, 42, 0, paris),
	}}, []domain.SourceState{
		{Source: "AMF", Status: domain.SourceOK},
		{Source: "ACPR", Status: domain.SourceUnavailable},
		{Source: "ARCEP", Status: domain.SourceNotApplicable},
	})
}

// check sends GET target to a handler whose check answers report, holds
// the exchange to the contract and returns the response and the fake. A
// 400 answers a request the contract refuses: only the answer is held.
func check(t *testing.T, report domain.Report, target string) (*httptest.ResponseRecorder, *checker) {
	t.Helper()
	opts, _ := options(t, true)
	c := &checker{report: report}
	opts.Check = c
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	httpapi.NewHandler(opts).ServeHTTP(rec, req)
	ct, err := contract()
	if err != nil {
		t.Fatal(err)
	}
	hold := ct.Check
	if rec.Code == http.StatusBadRequest {
		hold = ct.CheckRefused
	}
	if err := hold(t.Context(), req, rec.Result()); err != nil {
		t.Errorf("GET %s breaks the contract: %v", target, err)
	}
	return rec, c
}

func TestCheckAnswersTheReport(t *testing.T) {
	t.Parallel()

	res, _ := check(t, blacklisted(), "/v1/check?q=armen.ltd")

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	if ct := res.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	want := `{"input":"armen.ltd","type":"domain","checked_at":"2026-11-02T08:25:53Z","summary":null,` +
		`"signals":[{"code":"amf_blacklist","level":"alerte",` +
		`"message":"Figure sur la liste noire de l'AMF (catégorie Usurpation) depuis le 08/10/2026.",` +
		`"source":"AMF","source_url":"https://www.amf-france.org/fr/espace-epargnants/proteger-son-epargne/listes-noires-et-mises-en-garde",` +
		`"observed_at":"2026-11-02T08:14:42Z"}],` +
		`"sources":[{"source":"AMF","status":"ok"},{"source":"ACPR","status":"indisponible"},` +
		`{"source":"ARCEP","status":"non_applicable"}]}` + "\n"
	if got := res.Body.String(); got != want {
		t.Errorf("body:\n got %s\nwant %s", got, want)
	}
}

func TestCheckIsNeverCached(t *testing.T) {
	t.Parallel()

	// The input of a report may be personal data.
	res, _ := check(t, blacklisted(), "/v1/check?q=armen.ltd")
	if got := res.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
}

func TestCheckWithoutSignalSaysSoWithoutVouching(t *testing.T) {
	t.Parallel()

	report := domain.NewReport("552100554", time.Date(2026, 11, 2, 8, 25, 53, 0, time.UTC), nil, nil)
	res, _ := check(t, report, "/v1/check?q=552100554")

	body := decode(t, res)
	if body["summary"] != "Aucun signal connu dans nos sources" {
		t.Errorf("summary = %v, want « Aucun signal connu dans nos sources »", body["summary"])
	}
	if s, ok := body["signals"].([]any); !ok || len(s) != 0 {
		t.Errorf("signals = %v, want an empty array", body["signals"])
	}
	if s, ok := body["sources"].([]any); !ok || len(s) != 0 {
		t.Errorf("sources = %v, want an empty array", body["sources"])
	}
}

func TestCheckWritesTheAdviceOnlyWhenThereIsOne(t *testing.T) {
	t.Parallel()

	s := domain.Signal{
		Code: "domain_lookalike", Level: domain.LevelAttention, Message: "Ressemble.",
		Advice: "Tape toi-même l'adresse officielle plutôt que de suivre un lien reçu.",
		Source: "Domaines officiels", SourceURL: "https://www.ameli.fr/assure/mentions-legales-cgu",
		ObservedAt: time.Date(2026, 11, 2, 8, 25, 53, 0, time.UTC),
	}
	bare := s
	bare.Code, bare.Advice = "domain_recent", ""
	report := domain.NewReport("ameli-remboursement.com", s.ObservedAt, []domain.Signal{s, bare}, nil)
	res, _ := check(t, report, "/v1/check?q=ameli-remboursement.com")

	var body struct{ Signals []map[string]any }
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if got := body.Signals[0]["advice"]; got != s.Advice {
		t.Errorf("advice = %v, want %q", got, s.Advice)
	}
	if _, ok := body.Signals[1]["advice"]; ok {
		t.Errorf("signal without advice has the field: %v", body.Signals[1])
	}
}

func TestCheckReadsTheInputAndItsType(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		target string
		kind   domain.Kind
		input  string
	}{
		{"/v1/check?q=armen.ltd", domain.KindDomain, "armen.ltd"},
		{"/v1/check?q=https%3A%2F%2Fwww.Armen.ltd%2Fdepot%3Fx%3D1", domain.KindDomain, "www.armen.ltd"},
		{"/v1/check?q=+120027016+", domain.KindSIREN, "120027016"},
		// Fourteen digits are detected as a SIRET first; type forces a phone.
		{"/v1/check?q=00442079460958&type=phone", domain.KindPhone, "+442079460958"},
	} {
		res, c := check(t, blacklisted(), tt.target)
		if res.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", tt.target, res.Code)
			continue
		}
		asked := c.inputs()
		if len(asked) != 1 || asked[0].Kind() != tt.kind {
			t.Errorf("%s: checked %v, want one input of kind %v", tt.target, asked, tt.kind)
			continue
		}
		if asked[0].String() != tt.input {
			t.Errorf("%s: checked %q, want %q", tt.target, asked[0], tt.input)
		}
		if got := decode(t, res)["type"]; got != tt.kind.String() {
			t.Errorf("%s: type = %v, want %v", tt.target, got, tt.kind)
		}
	}
}

func TestCheckRefusesABadRequestWithoutChecking(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		target string
		status int
		code   string
	}{
		{"/v1/check", http.StatusBadRequest, "missing_parameter"},
		{"/v1/check?type=domain", http.StatusBadRequest, "missing_parameter"},
		{"/v1/check?q=", http.StatusBadRequest, "missing_parameter"},
		{"/v1/check?q=armen.ltd&type=url", http.StatusBadRequest, "invalid_parameter"},
		{"/v1/check?q=armen.ltd&type=auto", http.StatusBadRequest, "invalid_parameter"},
		{"/v1/check?q=armen.ltd&type=", http.StatusBadRequest, "invalid_parameter"},
		{"/v1/check?q=+++", http.StatusUnprocessableEntity, "invalid_input"},
		{"/v1/check?q=FR76+0000&type=iban", http.StatusUnprocessableEntity, "invalid_input"},
		{"/v1/check?q=not+a+domain", http.StatusUnprocessableEntity, "invalid_input"},
	} {
		res, c := check(t, blacklisted(), tt.target)
		if res.Code != tt.status {
			t.Errorf("%s: status = %d, want %d", tt.target, res.Code, tt.status)
		}
		if got := decode(t, res)["code"]; got != tt.code {
			t.Errorf("%s: code = %v, want %s", tt.target, got, tt.code)
		}
		if asked := c.inputs(); len(asked) != 0 {
			t.Errorf("%s: checked %v, want no check", tt.target, asked)
		}
	}
}

func TestCheckNamesTheTypeAnInvalidInputWasForcedInto(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct{ target, detail string }{
		{"/v1/check?q=a+b&type=domain", "L'entrée n'est ni un domaine ni une URL valide."},
		{"/v1/check?q=a+b&type=email", "L'entrée n'est pas une adresse e-mail valide."},
		{"/v1/check?q=a+b&type=phone", "L'entrée n'est pas un numéro de téléphone valide."},
		{"/v1/check?q=FR76+0000&type=iban", "L'entrée n'est pas un IBAN valide."},
		{"/v1/check?q=123&type=siren", "L'entrée n'est pas un SIREN valide."},
		{"/v1/check?q=123&type=siret", "L'entrée n'est pas un SIRET valide."},
		// Detected, the input was of no kind: the detail names them all.
		{"/v1/check?q=a+b", "L'entrée n'est ni un domaine, ni une URL, ni une adresse e-mail, " +
			"ni un numéro de téléphone, ni un IBAN, ni un SIREN, ni un SIRET."},
	} {
		res, _ := check(t, blacklisted(), tt.target)
		if got := decode(t, res)["detail"]; got != tt.detail {
			t.Errorf("%s: detail = %v, want %s", tt.target, got, tt.detail)
		}
	}
}

// BenchmarkCheck measures GET /v1/check through the whole handler, the
// middleware included, around a check that answers at once: the cost the
// API adds to the check itself.
func BenchmarkCheck(b *testing.B) {
	h := httpapi.NewHandler(httpapi.Options{
		Ready: readiness(true), Check: &benchChecker{report: blacklisted()},
		Logger: slog.New(slog.DiscardHandler), ProblemBase: problemBase,
	})
	req := httptest.NewRequestWithContext(b.Context(), http.MethodGet, "/v1/check?q=armen.ltd", nil)
	b.ReportAllocs()
	for b.Loop() {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("status = %d", rec.Code)
		}
	}
}

// benchChecker answers report and records nothing: the lock and the
// growing record of checker would be measured with the handler.
type benchChecker struct{ report domain.Report }

func (c *benchChecker) Check(context.Context, domain.Input) domain.Report { return c.report }

func (c *benchChecker) CheckAll(ctx context.Context, ins []domain.Input) []domain.Report {
	return checkEach(ctx, c.Check, ins)
}
