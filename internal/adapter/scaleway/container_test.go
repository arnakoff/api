// SPDX-License-Identifier: AGPL-3.0-only

package scaleway_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/scaleway"
)

const (
	containerID   = "0b6ea900-e5ac-4145-947b-785fe8f7143b" // container-update.json
	containerPath = "/containers/v1beta1/regions/fr-par/containers/" + containerID
	accessVar     = "ARNAKOFF_SNAPSHOT_BUCKET_ACCESS_KEY"
	secretVar     = "ARNAKOFF_SNAPSHOT_BUCKET_SECRET_KEY" //nolint:gosec // G101: the name of a variable.
)

func newContainerSecrets(f *fakeAPI) *scaleway.ContainerSecrets {
	s, err := scaleway.NewContainerSecrets(f.client(), scaleway.ContainerSecretsOptions{
		ContainerID: containerID,
		AccessVar:   accessVar,
		SecretVar:   secretVar,
		Timeout:     time.Second,
		Interval:    time.Millisecond,
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return s
}

func TestNewContainerSecretsRefusesBadOptions(t *testing.T) {
	t.Parallel()
	f := newFakeAPI(t, nil)
	cases := map[string]scaleway.ContainerSecretsOptions{
		"no container":     {AccessVar: accessVar, SecretVar: secretVar},
		"no variable":      {ContainerID: containerID, AccessVar: accessVar},
		"negative timeout": {ContainerID: containerID, AccessVar: accessVar, SecretVar: secretVar, Timeout: -time.Second},
	}
	for name, opts := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := scaleway.NewContainerSecrets(f.client(), opts); !errors.Is(err, scaleway.ErrInvalidOptions) {
				t.Errorf("NewContainerSecrets() = %v, want ErrInvalidOptions", err)
			}
		})
	}
}

func TestContainerSecretsSetsTheKeyThenWaitsUntilReady(t *testing.T) {
	t.Parallel()
	f := newFakeAPI(t, map[string][]answer{
		"PATCH " + containerPath: {{http.StatusOK, "container-update.json"}},
		"GET " + containerPath:   {{http.StatusOK, "container-update.json"}, {http.StatusOK, "container-ready.json"}},
	})

	if err := newContainerSecrets(f).InstallCredential(context.Background(), newKey); err != nil {
		t.Fatal(err)
	}

	calls := f.recorded()
	if calls[0].method != http.MethodPatch || calls[len(calls)-1].method != http.MethodGet || len(calls) != 3 {
		t.Fatalf("calls %+v, want a PATCH then GETs until ready", calls)
	}
	var sent map[string]json.RawMessage
	if err := json.Unmarshal([]byte(calls[0].body), &sent); err != nil {
		t.Fatal(err)
	}
	var secrets []struct{ Key, Value string }
	if err := json.Unmarshal(sent["secret_environment_variables"], &secrets); err != nil {
		t.Fatal(err)
	}
	want := []struct{ Key, Value string }{{accessVar, newKey.AccessKey}, {secretVar, string(newKey.SecretKey)}}
	if len(secrets) != 2 || secrets[0] != want[0] || secrets[1] != want[1] {
		t.Errorf("secrets %+v, want the two halves of the key only", secrets)
	}
	for field := range sent {
		if field != "secret_environment_variables" && !strings.HasPrefix(string(sent[field]), `"unknown_`) {
			t.Errorf("field %s = %s sent, want the secrets only", field, sent[field])
		}
	}
}

func TestContainerSecretsFailsWhenTheContainerDoesNotGetReady(t *testing.T) {
	t.Parallel()
	f := newFakeAPI(t, map[string][]answer{
		"PATCH " + containerPath: {{http.StatusOK, "container-update.json"}},
		"GET " + containerPath:   {{http.StatusOK, "container-error.json"}},
	})

	err := newContainerSecrets(f).InstallCredential(context.Background(), newKey)

	if err == nil || !strings.Contains(err.Error(), "error") {
		t.Errorf("err = %v, want the error status", err)
	}
}

func TestContainerSecretsReportsAFailedUpdate(t *testing.T) {
	t.Parallel()
	f := newFakeAPI(t, map[string][]answer{
		"PATCH " + containerPath: {{http.StatusForbidden, ""}},
	})

	err := newContainerSecrets(f).InstallCredential(context.Background(), newKey)

	if err == nil || strings.Contains(err.Error(), string(newKey.SecretKey)) {
		t.Errorf("err = %v, want an error without the secret key", err)
	}
	if len(f.recorded()) != 1 {
		t.Error("waited on a container that was not updated")
	}
}
