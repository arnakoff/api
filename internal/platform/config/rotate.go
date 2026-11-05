// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"errors"
	"fmt"
	"strconv"
)

// Environment variables of the key rotation job (ADR 0007), read by Load.
//
//nolint:gosec // G101: the names of variables, not their values.
const (
	EnvRotateAccessKey         = "ARNAKOFF_ROTATE_ACCESS_KEY"
	EnvRotateSecretKey         = "ARNAKOFF_ROTATE_SECRET_KEY"
	EnvRotateOrganizationID    = "ARNAKOFF_ROTATE_ORGANIZATION_ID"
	EnvRotateProjectID         = "ARNAKOFF_ROTATE_PROJECT_ID"
	EnvRotateBucket            = "ARNAKOFF_ROTATE_BUCKET"
	EnvRotateAppAPI            = "ARNAKOFF_ROTATE_APP_API"
	EnvRotateAppSync           = "ARNAKOFF_ROTATE_APP_SYNC"
	EnvRotateAppCI             = "ARNAKOFF_ROTATE_APP_CI"
	EnvRotateAppSelf           = "ARNAKOFF_ROTATE_APP_SELF"
	EnvRotateAppMail           = "ARNAKOFF_ROTATE_APP_MAIL"
	EnvRotateSMTPAddr          = "ARNAKOFF_ROTATE_SMTP_ADDR"
	EnvRotateSMTPUsername      = "ARNAKOFF_ROTATE_SMTP_USERNAME"
	EnvRotateAPIContainerID    = "ARNAKOFF_ROTATE_API_CONTAINER_ID"
	EnvRotateSyncAccessSecret  = "ARNAKOFF_ROTATE_SYNC_ACCESS_SECRET_ID"
	EnvRotateSyncSecretSecret  = "ARNAKOFF_ROTATE_SYNC_SECRET_SECRET_ID"
	EnvRotateSelfAccessSecret  = "ARNAKOFF_ROTATE_SELF_ACCESS_SECRET_ID"
	EnvRotateSelfSecretSecret  = "ARNAKOFF_ROTATE_SELF_SECRET_SECRET_ID"
	EnvGitHubAppClientID       = "ARNAKOFF_GITHUB_APP_CLIENT_ID"
	EnvGitHubAppInstallationID = "ARNAKOFF_GITHUB_APP_INSTALLATION_ID"
	EnvGitHubAppPrivateKey     = "ARNAKOFF_GITHUB_APP_PRIVATE_KEY"
)

// Rotate is the configuration of the key rotation job: its own Scaleway
// key, the closed list of applications whose keys it rotates and where it
// puts them. Every field is empty when unset; RequireRotate tells whether
// it is complete. Identifiers are UUIDs, checked by Load.
type Rotate struct {
	// AccessKey and SecretKey are the key of the job itself.
	AccessKey string
	SecretKey Secret

	// OrganizationID is the organization of the applications, which the
	// API requires to list their keys; ProjectID is the default project of
	// the keys it creates.
	OrganizationID string
	ProjectID      string

	// Bucket is the snapshot bucket a new key of api or sync must read.
	Bucket string

	// Apps are the IAM applications of the closed list.
	Apps RotateApps

	// APIContainerID is the api container, whose secrets hold its key.
	APIContainerID string

	// SyncSecrets and SelfSecrets are the Secret Manager secrets the sync
	// job and the rotation job read their key from, one per half.
	SyncSecrets, SelfSecrets SecretIDs

	// GitHubApp writes the key of ci-deploy to the GitHub environment.
	GitHubApp GitHubApp

	// SMTP is the server a new key of mail must authenticate with.
	SMTP RotateSMTP
}

// RotateSMTP is the SMTP server of the mail application (#295): its
// "host:port", which speaks TLS at once, and the username its keys
// authenticate as.
type RotateSMTP struct {
	Addr, Username string
}

// RotateApps are the IAM applications whose keys the job rotates.
type RotateApps struct {
	API, Sync, CI, Self, Mail string
}

// SecretIDs are the two secrets of a key: its access key and its secret
// key.
type SecretIDs struct {
	Access, Secret string
}

// GitHubApp is the GitHub App the job writes environment secrets as.
type GitHubApp struct {
	ClientID       string
	InstallationID int64
	// PrivateKey is its private key, in PEM, checked where it is used.
	PrivateKey Secret
}

