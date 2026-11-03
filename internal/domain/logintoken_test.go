// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

func TestNewLoginTokenDrawsAURLSafeTokenItParsesBack(t *testing.T) {
	t.Parallel()
	tok, err := domain.NewLoginToken(bytes.NewReader(bytes.Repeat([]byte{0x11, 0x2a, 0x3f}, 30)))
	if err != nil {
		t.Fatal(err)
	}
	s := tok.String()
	if len(s) != domain.LoginTokenLength || strings.ContainsAny(s, "+/=#?&") {
		t.Fatalf("token %q: want %d URL-safe characters", s, domain.LoginTokenLength)
	}
	back, err := domain.ParseLoginToken(s)
	if err != nil || back != tok {
		t.Fatalf("ParseLoginToken(%q) = %v, %v; want the token back", s, back, err)
	}
}

func TestNewLoginTokenFailsWhenRandomIsShort(t *testing.T) {
	t.Parallel()
	if _, err := domain.NewLoginToken(bytes.NewReader(make([]byte, 3))); err == nil {
		t.Fatal("want an error from a short random source")
	}
}

func TestParseLoginTokenRefusesAMalformedToken(t *testing.T) {
	t.Parallel()
	valid := strings.Repeat("A", domain.LoginTokenLength)
	for _, s := range []string{
		"", valid[1:], valid + "A", strings.Repeat("+", domain.LoginTokenLength),
		valid[:domain.LoginTokenLength-1] + "=", "ark_live_" + valid[9:],
	} {
		if _, err := domain.ParseLoginToken(s); !errors.Is(err, domain.ErrInvalidLoginToken) {
			t.Errorf("ParseLoginToken(%q) error = %v, want ErrInvalidLoginToken", s, err)
		}
	}
}

func TestDigestLoginTokenDependsOnTheSecretAndTheToken(t *testing.T) {
	t.Parallel()
	a, _ := domain.ParseLoginToken(strings.Repeat("A", domain.LoginTokenLength))
	b, _ := domain.ParseLoginToken(strings.Repeat("B", domain.LoginTokenLength))
	s1, s2 := []byte("secret-one-0123456789abcdef"), []byte("secret-two-0123456789abcdef")
	again, _ := domain.ParseLoginToken(a.String())
	if domain.DigestLoginToken(s1, a) != domain.DigestLoginToken(s1, again) {
		t.Fatal("digest is not stable")
	}
	if domain.DigestLoginToken(s1, a) == domain.DigestLoginToken(s2, a) {
		t.Fatal("digest ignores the secret")
	}
	if domain.DigestLoginToken(s1, a) == domain.DigestLoginToken(s1, b) {
		t.Fatal("digest ignores the token")
	}
}

func TestNewLoginRequestExpiresAfterTheLinkLifetime(t *testing.T) {
	t.Parallel()
	to, _ := domain.ParseEmail("dev@exemple.fr")
	at := time.Date(2026, 11, 3, 10, 0, 0, 0, time.FixedZone("CET", 3600))
	var d domain.TokenDigest
	d[0] = 7
	r := domain.NewLoginRequest(d, to, at)
	want := domain.LoginRequest{
		Digest: d, Email: to, CreatedAt: at.UTC(), ExpiresAt: at.UTC().Add(domain.LoginLinkLifetime),
	}
	if r != want {
		t.Fatalf("NewLoginRequest = %+v, want %+v", r, want)
	}
	if domain.LoginLinkLifetime != 15*time.Minute {
		t.Fatalf("LoginLinkLifetime = %v, want the 15 minutes of #59", domain.LoginLinkLifetime)
	}
	if domain.LoginRequestRetention != 24*time.Hour {
		t.Fatalf("LoginRequestRetention = %v, want the 24 hours of ADR 0006", domain.LoginRequestRetention)
	}
}

func TestEmailIsZeroOnlyForTheZeroValue(t *testing.T) {
	t.Parallel()
	e, _ := domain.ParseEmail("dev@exemple.fr")
	if e.IsZero() || !(domain.Email{}).IsZero() {
		t.Fatal("IsZero must be true for the zero value only")
	}
}

func TestEmailAccountIgnoresCase(t *testing.T) {
	t.Parallel()
	a, _ := domain.ParseEmail("Jean.Dupont@Exemple.FR")
	b, _ := domain.ParseEmail("jean.dupont@exemple.fr")
	c, _ := domain.ParseEmail("jean.durand@exemple.fr")
	if a.Account() != "jean.dupont@exemple.fr" || a.Account() != b.Account() || a.Account() == c.Account() {
		t.Fatalf("Account() = %q, %q, %q; want the address in lower case", a.Account(), b.Account(), c.Account())
	}
}
