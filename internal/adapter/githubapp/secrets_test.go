// SPDX-License-Identifier: AGPL-3.0-only

package githubapp_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/nacl/box"

	"github.com/arnakoff/api/internal/adapter/githubapp"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/platform/httpclient"
	"github.com/arnakoff/api/internal/platform/httpclient/httpclienttest"
)

// Compile-time check: EnvironmentSecrets puts a key in service.
var _ app.CredentialInstaller = (*githubapp.EnvironmentSecrets)(nil)

const (
	clientID       = "Iv23liTESTCLIENTID"
	installationID = 4242
	// token is an installation token. The documentation gives no example
	// answer: the fields of tokenAnswer follow its schema (« à vérifier »
	// on a real answer at the bootstrap, #308).
	token       = "ghs_TESTTOKEN" //nolint:gosec // G101: a fake token of the tests.
	tokenAnswer = `{"token":"` + token + `","expires_at":"2026-11-05T04:10:00Z","permissions":{"environments":"write"},"repository_selection":"selected"}`
)

var (
	now = time.Date(2026, 11, 5, 3, 10, 0, 0, time.UTC)
	key = app.Credential{AccessKey: "SCWNEWKEYXXXXXXXXXXX", SecretKey: []byte("33333333-3333-3333-3333-333333333333")}
)

// put is a secret the fake GitHub received.
type put struct {
	EncryptedValue string `json:"encrypted_value"`
	KeyID          string `json:"key_id"`
	auth           string
}

// fakeGitHub answers the three calls of EnvironmentSecrets.
type fakeGitHub struct {
	t         *testing.T
	publicKey string // answer of the public key
	mu        sync.Mutex
	tokenReq  *http.Request
	tokenBody string
	puts      map[string]put
	failPut   int
}

func (f *fakeGitHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	body, _ := io.ReadAll(r.Body)
	if r.Header.Get("X-GitHub-Api-Version") != githubapp.APIVersion || r.Header.Get("Accept") != "application/vnd.github+json" {
		f.t.Errorf("%s %s: headers %v", r.Method, r.URL.Path, r.Header)
	}
	const env = "/repos/arnakoff/api/environments/production/secrets/"
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/app/installations/4242/access_tokens":
		f.tokenReq, f.tokenBody = r, string(body)
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, tokenAnswer)
	case r.Method == http.MethodGet && r.URL.Path == env+"public-key":
		_, _ = io.WriteString(w, f.publicKey)
	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, env):
		if f.failPut != 0 {
			w.WriteHeader(f.failPut)
			return
		}
		var p put
		if err := json.Unmarshal(body, &p); err != nil {
			f.t.Error(err)
		}
		p.auth = r.Header.Get("Authorization")
		f.puts[strings.TrimPrefix(r.URL.Path, env)] = p
		w.WriteHeader(http.StatusNoContent)
	default:
		f.t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}
}

