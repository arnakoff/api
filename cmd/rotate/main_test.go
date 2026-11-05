// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/platform/config"
)

func TestRunNeedsTheRotationConfig(t *testing.T) {
	t.Parallel()
	err := run(context.Background(), config.Config{})

	if !errors.Is(err, config.ErrInvalid) || !strings.Contains(err.Error(), config.EnvRotateAppSelf) {
		t.Errorf("run() = %v, want ErrInvalid naming the variables", err)
	}
}

func TestNewRotationWiresEveryTarget(t *testing.T) {
	t.Parallel()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Rotate{
		AccessKey: "SCWROTATORXXXXXXXXXX",
		SecretKey: config.Secret("44444444-4444-4444-4444-444444444444"),
		ProjectID: "11111111-1111-1111-1111-111111111111",
		Bucket:    "arnakoff-snapshots",
		Apps: config.RotateApps{
			API:  "22222222-2222-2222-2222-222222222222",
			Sync: "33333333-3333-3333-3333-333333333333",
			CI:   "55555555-5555-5555-5555-555555555555",
			Self: "66666666-6666-6666-6666-666666666666",
		},
		APIContainerID: "77777777-7777-7777-7777-777777777777",
		SyncSecrets:    config.SecretIDs{Access: "88888888-8888-8888-8888-888888888888", Secret: "99999999-9999-9999-9999-999999999999"},
		SelfSecrets:    config.SecretIDs{Access: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Secret: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"},
		GitHubApp: config.GitHubApp{
			ClientID:       "Iv23liTESTCLIENTID",
			InstallationID: 4242,
			PrivateKey:     pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}),
		},
	}

	if _, err := newRotation(cfg); err != nil {
		t.Errorf("newRotation() = %v", err)
	}

	cfg.Apps.Self = cfg.Apps.API
	if _, err := newRotation(cfg); err == nil {
		t.Error("an application listed twice accepted")
	}
}
