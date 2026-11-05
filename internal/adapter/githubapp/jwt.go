// SPDX-License-Identifier: AGPL-3.0-only

package githubapp

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"strconv"
	"time"
)

// JWT timing, from « Generating a JSON Web Token (JWT) for a GitHub App »
// (docs.github.com, read on 05/11/2026): iat 60 seconds in the past
// against clock drift, exp no more than 10 minutes ahead.
const (
	jwtBackdate = 60 * time.Second
	jwtLifetime = 9 * time.Minute
)

// parsePrivateKey parses the private key of a GitHub App, a PEM PKCS#1
// RSAPrivateKey as GitHub hands it out (« Managing private keys for GitHub
// Apps »). Its errors never quote the input.
func parsePrivateKey(b []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(b)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("private key: not a PEM RSA PRIVATE KEY block")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("private key: not a PKCS#1 RSA key")
	}
	return key, nil
}

// appJWT returns a JWT of the App issuer, signed RS256 with key, valid
// from now - jwtBackdate to now + jwtLifetime.
func appJWT(key *rsa.PrivateKey, issuer string, now time.Time) (string, error) {
	claims, err := json.Marshal(struct {
		Iat int64  `json:"iat"`
		Exp int64  `json:"exp"`
		Iss string `json:"iss"`
	}{now.Add(-jwtBackdate).Unix(), now.Add(jwtLifetime).Unix(), issuer})
	if err != nil {
		return "", err
	}
	signed := segment([]byte(`{"alg":"RS256","typ":"JWT"}`)) + "." + segment(claims)
	sum := sha256.Sum256([]byte(signed))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return "", errors.New("sign the JWT of the App " + strconv.Quote(issuer))
	}
	return signed + "." + segment(sig), nil
}

// segment is b in base64url without padding, as a JWT part.
func segment(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}
