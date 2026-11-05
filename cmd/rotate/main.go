// SPDX-License-Identifier: AGPL-3.0-only

// Command rotate replaces the Scaleway API keys of the snapshot bucket, of
// the CI and of itself before they expire, with no gap in service
// (ADR 0007). A private Serverless Job runs it once a day; it reads its
// configuration from the environment, rotates the keys that are due and
// exits, non-zero on a failure.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/scaleway/scaleway-sdk-go/scw"

	"github.com/arnakoff/api/internal/adapter/githubapp"
	"github.com/arnakoff/api/internal/adapter/scaleway"
	"github.com/arnakoff/api/internal/adapter/store/objectstore"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/platform/clock"
	"github.com/arnakoff/api/internal/platform/config"
	"github.com/arnakoff/api/internal/platform/httpclient"
	"github.com/arnakoff/api/internal/platform/startup"
)

// Where the key of ci-deploy goes: the secrets the deploy workflow reads
// (ADR 0005, ADR 0007).
const (
	githubOwner       = "arnakoff"
	githubRepo        = "api"
	githubEnvironment = "production"
	githubAccessName  = "SCW_ACCESS_KEY"
	githubSecretName  = "SCW_SECRET_KEY" //nolint:gosec // G101: the name of a secret, not its value.
)

func main() {
	os.Exit(startup.Run("rotate", os.LookupEnv, os.Stderr, run))
}

// run rotates the keys that are due and logs what it did.
func run(ctx context.Context, cfg config.Config) error {
	if err := cfg.RequireRotate(); err != nil {
		return err
	}
	rotate, err := newRotation(cfg.Rotate)
	if err != nil {
		return err
	}
	done, err := rotate.Run(ctx)
	for _, r := range done {
		slog.InfoContext(ctx, "key rotation", "target", r.Name, "application", r.Application,
			"rotated", r.Rotated, "new_key", r.New, "deleted", r.Deleted)
	}
	return err
}

// newRotation wires the rotation of the keys of api, sync, ci-deploy and
// the job itself, each client allowed its own host only.
func newRotation(cfg config.Rotate) (*app.RotateCredentials, error) {
	clients, err := hostClients(scaleway.Host, objectstore.Host, githubapp.Host)
	if err != nil {
		return nil, err
	}
	cloud, err := scaleway.NewClient(scaleway.Options{HTTP: clients[scaleway.Host], AccessKey: cfg.AccessKey, SecretKey: cfg.SecretKey})
	if err != nil {
		return nil, err
	}
	targets, err := newTargets(cfg, cloud, clients)
	if err != nil {
		return nil, err
	}
	self := app.CredentialTarget{Name: "rotate", Application: cfg.Apps.Self, Installers: []app.CredentialInstaller{
		scaleway.NewSecretPair(cloud, cfg.SelfSecrets.Access, cfg.SelfSecrets.Secret),
	}}
	allowed := []string{self.Application}
	for _, t := range targets {
		allowed = append(allowed, t.Application)
	}
	issuer := scaleway.NewCredentials(cloud, cfg.ProjectID, allowed)
	return app.NewRotateCredentials(issuer, clock.System{}, targets, self)
}

// newTargets returns the targets rotated before the job itself: api and
// sync, whose new key must first read the bucket, then ci-deploy.
func newTargets(cfg config.Rotate, cloud *scw.Client, clients map[string]*httpclient.Client) ([]app.CredentialTarget, error) {
	bucket, err := objectstore.NewKeyCheck(objectstore.KeyCheckOptions{Bucket: cfg.Bucket, HTTP: clients[objectstore.Host]})
	if err != nil {
		return nil, err
	}
	apiSecrets, err := scaleway.NewContainerSecrets(cloud, scaleway.ContainerSecretsOptions{
		ContainerID: cfg.APIContainerID,
		AccessVar:   config.EnvSnapshotBucketAccessKey,
		SecretVar:   config.EnvSnapshotBucketSecretKey,
	})
	if err != nil {
		return nil, err
	}
	github, err := githubapp.NewEnvironmentSecrets(githubapp.Options{
		HTTP:           clients[githubapp.Host],
		ClientID:       cfg.GitHubApp.ClientID,
		InstallationID: cfg.GitHubApp.InstallationID,
		PrivateKey:     cfg.GitHubApp.PrivateKey.Bytes(),
		Owner:          githubOwner, Repo: githubRepo, Environment: githubEnvironment,
		AccessName: githubAccessName, SecretName: githubSecretName,
	})
	if err != nil {
		return nil, err
	}
	syncSecrets := scaleway.NewSecretPair(cloud, cfg.SyncSecrets.Access, cfg.SyncSecrets.Secret)
	return []app.CredentialTarget{
		{Name: "api", Application: cfg.Apps.API, Installers: []app.CredentialInstaller{bucket, apiSecrets}},
		{Name: "sync", Application: cfg.Apps.Sync, Installers: []app.CredentialInstaller{bucket, syncSecrets}},
		{Name: "ci-deploy", Application: cfg.Apps.CI, Installers: []app.CredentialInstaller{github}},
	}, nil
}

// hostClients returns an outbound client per host, each allowed its own
// host only, with the default timeouts, size limit and retries.
func hostClients(hosts ...string) (map[string]*httpclient.Client, error) {
	clients := make(map[string]*httpclient.Client, len(hosts))
	for _, h := range hosts {
		c, err := httpclient.New(httpclient.Config{AllowedHosts: []string{h}})
		if err != nil {
			return nil, err
		}
		clients[h] = c
	}
	return clients, nil
}
