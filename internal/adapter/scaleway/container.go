// SPDX-License-Identifier: AGPL-3.0-only

package scaleway

import (
	"context"
	"errors"
	"fmt"
	"time"

	container "github.com/scaleway/scaleway-sdk-go/api/container/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/platform/defaults"
)

// Defaults of ContainerSecretsOptions: a redeployment of the api container
// takes about a minute.
const (
	defaultReadyTimeout  = 10 * time.Minute
	defaultReadyInterval = 5 * time.Second
)

// ContainerSecretsOptions configures a ContainerSecrets.
type ContainerSecretsOptions struct {
	// ContainerID is the Serverless Container whose secrets change.
	ContainerID string

	// AccessVar and SecretVar are its secret environment variables that
	// hold the access key and the secret key.
	AccessVar, SecretVar string

	// Timeout bounds the wait for the container to be ready again, 10
	// minutes when zero; Interval is the pause between two reads of its
	// status, 5 seconds when zero.
	Timeout, Interval time.Duration
}

// ContainerSecrets sets a key in two secret environment variables of a
// Serverless Container. It implements app.CredentialInstaller.
type ContainerSecrets struct {
	api  *container.API
	opts ContainerSecretsOptions
}

// NewContainerSecrets returns a ContainerSecrets of opts, through client,
// or ErrInvalidOptions.
func NewContainerSecrets(client *scw.Client, opts ContainerSecretsOptions) (*ContainerSecrets, error) {
	err := errors.Join(
		defaults.Set("timeout", &opts.Timeout, defaultReadyTimeout),
		defaults.Set("interval", &opts.Interval, defaultReadyInterval),
	)
	switch {
	case err != nil:
		return nil, fmt.Errorf("%w: %w", ErrInvalidOptions, err)
	case opts.ContainerID == "" || opts.AccessVar == "" || opts.SecretVar == "":
		return nil, fmt.Errorf("%w: a container and two variables are required", ErrInvalidOptions)
	}
	return &ContainerSecrets{api: container.NewAPI(client), opts: opts}, nil
}

// InstallCredential sets c in the two variables and waits until the
// container, redeployed with them, is ready. It goes through the
// containers/v1beta1 API, which keeps the secrets it is not given; the
// traffic moves to the new version once it is ready (ADR 0007).
func (s *ContainerSecrets) InstallCredential(ctx context.Context, c app.Credential) error {
	accessKey, secretKey := c.AccessKey, string(c.SecretKey)
	//nolint:staticcheck // SA1019: ADR 0007 keeps v1beta1, which keeps the secrets it is not given; v1 takes a whole table (#308).
	_, err := s.api.UpdateContainer(&container.UpdateContainerRequest{
		Region:      region,
		ContainerID: s.opts.ContainerID,
		SecretEnvironmentVariables: []*container.Secret{
			{Key: s.opts.AccessVar, Value: &accessKey},
			{Key: s.opts.SecretVar, Value: &secretKey},
		},
	}, scw.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("set the key in the secrets of the container %s: %w", s.opts.ContainerID, err)
	}
	ready, err := s.api.WaitForContainer(&container.WaitForContainerRequest{
		Region:        region,
		ContainerID:   s.opts.ContainerID,
		Timeout:       &s.opts.Timeout,
		RetryInterval: &s.opts.Interval,
	}, scw.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("wait for the container %s: %w", s.opts.ContainerID, err)
	}
	if ready.Status != container.ContainerStatusReady {
		return fmt.Errorf("container %s is %s after its update: %s", s.opts.ContainerID, ready.Status, message(ready.ErrorMessage))
	}
	return nil
}

// message returns *m, or "no message".
func message(m *string) string {
	if m == nil || *m == "" {
		return "no message"
	}
	return *m
}