// setup returns EnvironmentSecrets on a fake GitHub that answers
// publicKey, the fake and the private key of the App.
func setup(t *testing.T, publicKey string) (*githubapp.EnvironmentSecrets, *fakeGitHub, *rsa.PrivateKey) {
	t.Helper()
	appKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeGitHub{t: t, publicKey: publicKey, puts: map[string]put{}}
	srv := httptest.NewTLSServer(f)
	t.Cleanup(srv.Close)
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(appKey)})
	//nolint:gosec // G101: a key drawn for this test.
	s, err := githubapp.NewEnvironmentSecrets(githubapp.Options{
		HTTP:           httpclienttest.NewClient(t, srv, httpclient.Config{}),
		APIURL:         srv.URL,
		ClientID:       clientID,
		InstallationID: installationID,
		PrivateKey:     pemKey,
		Owner:          "arnakoff",
		Repo:           "api",
		Environment:    "production",
		AccessName:     "SCW_ACCESS_KEY",
		SecretName:     "SCW_SECRET_KEY",
		Now:            func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	return s, f, appKey
}

func realPublicKey(t *testing.T) string {
	t.Helper()
	// Answer of GitHub for arnakoff/api, environment production, read on
	// 05/11/2026 with gh api: a public key.
	b, err := os.ReadFile("../../../testdata/github/environment-public-key.json")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestEnvironmentSecretsWritesBothHalvesOfTheKey(t *testing.T) {
	t.Parallel()
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	answer := `{"key_id":"568250167242549743","key":"` + base64.StdEncoding.EncodeToString(pub[:]) + `"}`
	s, f, _ := setup(t, answer)

	if err := s.InstallCredential(context.Background(), key); err != nil {
		t.Fatal(err)
	}

	want := map[string]string{"SCW_ACCESS_KEY": key.AccessKey, "SCW_SECRET_KEY": string(key.SecretKey)}
	for name, value := range want {
		p, ok := f.puts[name]
		if !ok {
			t.Fatalf("%s not written", name)
		}
		sealed, err := base64.StdEncoding.DecodeString(p.EncryptedValue)
		if err != nil {
			t.Fatal(err)
		}
		opened, ok := box.OpenAnonymous(nil, sealed, pub, priv)
		if !ok || string(opened) != value || p.KeyID != "568250167242549743" {
			t.Errorf("%s: opened %q (%v), key id %s", name, opened, ok, p.KeyID)
		}
		if p.auth != "Bearer "+token {
			t.Errorf("%s written with %q, want the installation token", name, p.auth)
		}
	}
}

func TestEnvironmentSecretsSealsForTheRealPublicKey(t *testing.T) {
	t.Parallel()
	s, f, _ := setup(t, realPublicKey(t))

	if err := s.InstallCredential(context.Background(), key); err != nil {
		t.Fatal(err)
	}

	p := f.puts["SCW_SECRET_KEY"]
	sealed, _ := base64.StdEncoding.DecodeString(p.EncryptedValue)
	if p.KeyID != "3380204578043523366" || len(sealed) != len(key.SecretKey)+box.AnonymousOverhead {
		t.Errorf("key id %s, %d bytes sealed", p.KeyID, len(sealed))
	}
	if strings.Contains(p.EncryptedValue, string(key.SecretKey)) {
		t.Error("secret key sent in clear")
	}
}

func TestEnvironmentSecretsAsksANarrowToken(t *testing.T) {
	t.Parallel()
	s, f, appKey := setup(t, realPublicKey(t))

	if err := s.InstallCredential(context.Background(), key); err != nil {
		t.Fatal(err)
	}

	var body struct {
		Repositories []string          `json:"repositories"`
		Permissions  map[string]string `json:"permissions"`
	}
	if err := json.Unmarshal([]byte(f.tokenBody), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Repositories) != 1 || body.Repositories[0] != "api" || len(body.Permissions) != 1 || body.Permissions["environments"] != "write" {
		t.Errorf("token asked with %s, want arnakoff/api and environments: write only", f.tokenBody)
	}
	checkJWT(t, f.tokenReq.Header.Get("Authorization"), &appKey.PublicKey)
}

// checkJWT checks that auth is "Bearer " and a JWT of the App signed with
// pub, issued 60 s before now and valid 10 minutes at most.
func checkJWT(t *testing.T, auth string, pub *rsa.PublicKey) {
	t.Helper()
	jwt, ok := strings.CutPrefix(auth, "Bearer ")
	parts := strings.Split(jwt, ".")
	if !ok || len(parts) != 3 {
		t.Fatalf("authorization %q is not a bearer JWT", auth)
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], sig); err != nil {
		t.Errorf("signature: %v", err)
	}
	var header struct{ Alg, Typ string }
	var claims struct {
		Iat, Exp int64
		Iss      string
	}
	decode(t, parts[0], &header)
	decode(t, parts[1], &claims)
	if header.Alg != "RS256" || claims.Iss != clientID || claims.Iat != now.Unix()-60 || claims.Exp > now.Unix()+600 || claims.Exp <= now.Unix() {
		t.Errorf("header %+v, claims %+v", header, claims)
	}
}

func decode(t *testing.T, part string, v any) {
	t.Helper()
	b, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatal(err)
	}
}

func TestEnvironmentSecretsReportsAFailedWrite(t *testing.T) {
	t.Parallel()
	s, f, _ := setup(t, realPublicKey(t))
	f.failPut = http.StatusForbidden

	err := s.InstallCredential(context.Background(), key)

	var se *httpclient.StatusError
	if !errors.As(err, &se) || se.StatusCode != http.StatusForbidden || !strings.Contains(err.Error(), "SCW_ACCESS_KEY") {
		t.Errorf("err = %v, want a status error naming the secret", err)
	}
	if strings.Contains(err.Error(), token) || strings.Contains(err.Error(), string(key.SecretKey)) {
		t.Error("error quotes a secret")
	}
}

func TestNewEnvironmentSecretsRefusesABadPrivateKey(t *testing.T) {
	t.Parallel()
	_, err := githubapp.NewEnvironmentSecrets(githubapp.Options{PrivateKey: []byte("not a key")})

	if err == nil || strings.Contains(err.Error(), "not a key") {
		t.Errorf("err = %v, want an error without the key", err)
	}
}
