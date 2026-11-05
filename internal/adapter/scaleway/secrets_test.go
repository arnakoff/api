// SPDX-License-Identifier: AGPL-3.0-only

package scaleway_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/scaleway"
	"github.com/arnakoff/api/internal/app"
)

//nolint:gosec // G101: identifiers of secrets, not secrets.
const (
	accessSecretID = "19b0456f-43e4-40de-adfc-e081c5f8a801" // secret-create-version.json
	secretSecretID = "29b0456f-43e4-40de-adfc-e081c5f8a802"
	versionsPath   = "/secret-manager/v1beta1/regions/fr-par/secrets/"
)

// newKey is a key just created, as the rotator installs it.
var newKey = app.Credential{
	AccessKey: "SCWNEWKEYXXXXXXXXXXX",
	SecretKey: []byte("33333333-3333-3333-3333-333333333333"),
	ExpiresAt: time.Date(2027, 2, 2, 3, 10, 0, 0, time.UTC),
}

func TestSecretPairWritesBothHalvesOfTheKey(t *testing.T) {
	t.Parallel()
	f := newFakeAPI(t, map[string][]answer{
		"POST " + versionsPath + accessSecretID + "/versions": {{http.StatusOK, "secret-create-version.json"}},
		"POST " + versionsPath + secretSecretID + "/versions": {{http.StatusOK, "secret-create-version.json"}},
	})
	pair := scaleway.NewSecretPair(f.client(), accessSecretID, secretSecretID)

	if err := pair.InstallCredential(context.Background(), newKey); err != nil {
		t.Fatal(err)
	}

	// The SDK also reads /metadata after a write, to name the version.
	calls := slices.DeleteFunc(f.recorded(), func(c call) bool { return c.path == "/metadata" })
	if len(calls) != 2 {
		t.Fatalf("%d writes, want 2", len(calls))
	}
	want := map[string]string{accessSecretID: newKey.AccessKey, secretSecretID: string(newKey.SecretKey)}
	for _, c := range calls {
		var sent struct {
			Data            string `json:"data"`
			DisablePrevious bool   `json:"disable_previous"`
			Description     string `json:"description"`
		}
		if err := json.Unmarshal([]byte(c.body), &sent); err != nil {
			t.Fatal(err)
		}
		id := strings.TrimSuffix(strings.TrimPrefix(c.path, versionsPath), "/versions")
		data, _ := base64.StdEncoding.DecodeString(sent.Data)
		if string(data) != want[id] || !sent.DisablePrevious {
			t.Errorf("secret %s: wrote %q, disable previous %v", id, data, sent.DisablePrevious)
		}
		if !strings.Contains(sent.Description, newKey.AccessKey) || strings.Contains(sent.Description, string(newKey.SecretKey)) {
			t.Errorf("description %q, want the access key only", sent.Description)
		}
	}
}

func TestSecretPairReportsAFailedWrite(t *testing.T) {
	t.Parallel()
	f := newFakeAPI(t, map[string][]answer{
		"POST " + versionsPath + accessSecretID + "/versions": {{http.StatusOK, "secret-create-version.json"}},
		"POST " + versionsPath + secretSecretID + "/versions": {{http.StatusForbidden, ""}},
	})
	pair := scaleway.NewSecretPair(f.client(), accessSecretID, secretSecretID)

	err := pair.InstallCredential(context.Background(), newKey)

	if err == nil || !strings.Contains(err.Error(), secretSecretID) {
		t.Errorf("err = %v, want an error naming the secret", err)
	}
	if strings.Contains(err.Error(), string(newKey.SecretKey)) {
		t.Error("error quotes the secret key")
	}
}
