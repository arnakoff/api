// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/app"
)

var rotateNow = time.Date(2026, 11, 5, 3, 10, 0, 0, time.UTC)

const day = 24 * time.Hour

// rotation is a RotateCredentials over the api, sync, ci and self
// applications, each with one installer, sharing log.
type rotation struct {
	log        []string
	iam        *cloudIAM
	installers map[string]*installer
	rotate     *app.RotateCredentials
}

func newRotation(t *testing.T) *rotation {
	t.Helper()
	r := &rotation{installers: map[string]*installer{}}
	r.iam = newCloudIAM(&r.log)
	target := func(name string) app.CredentialTarget {
		inst := &installer{name: name, log: &r.log}
		r.installers[name] = inst
		return app.CredentialTarget{Name: name, Application: name + "-app", Installers: []app.CredentialInstaller{inst}}
	}
	targets := []app.CredentialTarget{target("api"), target("sync"), target("ci")}
	rotate, err := app.NewRotateCredentials(r.iam, &clock{now: rotateNow}, targets, target("self"))
	if err != nil {
		t.Fatal(err)
	}
	r.rotate = rotate
	return r
}

// age gives every application a key created age ago.
func (r *rotation) age(age time.Duration) {
	for _, name := range []string{"api", "sync", "ci", "self"} {
		r.iam.add(name+"-app", "OLD-"+name, rotateNow.Add(-age))
	}
}

func TestRotateCredentialsRotatesOldKeysInOrder(t *testing.T) {
	t.Parallel()
	r := newRotation(t)
	r.age(31 * day)

	got, err := r.rotate.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"list api-app", "create api-app", "install api NEW1", "delete api-app OLD-api",
		"list sync-app", "create sync-app", "install sync NEW2", "delete sync-app OLD-sync",
		"list ci-app", "create ci-app", "install ci NEW3", "delete ci-app OLD-ci",
		"list self-app", "create self-app", "install self NEW4", "delete self-app OLD-self",
	}
	if !slices.Equal(r.log, want) {
		t.Errorf("steps:\n got %q\nwant %q", r.log, want)
	}
	if len(got) != 4 {
		t.Fatalf("got %d rotations, want 4", len(got))
	}
	first := got[0]
	if first.Name != "api" || !first.Rotated || first.New.AccessKey != "NEW1" || !slices.Equal(first.Deleted, []string{"OLD-api"}) {
		t.Errorf("first rotation = %+v", first)
	}
	if want := rotateNow.Add(app.CredentialLifetime); !first.New.ExpiresAt.Equal(want) {
		t.Errorf("expires at %v, want %v", first.New.ExpiresAt, want)
	}
}

// A key made by hand may live shorter than CredentialLifetime, such as the
// 30-day key of ci-deploy: it is due once it expires sooner than a key the
// job made would at CredentialMaxAge.
func TestRotateCredentialsRotatesAKeyCloseToItsExpiry(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		expires time.Time
		want    bool
	}{
		{"expires in 29 days", rotateNow.Add(29 * day), true},
		{"expires in 59 days", rotateNow.Add(app.CredentialLifetime - app.CredentialMaxAge), true},
		{"expires in 60 days", rotateNow.Add(60 * day), false},
		{"never expires", time.Time{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newRotation(t)
			r.age(day)
			r.iam.keys["ci-app"][0].ExpiresAt = tc.expires

			if _, err := r.rotate.Run(context.Background()); err != nil {
				t.Fatal(err)
			}

			if got := slices.Contains(r.log, "create ci-app"); got != tc.want {
				t.Errorf("ci rotated = %v, want %v (steps %q)", got, tc.want, r.log)
			}
		})
	}
}

func TestRotateCredentialsKeepsRecentKeys(t *testing.T) {
	t.Parallel()
	r := newRotation(t)
	r.age(29 * day)

	got, err := r.rotate.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range r.log {
		if !strings.HasPrefix(s, "list ") {
			t.Errorf("unexpected step %q", s)
		}
	}
	for _, rot := range got {
		if rot.Rotated {
			t.Errorf("%s rotated, want kept", rot.Name)
		}
	}
}

// A run that failed after creating a key leaves it, unused, beside the
// key in service: the next run, the next day, rotates again and deletes
// both (ADR 0007: « un échec est repris le lendemain »).
func TestRotateCredentialsRetriesTheDayAfterAFailedRun(t *testing.T) {
	t.Parallel()
	r := newRotation(t)
	r.age(31 * day)
	r.iam.add("api-app", "UNUSED", rotateNow.Add(-day))

	got, err := r.rotate.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !got[0].Rotated || !slices.Equal(got[0].Deleted, []string{"OLD-api", "UNUSED"}) {
		t.Errorf("api rotation = %+v, want a new key and both older keys deleted", got[0])
	}
}

