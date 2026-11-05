// SPDX-License-Identifier: AGPL-3.0-only

package scaleway

import (
	"context"
	"fmt"

	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"

	"github.com/arnakoff/api/internal/app"
)

// SecretPair keeps a key in two secrets of Secret Manager, one per half,
// as a Serverless Job reads them: one secret per environment variable. It
// implements app.CredentialInstaller.
type SecretPair struct {
	api            *secret.API
	accessSecretID string
	secretSecretID string
}

// NewSecretPair returns a SecretPair writing the access key to the secret
// accessSecretID and the secret key to secretSecretID, through client.
func NewSecretPair(client *scw.Client, accessSecretID, secretSecretID string) *SecretPair {
	return &SecretPair{api: secret.NewAPI(client), accessSecretID: accessSecretID, secretSecretID: secretSecretID}
}

// InstallCredential writes each half of c as a new version of its secret
// and disables the previous version, so the next run of the job reads c.
func (p *SecretPair) InstallCredential(ctx context.Context, c app.Credential) error {
	if err := p.write(ctx, p.accessSecretID, []byte(c.AccessKey), c.AccessKey); err != nil {
		return err
	}
	return p.write(ctx, p.secretSecretID, c.SecretKey, c.AccessKey)
}

// write adds data as the new version of the secret id, described by the
// access key it belongs to.
func (p *SecretPair) write(ctx context.Context, id string, data []byte, accessKey string) error {
	disable := true
	description := "key " + accessKey + ", rotated (ADR 0007)"
	_, err := p.api.CreateSecretVersion(&secret.CreateSecretVersionRequest{
		Region:          region,
		SecretID:        id,
		Data:            data,
		Description:     &description,
		DisablePrevious: &disable,
	}, scw.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("write a version of the secret %s: %w", id, err)
	}
	return nil
}
