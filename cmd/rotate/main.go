// SPDX-License-Identifier: AGPL-3.0-only

// Command rotate replaces the Scaleway API keys of the snapshot bucket, of
// the mail, of the CI and of itself before they expire, with no gap in service
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
	"github.com/arnakoff/api/internal/adapter/smtpmail"
	"github.com/arnakoff/api/internal/adapter/store/objectstore"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
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

// newRotation wires the rotation of the keys of api, sync, mail,
// ci-deploy and the job itself, each client allowed its own host only.
func newRotation(cfg config.Rotate) (*app.RotateCredentials, error) {
	clients, err := hostClients(
		// The Scaleway SDK does not retry: Do does, for its idempotent
		// calls (#312). The S3 SDK of the bucket check retries by itself,
		// and the GitHub App calls go through Send, which retries.
		httpclient.Config{AllowedHosts: []string{scaleway.Host}, RetryDo: true},
		httpclient.Config{AllowedHosts: []string{objectstore.Host}},
		httpclient.Config{AllowedHosts: []string{githubapp.Host}},
	)
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
	issuer := scaleway.NewCredentials(cloud, scaleway.Scope{OrganizationID: cfg.OrganizationID, ProjectID: cfg.ProjectID}, allowed)
	return app.NewRotateCredentials(issuer, clock.System{}, targets, self)
}

// newTargets returns the targets rotated before the job itself: sync and
// api, whose new key must first read the bucket, then mail, then
// ci-deploy. sync comes
// first: on a fresh install, the api container gets ready only once sync
// has run with its key (docs/deploy.md, step 7).
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
	mail, err := mailTarget(cfg, cloud)
	if err != nil {
		return nil, err
	}
	syncSecrets := scaleway.NewSecretPair(cloud, cfg.SyncSecrets.Access, cfg.SyncSecrets.Secret)
	return []app.CredentialTarget{
		{Name: "sync", Application: cfg.Apps.Sync, Installers: []app.CredentialInstaller{bucket, syncSecrets}},
		{Name: "api", Application: cfg.Apps.API, Installers: []app.CredentialInstaller{bucket, apiSecrets}},
		mail,
		{Name: "ci-deploy", Application: cfg.Apps.CI, Installers: []app.CredentialInstaller{github}},
	}, nil
}

// mailTarget returns the target of the mail application (#295): a new key
// must first authenticate with the SMTP server, then becomes the SMTP
// password of the api container; its username, the project, never
// changes.
func mailTarget(cfg config.Rotate, cloud *scw.Client) (app.CredentialTarget, error) {
	// EHLO names the domain of the sender address, as the API does.
	from, err := domain.ParseEmail(config.DefaultMailFrom)
	if err != nil {
		return app.CredentialTarget{}, err
	}
	check, err := smtpmail.NewAuthCheck(smtpmail.AuthCheckOptions{
		Addr: cfg.SMTP.Addr, Username: cfg.SMTP.Username, HelloName: from.Domain().String(),
	})
	if err != nil {
		return app.CredentialTarget{}, err
	}
	password, err := scaleway.NewContainerSecrets(cloud, scaleway.ContainerSecretsOptions{
		ContainerID: cfg.APIContainerID,
		SecretVar:   config.EnvSMTPPassword,
	})
	if err != nil {
		return app.CredentialTarget{}, err
	}
	return app.CredentialTarget{Name: "mail", Application: cfg.Apps.Mail, Installers: []app.CredentialInstaller{check, password}}, nil
}

// hostClients returns an outbound client per config, keyed by its one
// allowed host, with the default timeouts, size limit and retries.
func hostClients(cfgs ...httpclient.Config) (map[string]*httpclient.Client, error) {
	clients := make(map[string]*httpclient.Client, len(cfgs))
	for _, cfg := range cfgs {
		c, err := httpclient.New(cfg)
		if err != nil {
			return nil, err
		}
		clients[cfg.AllowedHosts[0]] = c
	}
	return clients, nil
}
