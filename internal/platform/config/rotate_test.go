// SPDX-License-Identifier: AGPL-3.0-only

package config_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/platform/config"
)

const (
	rotateSecretKey  = "44444444-4444-4444-4444-444444444444"
	rotateAppPEMBody = "-----BEGIN RSA PRIVATE KEY-----\nTESTPEM\n-----END RSA PRIVATE KEY-----\n" //nolint:gosec // G101: not a key, a PEM frame around a marker.
)

// rotateVars are a complete configuration of the key rotation job.
func rotateVars() map[string]string {
	return map[string]string{
		"ARNAKOFF_LOG_KEY":                      validKey,
		"ARNAKOFF_ROTATE_ACCESS_KEY":            "SCWROTATORXXXXXXXXXX",
		"ARNAKOFF_ROTATE_SECRET_KEY":            rotateSecretKey,
		"ARNAKOFF_ROTATE_ORGANIZATION_ID":       "cccccccc-cccc-cccc-cccc-cccccccccccc",
		"ARNAKOFF_ROTATE_PROJECT_ID":            "11111111-1111-1111-1111-111111111111",
		"ARNAKOFF_ROTATE_BUCKET":                "arnakoff-snapshots",
		"ARNAKOFF_ROTATE_APP_API":               "22222222-2222-2222-2222-222222222222",
		"ARNAKOFF_ROTATE_APP_SYNC":              "33333333-3333-3333-3333-333333333333",
		"ARNAKOFF_ROTATE_APP_CI":                "55555555-5555-5555-5555-555555555555",
		"ARNAKOFF_ROTATE_APP_SELF":              "66666666-6666-6666-6666-666666666666",
		"ARNAKOFF_ROTATE_API_CONTAINER_ID":      "77777777-7777-7777-7777-777777777777",
		"ARNAKOFF_ROTATE_SYNC_ACCESS_SECRET_ID": "88888888-8888-8888-8888-888888888888",
		"ARNAKOFF_ROTATE_SYNC_SECRET_SECRET_ID": "99999999-9999-9999-9999-999999999999",
		"ARNAKOFF_ROTATE_SELF_ACCESS_SECRET_ID": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"ARNAKOFF_ROTATE_SELF_SECRET_SECRET_ID": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		"ARNAKOFF_GITHUB_APP_CLIENT_ID":         "Iv23liTESTCLIENTID",
		"ARNAKOFF_GITHUB_APP_INSTALLATION_ID":   "4242",
		"ARNAKOFF_GITHUB_APP_PRIVATE_KEY":       rotateAppPEMBody,
	}
}

func TestLoadReadsTheRotation(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(rotateVars()))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.RequireRotate(); err != nil {
		t.Fatalf("RequireRotate: %v", err)
	}
	r := cfg.Rotate
	if r.AccessKey != "SCWROTATORXXXXXXXXXX" || string(r.SecretKey.Bytes()) != rotateSecretKey ||
		r.Apps.CI != "55555555-5555-5555-5555-555555555555" || r.GitHubApp.InstallationID != 4242 ||
		string(r.GitHubApp.PrivateKey.Bytes()) != rotateAppPEMBody || r.Bucket != "arnakoff-snapshots" || r.OrganizationID != "cccccccc-cccc-cccc-cccc-cccccccccccc" {
		t.Errorf("Rotate = %+v", r)
	}
	if s := fmt.Sprintf("%v %+v", r, cfg); strings.Contains(s, rotateSecretKey) || strings.Contains(s, "TESTPEM") {
		t.Errorf("formatted config shows a secret: %s", s)
	}
}

func TestRequireRotateNeedsEveryVariable(t *testing.T) {
	t.Parallel()
	for name := range rotateVars() {
		if name == "ARNAKOFF_LOG_KEY" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			vars := rotateVars()
			delete(vars, name)
			cfg, err := config.Load(env(vars))
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if err := cfg.RequireRotate(); !errors.Is(err, config.ErrInvalid) || !strings.Contains(err.Error(), name) {
				t.Errorf("RequireRotate() = %v, want ErrInvalid naming %s", err, name)
			}
		})
	}
}

func TestLoadRejectsABadRotation(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"ARNAKOFF_ROTATE_APP_API":             "not-a-uuid",
		"ARNAKOFF_ROTATE_PROJECT_ID":          "11111111-1111-1111-1111-11111111111G",
		"ARNAKOFF_ROTATE_BUCKET":              "Not_A_Bucket",
		"ARNAKOFF_GITHUB_APP_INSTALLATION_ID": "-3",
	}
	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			vars := rotateVars()
			vars[name] = value
			if _, err := config.Load(env(vars)); !errors.Is(err, config.ErrInvalid) || !strings.Contains(err.Error(), name) {
				t.Errorf("Load() = %v, want ErrInvalid naming %s", err, name)
			}
		})
	}
}
