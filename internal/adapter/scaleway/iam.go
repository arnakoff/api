// SPDX-License-Identifier: AGPL-3.0-only

package scaleway

import (
	"context"
	"errors"
	"fmt"
	"time"

	iam "github.com/scaleway/scaleway-sdk-go/api/iam/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"

	"github.com/arnakoff/api/internal/app"
)

// keyDescription is the description of every key the rotator creates, so a
// reader of the console or of the Audit Trail knows where it comes from.
const keyDescription = "created by the key rotation job (ADR 0007)"

// ErrApplicationNotAllowed means a key of an application out of the
// closed list of the rotator was asked for, or met. Nothing was changed.
var ErrApplicationNotAllowed = errors.New("application not in the list of the rotator")

// Credentials creates, lists and deletes the API keys of a closed list of
// IAM applications. It implements app.CredentialIssuer.
type Credentials struct {
	api     *iam.API
	scope   Scope
	allowed map[string]bool
}

// Scope is where the keys are: the organization of the applications, which
// the API requires to list keys, and the default project of a new key.
type Scope struct {
	OrganizationID string
	ProjectID      string
}

// NewCredentials returns Credentials through client, for the applications
// allowed only, in scope.
func NewCredentials(client *scw.Client, scope Scope, allowed []string) *Credentials {
	set := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		set[a] = true
	}
	return &Credentials{api: iam.NewAPI(client), scope: scope, allowed: set}
}

// ListCredentials returns the keys of application.
func (c *Credentials) ListCredentials(ctx context.Context, application string) ([]app.Credential, error) {
	if err := c.check(application); err != nil {
		return nil, err
	}
	resp, err := c.api.ListAPIKeys(&iam.ListAPIKeysRequest{
		OrganizationID: &c.scope.OrganizationID,
		BearerID:       &application,
		BearerType:     iam.BearerTypeApplication,
	}, scw.WithContext(ctx), scw.WithAllPages())
	if err != nil {
		return nil, fmt.Errorf("list the keys of %s: %w", application, err)
	}
	keys := make([]app.Credential, 0, len(resp.APIKeys))
	for _, k := range resp.APIKeys {
		if err := belongs(k, application); err != nil {
			return nil, err
		}
		keys = append(keys, credential(k))
	}
	return keys, nil
}

// CreateCredential creates a key of application valid until expiresAt.
func (c *Credentials) CreateCredential(ctx context.Context, application string, expiresAt time.Time) (app.Credential, error) {
	if err := c.check(application); err != nil {
		return app.Credential{}, err
	}
	k, err := c.api.CreateAPIKey(&iam.CreateAPIKeyRequest{
		ApplicationID:    &application,
		ExpiresAt:        &expiresAt,
		DefaultProjectID: &c.scope.ProjectID,
		Description:      keyDescription,
	}, scw.WithContext(ctx))
	if err != nil {
		return app.Credential{}, fmt.Errorf("create a key of %s: %w", application, err)
	}
	if err := belongs(k, application); err != nil {
		return app.Credential{}, err
	}
	if k.SecretKey == nil || *k.SecretKey == "" {
		return app.Credential{}, fmt.Errorf("create a key of %s: key %s came without its secret key", application, k.AccessKey)
	}
	created := credential(k)
	created.SecretKey = []byte(*k.SecretKey)
	return created, nil
}

// DeleteCredential deletes the key accessKey once it has checked that it
// is a key of application.
func (c *Credentials) DeleteCredential(ctx context.Context, application, accessKey string) error {
	if err := c.check(application); err != nil {
		return err
	}
	k, err := c.api.GetAPIKey(&iam.GetAPIKeyRequest{AccessKey: accessKey}, scw.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("read the key %s: %w", accessKey, err)
	}
	if err := belongs(k, application); err != nil {
		return err
	}
	if err := c.api.DeleteAPIKey(&iam.DeleteAPIKeyRequest{AccessKey: accessKey}, scw.WithContext(ctx)); err != nil {
		return fmt.Errorf("delete the key %s: %w", accessKey, err)
	}
	return nil
}

// check returns ErrApplicationNotAllowed unless application is allowed.
func (c *Credentials) check(application string) error {
	if !c.allowed[application] {
		return fmt.Errorf("%w: %s", ErrApplicationNotAllowed, application)
	}
	return nil
}

// belongs returns ErrApplicationNotAllowed unless k is a key of
// application.
func belongs(k *iam.APIKey, application string) error {
	if k.ApplicationID == nil || *k.ApplicationID != application {
		return fmt.Errorf("%w: key %s is not a key of %s", ErrApplicationNotAllowed, k.AccessKey, application)
	}
	return nil
}

// credential returns k without its secret key.
func credential(k *iam.APIKey) app.Credential {
	c := app.Credential{AccessKey: k.AccessKey}
	if k.CreatedAt != nil {
		c.CreatedAt = *k.CreatedAt
	}
	if k.ExpiresAt != nil {
		c.ExpiresAt = *k.ExpiresAt
	}
	return c
}
