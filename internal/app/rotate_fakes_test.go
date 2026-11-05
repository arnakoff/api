// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/arnakoff/api/internal/app"
)

// cloudIAM is an in-memory app.CredentialIssuer that records every call in
// log, shared with the installers so a test reads the order of the steps.
type cloudIAM struct {
	log   *[]string
	keys  map[string][]app.Credential
	next  int
	fails map[string]error // by step, such as "create api" or "delete api AK1"
}

func newCloudIAM(log *[]string) *cloudIAM {
	return &cloudIAM{log: log, keys: map[string][]app.Credential{}, fails: map[string]error{}}
}

// add gives application a key created at created.
func (c *cloudIAM) add(application, accessKey string, created time.Time) {
	c.keys[application] = append(c.keys[application], app.Credential{AccessKey: accessKey, CreatedAt: created})
}

func (c *cloudIAM) step(s string) error {
	*c.log = append(*c.log, s)
	return c.fails[s]
}

func (c *cloudIAM) ListCredentials(_ context.Context, application string) ([]app.Credential, error) {
	if err := c.step("list " + application); err != nil {
		return nil, err
	}
	return slices.Clone(c.keys[application]), nil
}

func (c *cloudIAM) CreateCredential(_ context.Context, application string, expiresAt time.Time) (app.Credential, error) {
	if err := c.step("create " + application); err != nil {
		return app.Credential{}, err
	}
	c.next++
	k := app.Credential{
		AccessKey: fmt.Sprintf("NEW%d", c.next),
		SecretKey: []byte(fmt.Sprintf("secret-%d", c.next)),
		ExpiresAt: expiresAt,
	}
	c.keys[application] = append(c.keys[application], app.Credential{AccessKey: k.AccessKey, ExpiresAt: expiresAt})
	return k, nil
}

func (c *cloudIAM) DeleteCredential(_ context.Context, application, accessKey string) error {
	if err := c.step("delete " + application + " " + accessKey); err != nil {
		return err
	}
	c.keys[application] = slices.DeleteFunc(c.keys[application], func(k app.Credential) bool { return k.AccessKey == accessKey })
	return nil
}

// installer is an app.CredentialInstaller that logs the key it puts in
// service and keeps a copy of its secret.
type installer struct {
	name    string
	log     *[]string
	err     error
	secrets []string
}

func (i *installer) InstallCredential(_ context.Context, k app.Credential) error {
	*i.log = append(*i.log, "install "+i.name+" "+k.AccessKey)
	i.secrets = append(i.secrets, string(k.SecretKey))
	return i.err
}