// RequireRotate returns an error, wrapping ErrInvalid and naming each
// missing variable, unless the rotation job is configured.
func (c Config) RequireRotate() error {
	r := &c.Rotate
	required := []requiredVar{
		{EnvRotateAccessKey, r.AccessKey != ""},
		{EnvRotateSecretKey, len(r.SecretKey) != 0},
		{EnvRotateBucket, r.Bucket != ""},
		{EnvRotateSMTPAddr, r.SMTP.Addr != ""},
		{EnvRotateSMTPUsername, r.SMTP.Username != ""},
		{EnvGitHubAppClientID, r.GitHubApp.ClientID != ""},
		{EnvGitHubAppInstallationID, r.GitHubApp.InstallationID != 0},
		{EnvGitHubAppPrivateKey, len(r.GitHubApp.PrivateKey) != 0},
	}
	for _, id := range r.ids() {
		required = append(required, requiredVar{id.name, *id.field != ""})
	}
	var errs []error
	for _, v := range required {
		if !v.set {
			errs = append(errs, fmt.Errorf("%w: %s: required by the rotation job", ErrInvalid, v.name))
		}
	}
	return errors.Join(errs...)
}

// requiredVar is a variable of the rotation job, and whether it is set.
type requiredVar struct {
	name string
	set  bool
}

// idVar is a variable of the rotation job that holds a UUID, and its
// field.
type idVar struct {
	name  string
	field *string
}

// ids are the variables of r that hold a UUID.
func (r *Rotate) ids() []idVar {
	return []idVar{
		{EnvRotateOrganizationID, &r.OrganizationID},
		{EnvRotateProjectID, &r.ProjectID},
		{EnvRotateAppAPI, &r.Apps.API},
		{EnvRotateAppSync, &r.Apps.Sync},
		{EnvRotateAppCI, &r.Apps.CI},
		{EnvRotateAppSelf, &r.Apps.Self},
		{EnvRotateAppMail, &r.Apps.Mail},
		{EnvRotateAPIContainerID, &r.APIContainerID},
		{EnvRotateSyncAccessSecret, &r.SyncSecrets.Access},
		{EnvRotateSyncSecretSecret, &r.SyncSecrets.Secret},
		{EnvRotateSelfAccessSecret, &r.SelfSecrets.Access},
		{EnvRotateSelfSecretSecret, &r.SelfSecrets.Secret},
	}
}

// parseRotate sets *r from the variables of the rotation job, read with
// lookup.
func parseRotate(lookup func(string) (string, bool), r *Rotate) error {
	errs := []error{
		parse(lookup, EnvRotateAccessKey, "", &r.AccessKey, parseAny),
		parse(lookup, EnvRotateSecretKey, "", &r.SecretKey, parseOptionalSecret),
		parse(lookup, EnvRotateBucket, "", &r.Bucket, parseBucketName),
		parse(lookup, EnvRotateSMTPAddr, "", &r.SMTP.Addr, parseHostPort),
		parse(lookup, EnvRotateSMTPUsername, "", &r.SMTP.Username, parseAny),
		parse(lookup, EnvGitHubAppClientID, "", &r.GitHubApp.ClientID, parseAny),
		parse(lookup, EnvGitHubAppInstallationID, "", &r.GitHubApp.InstallationID, parseOptionalID),
		parse(lookup, EnvGitHubAppPrivateKey, "", &r.GitHubApp.PrivateKey, parseOptionalSecret),
	}
	for _, id := range r.ids() {
		errs = append(errs, parse(lookup, id.name, "", id.field, parseOptionalUUID))
	}
	return errors.Join(errs...)
}

// parseOptionalUUID accepts an empty string, as none, or a UUID in its
// canonical lower-case form, as Scaleway writes its identifiers.
func parseOptionalUUID(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	if len(s) != 36 {
		return "", errors.New("not a UUID")
	}
	for i, r := range s {
		dash := i == 8 || i == 13 || i == 18 || i == 23
		hex := r >= '0' && r <= '9' || r >= 'a' && r <= 'f'
		if dash != (r == '-') || !dash && !hex {
			return "", errors.New("not a lower-case UUID")
		}
	}
	return s, nil
}

// parseOptionalID accepts an empty string, as none, or a positive integer.
func parseOptionalID(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return 0, errors.New("not a positive integer")
	}
	return n, nil
}