func TestRotateCredentialsDeletesEveryOlderKey(t *testing.T) {
	t.Parallel()
	r := newRotation(t)
	r.age(60 * day)
	r.iam.add("api-app", "STRAY", rotateNow.Add(-31*day))

	got, err := r.rotate.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(got[0].Deleted, []string{"OLD-api", "STRAY"}) {
		t.Errorf("deleted %q, want both older keys", got[0].Deleted)
	}
}

func TestRotateCredentialsCreatesAMissingKey(t *testing.T) {
	t.Parallel()
	r := newRotation(t)

	got, err := r.rotate.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !got[0].Rotated || len(got[0].Deleted) != 0 {
		t.Errorf("first rotation = %+v, want a new key and nothing deleted", got[0])
	}
}

func TestRotateCredentialsStopsBeforeDeletingOnFailure(t *testing.T) {
	t.Parallel()
	errBoom := errors.New("boom")
	steps := []struct {
		name  string
		setup func(r *rotation)
		last  string
	}{
		{"list", func(r *rotation) { r.iam.fails["list sync-app"] = errBoom }, "list sync-app"},
		{"create", func(r *rotation) { r.iam.fails["create sync-app"] = errBoom }, "create sync-app"},
		{"install", func(r *rotation) { r.installers["sync"].err = errBoom }, "install sync NEW2"},
		{"delete", func(r *rotation) { r.iam.fails["delete sync-app OLD-sync"] = errBoom }, "delete sync-app OLD-sync"},
	}
	for _, tc := range steps {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newRotation(t)
			r.age(31 * day)
			tc.setup(r)

			got, err := r.rotate.Run(context.Background())

			if !errors.Is(err, errBoom) || !strings.Contains(err.Error(), "sync") {
				t.Errorf("err = %v, want boom naming sync", err)
			}
			if last := r.log[len(r.log)-1]; last != tc.last {
				t.Errorf("last step %q, want %q: nothing after a failure", last, tc.last)
			}
			if len(got) == 0 || got[0].Name != "api" || !got[0].Rotated {
				t.Errorf("rotations = %+v, want api done before the failure", got)
			}
		})
	}
}

func TestRotateCredentialsWipesTheNewSecrets(t *testing.T) {
	t.Parallel()
	r := newRotation(t)
	r.age(31 * day)

	got, err := r.rotate.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if s := r.installers["api"].secrets; len(s) != 1 || s[0] != "secret-1" {
		t.Errorf("api installer got %q, want the new secret", s)
	}
	for _, rot := range got {
		if len(rot.New.SecretKey) != 0 {
			t.Errorf("%s: secret key returned", rot.Name)
		}
	}
}

func TestNewRotateCredentialsRefusesABadTarget(t *testing.T) {
	t.Parallel()
	inst := []app.CredentialInstaller{&installer{}}
	self := app.CredentialTarget{Name: "self", Application: "self-app", Installers: inst}
	cases := []struct {
		name    string
		targets []app.CredentialTarget
		self    app.CredentialTarget
	}{
		{"no application", []app.CredentialTarget{{Name: "api", Installers: inst}}, self},
		{"no name", []app.CredentialTarget{{Application: "api-app", Installers: inst}}, self},
		{"no installer", []app.CredentialTarget{{Name: "api", Application: "api-app"}}, self},
		{"twice the same application", []app.CredentialTarget{
			{Name: "api", Application: "api-app", Installers: inst},
			{Name: "sync", Application: "api-app", Installers: inst},
		}, self},
		{"self among the targets", []app.CredentialTarget{{Name: "api", Application: "self-app", Installers: inst}}, self},
		{"self without installer", nil, app.CredentialTarget{Name: "self", Application: "self-app"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := app.NewRotateCredentials(newCloudIAM(new([]string)), &clock{}, tc.targets, tc.self)
			if !errors.Is(err, app.ErrInvalidCredentialTarget) {
				t.Errorf("err = %v, want ErrInvalidCredentialTarget", err)
			}
		})
	}
}

func TestCredentialNeverPrintsItsSecret(t *testing.T) {
	t.Parallel()
	k := app.Credential{AccessKey: "SCWACCESS", SecretKey: []byte("the-secret"), ExpiresAt: rotateNow}
	var logged bytes.Buffer
	slog.New(slog.NewJSONHandler(&logged, nil)).Info("key", "key", k)
	out := fmt.Sprintf("%v %+v %#v %s", k, k, k, k) + logged.String()

	if strings.Contains(out, "the-secret") {
		t.Errorf("secret printed: %s", out)
	}
	if !strings.Contains(out, "SCWACCESS") {
		t.Errorf("access key missing: %s", out)
	}
}
