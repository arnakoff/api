// SPDX-License-Identifier: AGPL-3.0-only

package scaleway_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
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
	return newContainerSecretsOf(f, accessVar)
}

// newContainerSecretsOf returns the ContainerSecrets of the variable
// access, none when empty, and secretVar.
func newContainerSecretsOf(f *fakeAPI, access string) *scaleway.ContainerSecrets {
	s, err := scaleway.NewContainerSecrets(f.client(), scaleway.ContainerSecretsOptions{
		ContainerID: containerID,
		AccessVar:   access,
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
		"no container":       {AccessVar: accessVar, SecretVar: secretVar},
		"no secret variable": {ContainerID: containerID, AccessVar: accessVar},
		"negative timeout":   {ContainerID: containerID, AccessVar: accessVar, SecretVar: secretVar, Timeout: -time.Second},
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
	tests := []struct {
		name   string
		access string
		want   []secretVarValue
	}{
		{"both halves", accessVar, []secretVarValue{{accessVar, newKey.AccessKey}, {secretVar, string(newKey.SecretKey)}}},
		{"secret half only", "", []secretVarValue{{secretVar, string(newKey.SecretKey)}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFakeAPI(t, map[string][]answer{
				"PATCH " + containerPath: {{http.StatusOK, "container-update.json"}},
				"GET " + containerPath:   {{http.StatusOK, "container-update.json"}, {http.StatusOK, "container-ready.json"}},
			})

			if err := newContainerSecretsOf(f, tt.access).InstallCredential(context.Background(), newKey); err != nil {
				t.Fatal(err)
			}

			calls := f.recorded()
			if calls[0].method != http.MethodPatch || calls[len(calls)-1].method != http.MethodGet || len(calls) != 3 {
				t.Fatalf("calls %+v, want a PATCH then GETs until ready", calls)
			}
			if got := sentSecrets(t, calls[0].body); !slices.Equal(got, tt.want) {
				t.Errorf("secrets %+v, want %+v only", got, tt.want)
			}
		})
	}
}

// secretVarValue is a secret environment variable sent to the API.
type secretVarValue struct{ Key, Value string }

// sentSecrets returns the secrets of the PATCH body, failing t when it
// sets anything else.
func sentSecrets(t *testing.T, body string) []secretVarValue {
	t.Helper()
	var sent map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &sent); err != nil {
		t.Fatal(err)
	}
	var secrets []secretVarValue
	if err := json.Unmarshal(sent["secret_environment_variables"], &secrets); err != nil {
		t.Fatal(err)
	}
	for field := range sent {
		if field != "secret_environment_variables" && !strings.HasPrefix(string(sent[field]), `"unknown_`) {
			t.Errorf("field %s = %s sent, want the secrets only", field, sent[field])
		}
	}
	return secrets
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
