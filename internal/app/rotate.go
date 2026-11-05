// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"
)

// Timing of the rotation, as ADR 0007 decides: the keys of an application
// are replaced once one of them is older than CredentialMaxAge, or expires
// within CredentialLifetime - CredentialMaxAge, by a key valid
// CredentialLifetime. A failed run leaves the old key in service and
// due, so the next run, the next day, tries again long before it expires.
const (
	CredentialMaxAge   = 30 * 24 * time.Hour
	CredentialLifetime = 89 * 24 * time.Hour
)

// ErrInvalidCredentialTarget means NewRotateCredentials was given a target
// it cannot rotate.
var ErrInvalidCredentialTarget = errors.New("invalid credential target")

// Credential is an API key of a cloud application, such as the key the
// sync job reads the snapshot bucket with (ADR 0007). Its secret key is set
// only on a key just created. It never prints its secret key: every fmt
// verb and log/slog write its access key and expiry only.
type Credential struct {
	// AccessKey identifies the key; it is not a secret.
	AccessKey string

	// SecretKey is the secret half, set by CreateCredential only.
	SecretKey []byte

	CreatedAt time.Time
	ExpiresAt time.Time
}

// Format implements fmt.Formatter: every verb writes the access key and
// the expiry, never the secret key.
func (c Credential) Format(f fmt.State, _ rune) {
	_, _ = io.WriteString(f, c.AccessKey+" (expires "+c.ExpiresAt.Format(time.RFC3339)+")")
}

// LogValue implements slog.LogValuer: the access key and the expiry only.
func (c Credential) LogValue() slog.Value {
	return slog.GroupValue(slog.String("access_key", c.AccessKey), slog.Time("expires_at", c.ExpiresAt))
}

// CredentialIssuer creates, lists and deletes the API keys of cloud
// applications.
type CredentialIssuer interface {
	// ListCredentials returns the keys of application, without their
	// secret key.
	ListCredentials(ctx context.Context, application string) ([]Credential, error)
	// CreateCredential creates a key of application valid until
	// expiresAt and returns it with its secret key.
	CreateCredential(ctx context.Context, application string, expiresAt time.Time) (Credential, error)
	// DeleteCredential deletes the key accessKey of application.
	DeleteCredential(ctx context.Context, application, accessKey string) error
}

// CredentialInstaller puts a new key in service for one of its consumers,
// or checks that it works, and returns once it is done.
type CredentialInstaller interface {
	InstallCredential(ctx context.Context, c Credential) error
}

// CredentialTarget is an application whose key is rotated, and what puts a
// new key in service, in order.
type CredentialTarget struct {
	// Name names the target in reports, such as "sync".
	Name string
	// Application identifies the cloud application.
	Application string
	// Installers put a new key in service, in order.
	Installers []CredentialInstaller
}

// CredentialRotation is what Run did for one target.
type CredentialRotation struct {
	Name        string
	Application string

	// Rotated reports whether a new key was created.
	Rotated bool

	// New is the new key, its secret key wiped.
	New Credential

	// Deleted are the access keys of the old keys deleted.
	Deleted []string
}

// RotateCredentials replaces the API keys of a closed list of cloud
// applications before they expire, with no gap in service (ADR 0007).
type RotateCredentials struct {
	issuer  CredentialIssuer
	clock   Clock
	targets []CredentialTarget
}

// NewRotateCredentials returns a RotateCredentials of targets, then of
// self, the application whose key Run itself uses, rotated last since its
// old key stops working once deleted. It returns
// ErrInvalidCredentialTarget for a target without a name, an application
// or an installer, or an application listed twice.
func NewRotateCredentials(issuer CredentialIssuer, clock Clock, targets []CredentialTarget, self CredentialTarget) (*RotateCredentials, error) {
	all := append(append([]CredentialTarget(nil), targets...), self)
	seen := make(map[string]bool, len(all))
	for _, t := range all {
		switch {
		case t.Name == "" || t.Application == "":
			return nil, fmt.Errorf("%w: a name and an application are required", ErrInvalidCredentialTarget)
		case len(t.Installers) == 0:
			return nil, fmt.Errorf("%w: %s has no installer", ErrInvalidCredentialTarget, t.Name)
		case seen[t.Application]:
			return nil, fmt.Errorf("%w: %s: application listed twice", ErrInvalidCredentialTarget, t.Name)
		}
		seen[t.Application] = true
	}
	return &RotateCredentials{issuer: issuer, clock: clock, targets: all}, nil
}

// Run rotates the key of each target in turn. For a target with a key
// older than CredentialMaxAge, or with none, it creates a key,
// runs the installers in order, then deletes every older key. It stops at
// the first failure, before deleting anything of that target, so the old
// keys stay in service; it returns what it did up to there.
func (r *RotateCredentials) Run(ctx context.Context) ([]CredentialRotation, error) {
	done := make([]CredentialRotation, 0, len(r.targets))
	for _, t := range r.targets {
		rot, err := r.rotate(ctx, t)
		done = append(done, rot)
		if err != nil {
			return done, fmt.Errorf("rotate the key of %s: %w", t.Name, err)
		}
	}
	return done, nil
}

// rotate rotates the key of t if it is due.
func (r *RotateCredentials) rotate(ctx context.Context, t CredentialTarget) (CredentialRotation, error) {
	rot := CredentialRotation{Name: t.Name, Application: t.Application}
	old, err := r.issuer.ListCredentials(ctx, t.Application)
	if err != nil {
		return rot, fmt.Errorf("list keys: %w", err)
	}
	now := r.clock.Now()
	if !due(old, now) {
		return rot, nil
	}
	key, err := r.issuer.CreateCredential(ctx, t.Application, now.Add(CredentialLifetime))
	if err != nil {
		return rot, fmt.Errorf("create key: %w", err)
	}
	defer clear(key.SecretKey)
	rot.Rotated = true
	rot.New = Credential{AccessKey: key.AccessKey, CreatedAt: key.CreatedAt, ExpiresAt: key.ExpiresAt}
	for _, inst := range t.Installers {
		if err := inst.InstallCredential(ctx, key); err != nil {
			return rot, fmt.Errorf("install key %s: %w", key.AccessKey, err)
		}
	}
	for _, k := range old {
		if err := r.issuer.DeleteCredential(ctx, t.Application, k.AccessKey); err != nil {
			return rot, fmt.Errorf("delete key %s: %w", k.AccessKey, err)
		}
		rot.Deleted = append(rot.Deleted, k.AccessKey)
	}
	return rot, nil
}

// due reports whether keys is empty or holds a key older than
// CredentialMaxAge, or one that expires sooner than a key the job made
// would at that age, such as a short key made by hand. The key in service
// is the oldest: a key a failed run created is newer and was never put in
// service everywhere.
func due(keys []Credential, now time.Time) bool {
	if len(keys) == 0 {
		return true
	}
	for _, k := range keys {
		old := now.Sub(k.CreatedAt) >= CredentialMaxAge
		expiring := !k.ExpiresAt.IsZero() && k.ExpiresAt.Sub(now) <= CredentialLifetime-CredentialMaxAge
		if old || expiring {
			return true
		}
	}
	return false
}
