// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// linkBase is the page of the site the login links of the tests open.
const linkBase = "https://arnakoff.com/cle"

// keyRequestsFixture is a KeyRequests over in-memory fakes, at t0.
type keyRequestsFixture struct {
	reqs   *app.KeyRequests
	logins *loginStore
	keys   *keyStore
	mail   *mailer
	clock  *clock
}

func newKeyRequests(t *testing.T) keyRequestsFixture {
	t.Helper()
	f := keyRequestsFixture{logins: &loginStore{}, keys: &keyStore{}, mail: &mailer{}, clock: &clock{now: t0}}
	f.reqs = app.NewKeyRequests(app.KeyRequestDeps{
		Logins: f.logins, Keys: f.keys, Mailer: f.mail, Secret: keySecret,
		Environment: domain.KeyLive, Clock: f.clock, Random: rand.Reader, LinkBase: linkBase,
	})
	return f
}

func mustEmail(t *testing.T, s string) domain.Email {
	t.Helper()
	e, err := domain.ParseEmail(s)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// tokenOf returns the token of the link in body.
func tokenOf(t *testing.T, body string) string {
	t.Helper()
	_, after, ok := strings.Cut(body, linkBase+"#")
	if !ok {
		t.Fatalf("no link to %s# in %q", linkBase, body)
	}
	tok, _, _ := strings.Cut(after, "\n")
	return tok
}

func TestRequestLinkStoresTheDigestAndMailsTheLink(t *testing.T) {
	t.Parallel()
	f := newKeyRequests(t)
	to := mustEmail(t, "Dev@Exemple.fr")

	if err := f.reqs.RequestLink(t.Context(), to); err != nil {
		t.Fatal(err)
	}

	mails := f.mail.mails()
	if len(mails) != 1 || mails[0].to != to {
		t.Fatalf("mails = %+v, want one to %v", mails, to.Redacted())
	}
	if mails[0].subject != "Votre lien pour obtenir une clé API Arnakoff" {
		t.Errorf("subject = %q", mails[0].subject)
	}
	tok := tokenOf(t, mails[0].body)
	parsed, err := domain.ParseLoginToken(tok)
	if err != nil {
		t.Fatalf("link token %q: %v", tok, err)
	}
	reqs := f.logins.requests()
	want := domain.NewLoginRequest(domain.DigestLoginToken(keySecret, parsed), to, t0)
	if len(reqs) != 1 || reqs[0].LoginRequest != want {
		t.Fatalf("stored = %+v, want %+v", reqs, want)
	}
}

func TestRequestLinkMailsTheWordingTheOwnerApproved(t *testing.T) {
	t.Parallel()
	f := newKeyRequests(t)
	if err := f.reqs.RequestLink(t.Context(), mustEmail(t, "dev@exemple.fr")); err != nil {
		t.Fatal(err)
	}
	body := f.mail.mails()[0].body
	tok := tokenOf(t, body)
	want := "Bonjour,\n\n" +
		"Pour obtenir votre clé API Arnakoff, ouvrez ce lien dans les 15 minutes :\n" +
		linkBase + "#" + tok + "\n\n" +
		"Ce lien ne sert qu'une fois. Si vous n'avez rien demandé, ignorez ce message : aucune clé ne sera créée.\n\n" +
		"Arnakoff — https://arnakoff.com\n"
	if body != want {
		t.Errorf("body =\n%s\nwant\n%s", body, want)
	}
}

func TestRequestLinkFailsWithoutMailingWhenTheStoreFails(t *testing.T) {
	t.Parallel()
	f := newKeyRequests(t)
	down := errors.New("database down")
	f.logins.err = down

	err := f.reqs.RequestLink(t.Context(), mustEmail(t, "dev@exemple.fr"))
	if !errors.Is(err, down) {
		t.Fatalf("error = %v, want it to wrap %v", err, down)
	}
	if strings.Contains(err.Error(), "dev@") {
		t.Errorf("error %q quotes the address", err)
	}
	if n := len(f.mail.mails()); n != 0 {
		t.Errorf("%d mails sent, want none", n)
	}
}

func TestRequestLinkWrapsTheMailerErrorWithoutTheAddress(t *testing.T) {
	t.Parallel()
	f := newKeyRequests(t)
	down := errors.New("smtp down")
	f.mail.err = down

	err := f.reqs.RequestLink(t.Context(), mustEmail(t, "dev@exemple.fr"))
	if !errors.Is(err, down) || strings.Contains(err.Error(), "dev@") {
		t.Fatalf("error = %v, want it to wrap %v without the address", err, down)
	}
}

func TestRequestLinkFailsWhenRandomFails(t *testing.T) {
	t.Parallel()
	f := newKeyRequests(t)
	f.reqs = app.NewKeyRequests(app.KeyRequestDeps{
		Logins: f.logins, Keys: f.keys, Mailer: f.mail, Secret: keySecret, Environment: domain.KeyLive,
		Clock: f.clock, Random: iotest.ErrReader(errors.New("no entropy")), LinkBase: linkBase,
	})
	if err := f.reqs.RequestLink(t.Context(), mustEmail(t, "dev@exemple.fr")); err == nil {
		t.Fatal("want an error")
	}
}

// requested sends a link to addr and returns its token.
func (f keyRequestsFixture) requested(t *testing.T, addr string) string {
	t.Helper()
	if err := f.reqs.RequestLink(t.Context(), mustEmail(t, addr)); err != nil {
		t.Fatal(err)
	}
	mails := f.mail.mails()
	return tokenOf(t, mails[len(mails)-1].body)
}

func TestRedeemIssuesAKeyToTheAccountOnce(t *testing.T) {
	t.Parallel()
	f := newKeyRequests(t)
	tok := f.requested(t, "dev@exemple.fr")
	f.clock.now = t0.Add(domain.LoginLinkLifetime - time.Second)

	key, rec, err := f.reqs.Redeem(t.Context(), tok)
	if err != nil {
		t.Fatal(err)
	}
	want := domain.KeyRecord{
		ID: 1, Digest: domain.DigestAPIKey(keySecret, key), Environment: domain.KeyLive,
		Label: app.SelfServiceKeyLabel, Hint: key.Hint(), Email: mustEmail(t, "dev@exemple.fr"),
		CreatedAt: f.clock.now,
	}
	if rec != want || key.Environment() != domain.KeyLive {
		t.Fatalf("Redeem() = %+v, want %+v", rec, want)
	}
	if _, _, err := f.reqs.Redeem(t.Context(), tok); !errors.Is(err, app.ErrLoginLinkInvalid) {
		t.Fatalf("second Redeem() error = %v, want ErrLoginLinkInvalid", err)
	}
}

func TestRedeemRevokesThePreviousKeyOfTheAccount(t *testing.T) {
	t.Parallel()
	f := newKeyRequests(t)
	_, first, err := f.reqs.Redeem(t.Context(), f.requested(t, "dev@exemple.fr"))
	if err != nil {
		t.Fatal(err)
	}
	f.clock.now = t0.Add(time.Hour)
	_, second, err := f.reqs.Redeem(t.Context(), f.requested(t, "DEV@exemple.fr"))
	if err != nil {
		t.Fatal(err)
	}
	recs, _ := f.keys.ListKeys(t.Context())
	if len(recs) != 2 || recs[0].ID != first.ID || recs[0].RevokedAt != f.clock.now || !recs[1].Active() || recs[1].ID != second.ID {
		t.Fatalf("keys = %+v, want the first revoked when the second was issued", recs)
	}
}

func TestRedeemRefusesABadToken(t *testing.T) {
	t.Parallel()
	f := newKeyRequests(t)
	f.requested(t, "dev@exemple.fr")
	expired := newKeyRequests(t)
	old := expired.requested(t, "dev@exemple.fr")
	expired.clock.now = t0.Add(domain.LoginLinkLifetime)

	for name, c := range map[string]struct {
		f   keyRequestsFixture
		tok string
	}{
		"malformed": {f, "not a token"},
		"unknown":   {f, strings.Repeat("Z", domain.LoginTokenLength)},
		"expired":   {expired, old},
	} {
		if _, _, err := c.f.reqs.Redeem(t.Context(), c.tok); !errors.Is(err, app.ErrLoginLinkInvalid) {
			t.Errorf("%s: error = %v, want ErrLoginLinkInvalid", name, err)
		}
	}
	if recs, _ := f.keys.ListKeys(t.Context()); len(recs) != 0 {
		t.Errorf("keys issued = %d, want none", len(recs))
	}
}

func TestRedeemWrapsTheStoreErrors(t *testing.T) {
	t.Parallel()
	down := errors.New("database down")

	logins := newKeyRequests(t)
	tok := logins.requested(t, "dev@exemple.fr")
	logins.logins.err = down
	if _, _, err := logins.reqs.Redeem(t.Context(), tok); !errors.Is(err, down) {
		t.Errorf("login store down: error = %v, want it to wrap %v", err, down)
	}

	keys := newKeyRequests(t)
	tok = keys.requested(t, "dev@exemple.fr")
	keys.keys.err = down
	if _, _, err := keys.reqs.Redeem(t.Context(), tok); !errors.Is(err, down) || strings.Contains(err.Error(), "dev@") {
		t.Errorf("key store down: error = %v, want it to wrap %v without the address", err, down)
	}
}
