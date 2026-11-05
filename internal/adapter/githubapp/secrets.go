// SPDX-License-Identifier: AGPL-3.0-only

package githubapp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"golang.org/x/crypto/nacl/box"

	"github.com/arnakoff/api/internal/app"
)

// ErrInvalidOptions means NewEnvironmentSecrets was given Options it
// cannot use. Its errors never quote the private key.
var ErrInvalidOptions = errors.New("invalid GitHub App options")

// Options configures EnvironmentSecrets.
type Options struct {
	// HTTP sends the requests. Required.
	HTTP Sender

	// APIURL is the base URL of the REST API; "https://" + Host when
	// empty.
	APIURL string

	// ClientID identifies the App, as GitHub recommends for the issuer of
	// its JWT; InstallationID is its installation on Owner.
	ClientID       string
	InstallationID int64

	// PrivateKey is a private key of the App, in PEM PKCS#1. Required.
	PrivateKey []byte

	// Owner, Repo and Environment name the GitHub environment whose
	// secrets AccessName and SecretName receive the two halves of a key.
	Owner, Repo, Environment string
	AccessName, SecretName   string

	// Now tells the time the JWT is issued at; time.Now when nil.
	Now func() time.Time
}

// EnvironmentSecrets writes a key to two secrets of a GitHub environment,
// as a GitHub App (ADR 0007). It implements app.CredentialInstaller.
type EnvironmentSecrets struct {
	opts Options
	key  *rsa.PrivateKey
}

// NewEnvironmentSecrets returns EnvironmentSecrets of opts, or
// ErrInvalidOptions.
func NewEnvironmentSecrets(opts Options) (*EnvironmentSecrets, error) {
	key, err := parsePrivateKey(opts.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidOptions, err)
	}
	if opts.HTTP == nil {
		return nil, fmt.Errorf("%w: no HTTP client", ErrInvalidOptions)
	}
	if opts.APIURL == "" {
		opts.APIURL = "https://" + Host
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &EnvironmentSecrets{opts: opts, key: key}, nil
}

// InstallCredential writes the two halves of c to their secrets, each
// sealed for the public key of the environment. The workflows that read
// them get c from their next run.
func (s *EnvironmentSecrets) InstallCredential(ctx context.Context, c app.Credential) error {
	token, err := s.installationToken(ctx)
	if err != nil {
		return err
	}
	pub, err := s.publicKey(ctx, token)
	if err != nil {
		return err
	}
	if err := s.write(ctx, token, pub, s.opts.AccessName, []byte(c.AccessKey)); err != nil {
		return err
	}
	return s.write(ctx, token, pub, s.opts.SecretName, c.SecretKey)
}

// installationToken returns a token of the installation restricted to the
// repository and to writing its environments: it expires within the hour.
func (s *EnvironmentSecrets) installationToken(ctx context.Context) (string, error) {
	jwt, err := appJWT(s.key, s.opts.ClientID, s.opts.Now())
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(map[string]any{
		"repositories": []string{s.opts.Repo},
		"permissions":  map[string]string{"environments": "write"},
	})
	if err != nil {
		return "", err
	}
	var answer struct {
		Token string `json:"token"`
	}
	path := fmt.Sprintf("/app/installations/%d/access_tokens", s.opts.InstallationID)
	if err := s.call(ctx, request{"POST", path, jwt, body}, &answer); err != nil {
		return "", fmt.Errorf("installation token: %w", err)
	}
	if answer.Token == "" {
		return "", errors.New("installation token: none in the answer")
	}
	return answer.Token, nil
}

// envKey is the public key of an environment.
type envKey struct {
	id  string
	key [32]byte
}

// publicKey returns the public key the secrets of the environment are
// sealed for.
func (s *EnvironmentSecrets) publicKey(ctx context.Context, token string) (envKey, error) {
	var answer struct {
		KeyID string `json:"key_id"`
		Key   string `json:"key"`
	}
	if err := s.call(ctx, request{"GET", s.secretsPath() + "public-key", token, nil}, &answer); err != nil {
		return envKey{}, fmt.Errorf("public key of the environment: %w", err)
	}
	raw, err := base64.StdEncoding.DecodeString(answer.Key)
	if err != nil || len(raw) != 32 || answer.KeyID == "" {
		return envKey{}, errors.New("public key of the environment: not a key id and a base64 key of 32 bytes")
	}
	k := envKey{id: answer.KeyID}
	copy(k.key[:], raw)
	return k, nil
}

// write sets the secret name of the environment to value, in a sealed box
// for pub, as « Encrypting secrets for the REST API » shows.
func (s *EnvironmentSecrets) write(ctx context.Context, token string, pub envKey, name string, value []byte) error {
	sealed, err := box.SealAnonymous(nil, value, &pub.key, rand.Reader)
	if err != nil {
		return fmt.Errorf("seal the secret %s: %w", name, err)
	}
	body, err := json.Marshal(map[string]string{
		"encrypted_value": base64.StdEncoding.EncodeToString(sealed),
		"key_id":          pub.id,
	})
	if err != nil {
		return err
	}
	if err := s.call(ctx, request{"PUT", s.secretsPath() + url.PathEscape(name), token, body}, nil); err != nil {
		return fmt.Errorf("write the secret %s: %w", name, err)
	}
	return nil
}

// secretsPath is the path of the secrets of the environment, with a
// trailing slash.
func (s *EnvironmentSecrets) secretsPath() string {
	return "/repos/" + url.PathEscape(s.opts.Owner) + "/" + url.PathEscape(s.opts.Repo) +
		"/environments/" + url.PathEscape(s.opts.Environment) + "/secrets/"
}
