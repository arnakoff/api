// SPDX-License-Identifier: AGPL-3.0-only

package scaleway_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/scaleway"
)

// Applications of the fixtures.
const (
	listedApp  = "a94c31ce-e268-4f19-a5d5-5f5075c46d9b" // iam-list-api-keys.json
	createdApp = "fe101044-4da6-4bdd-b133-fc9745b47b87" // iam-create-api-key.json, iam-get-api-key.json
	otherApp   = "0b8cc93b-dfd2-4427-b674-8d27affb0334"
	projectID  = "105bdce1-64c0-48ab-899d-868455867ecf"
	orgID      = "11111111-1111-1111-1111-111111111111" // the organization of the recorded list request
)

// scope is the organization and project of the tests.
var scope = scaleway.Scope{OrganizationID: orgID, ProjectID: projectID}

func newCredentials(f *fakeAPI) *scaleway.Credentials {
	return scaleway.NewCredentials(f.client(), scope, []string{listedApp, createdApp})
}

func TestListCredentialsListsTheKeysOfTheApplication(t *testing.T) {
	t.Parallel()
	f := newFakeAPI(t, map[string][]answer{
		"GET /iam/v1alpha1/api-keys": {{http.StatusOK, "iam-list-api-keys.json"}},
	})

	got, err := newCredentials(f).ListCredentials(context.Background(), listedApp)
	if err != nil {
		t.Fatal(err)
	}

	want := time.Date(2026, 6, 16, 9, 32, 1, 136614000, time.UTC)
	if len(got) != 1 || got[0].AccessKey != "SCWBN9HR9AA2PZHPSSNV" || !got[0].CreatedAt.Equal(want) {
		t.Errorf("got %+v", got)
	}
	q, _ := url.ParseQuery(f.recorded()[0].query)
	if q.Get("bearer_id") != listedApp || q.Get("bearer_type") != "application" {
		t.Errorf("query %q, want the keys of the application only", f.recorded()[0].query)
	}
	// The API refuses a list without it: « organization_id is required »
	// (first run in production, 05/11/2026).
	if q.Get("organization_id") != orgID {
		t.Errorf("query %q, want the organization", f.recorded()[0].query)
	}
	if f.recorded()[0].auth != secretKey {
		t.Error("request not authenticated with the rotator key")
	}
}

func TestListCredentialsRefusesAKeyOfAnotherApplication(t *testing.T) {
	t.Parallel()
	f := newFakeAPI(t, map[string][]answer{
		"GET /iam/v1alpha1/api-keys": {{http.StatusOK, "iam-list-api-keys.json"}},
	})
	c := scaleway.NewCredentials(f.client(), scope, []string{createdApp})

	_, err := c.ListCredentials(context.Background(), createdApp)

	if !errors.Is(err, scaleway.ErrApplicationNotAllowed) {
		t.Errorf("err = %v, want ErrApplicationNotAllowed", err)
	}
}

func TestCreateCredentialCreatesAKeyOfTheApplication(t *testing.T) {
	t.Parallel()
	f := newFakeAPI(t, map[string][]answer{
		"POST /iam/v1alpha1/api-keys": {{http.StatusOK, "iam-create-api-key.json"}},
	})
	expires := time.Date(2035, 7, 6, 9, 0, 0, 0, time.UTC)

	got, err := newCredentials(f).CreateCredential(context.Background(), createdApp, expires)
	if err != nil {
		t.Fatal(err)
	}

	if got.AccessKey != "SCWXXXXXXXXXXXXXXXXX" || string(got.SecretKey) != "11111111-1111-1111-1111-111111111111" || !got.ExpiresAt.Equal(expires) {
		t.Errorf("got %v, secret %q", got, got.SecretKey)
	}
	var sent struct {
		ApplicationID    string    `json:"application_id"`
		ExpiresAt        time.Time `json:"expires_at"`
		DefaultProjectID string    `json:"default_project_id"`
		Description      string    `json:"description"`
	}
	if err := json.Unmarshal([]byte(f.recorded()[0].body), &sent); err != nil {
		t.Fatal(err)
	}
	if sent.ApplicationID != createdApp || !sent.ExpiresAt.Equal(expires) || sent.DefaultProjectID != projectID || !strings.Contains(sent.Description, "ADR 0007") {
		t.Errorf("sent %+v", sent)
	}
}

func TestCreateCredentialRefusesAKeyWithoutSecret(t *testing.T) {
	t.Parallel()
	f := newFakeAPI(t, map[string][]answer{
		"POST /iam/v1alpha1/api-keys": {{http.StatusOK, "iam-get-api-key.json"}},
	})

	_, err := newCredentials(f).CreateCredential(context.Background(), createdApp, time.Now())

	if err == nil {
		t.Error("a created key without secret key accepted")
	}
}

func TestDeleteCredentialDeletesAKeyOfTheApplication(t *testing.T) {
	t.Parallel()
	f := newFakeAPI(t, map[string][]answer{
		"GET /iam/v1alpha1/api-keys/SCWXXXXXXXXXXXXXXXXX":    {{http.StatusOK, "iam-get-api-key.json"}},
		"DELETE /iam/v1alpha1/api-keys/SCWXXXXXXXXXXXXXXXXX": {{http.StatusNoContent, ""}},
	})

	if err := newCredentials(f).DeleteCredential(context.Background(), createdApp, "SCWXXXXXXXXXXXXXXXXX"); err != nil {
		t.Fatal(err)
	}

	// The SDK also reads /metadata after a GET of a key, to name it.
	calls := f.recorded()
	if calls[0].method != http.MethodGet || calls[len(calls)-1].method != http.MethodDelete {
		t.Errorf("calls %+v, want the key read, then deleted", calls)
	}
}

func TestDeleteCredentialRefusesAKeyOfAnotherApplication(t *testing.T) {
	t.Parallel()
	f := newFakeAPI(t, map[string][]answer{
		"GET /iam/v1alpha1/api-keys/SCWXXXXXXXXXXXXXXXXX":    {{http.StatusOK, "iam-get-api-key.json"}},
		"DELETE /iam/v1alpha1/api-keys/SCWXXXXXXXXXXXXXXXXX": {{http.StatusNoContent, ""}},
	})

	err := newCredentials(f).DeleteCredential(context.Background(), listedApp, "SCWXXXXXXXXXXXXXXXXX")

	if !errors.Is(err, scaleway.ErrApplicationNotAllowed) {
		t.Errorf("err = %v, want ErrApplicationNotAllowed", err)
	}
	for _, c := range f.recorded() {
		if c.method == http.MethodDelete {
			t.Error("key of another application deleted")
		}
	}
}

func TestCredentialsRefuseAnApplicationOutOfTheList(t *testing.T) {
	t.Parallel()
	f := newFakeAPI(t, nil)
	c := newCredentials(f)
	ctx := context.Background()

	_, errList := c.ListCredentials(ctx, otherApp)
	_, errCreate := c.CreateCredential(ctx, otherApp, time.Now())
	errDelete := c.DeleteCredential(ctx, otherApp, "SCWXXXXXXXXXXXXXXXXX")

	for _, err := range []error{errList, errCreate, errDelete} {
		if !errors.Is(err, scaleway.ErrApplicationNotAllowed) {
			t.Errorf("err = %v, want ErrApplicationNotAllowed", err)
		}
	}
	if n := len(f.recorded()); n != 0 {
		t.Errorf("%d calls sent for an application out of the list", n)
	}
}

func TestCredentialsReportAnAPIError(t *testing.T) {
	t.Parallel()
	f := newFakeAPI(t, map[string][]answer{
		"GET /iam/v1alpha1/api-keys": {{http.StatusForbidden, ""}},
	})

	_, err := newCredentials(f).ListCredentials(context.Background(), listedApp)

	if err == nil || !strings.Contains(err.Error(), listedApp) {
		t.Errorf("err = %v, want an error naming the application", err)
	}
}
