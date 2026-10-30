// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// Clock tells the current time; tests inject a fixed one.
type Clock interface {
	Now() time.Time
}

// SourceFetcher downloads and parses one upstream source.
type SourceFetcher interface {
	// Spec declares the source and what a sane file of it holds.
	Spec() domain.SnapshotSpec
	// Fetch downloads the source and parses it into a snapshot. Sync sets
	// its FetchedAt.
	Fetch(ctx context.Context) (domain.Snapshot, error)
}

// SnapshotStore keeps the last valid snapshot of each source across
// restarts and between the sync and API processes.
type SnapshotStore interface {
	// Save durably replaces the stored snapshot of s.Source.
	Save(ctx context.Context, s domain.Snapshot) error
	// Load returns the stored snapshot of source, or ErrNoSnapshot.
	Load(ctx context.Context, source domain.Source) (domain.Snapshot, error)
}

// ErrNoSnapshot is returned by SnapshotStore.Load when it holds no snapshot
// of the source.
var ErrNoSnapshot = errors.New("no stored snapshot")

// Sleeper pauses a scheduling loop; tests inject one that does not wait.
type Sleeper interface {
	// Sleep returns after d, or with the error of ctx once it is done.
	Sleep(ctx context.Context, d time.Duration) error
}

// Alerter tells the operator that a source could not be synced or reloaded.
// It reports its own failures: the caller has nothing to do about them.
type Alerter interface {
	Alert(ctx context.Context, a Alert)
}

// Alert is a sync or reload of one source that failed or was rejected.
type Alert struct {
	Source domain.Source
	At     time.Time
	Err    error
}

// Registry errors, wrapped by the registry adapters and checked with
// errors.Is.
var (
	// ErrNotFound means the registry answered that it holds no such entry:
	// an answer, not a failure.
	ErrNotFound = errors.New("not found in the registry")
	// ErrNotCovered means the registry does not cover this input, such as a
	// domain extension without an RDAP server.
	ErrNotCovered = errors.New("not covered by the registry")
)

// DomainRegistry reads what a domain registry publishes about a domain.
type DomainRegistry interface {
	// Source names the registry in reports.
	Source() domain.Source
	// Lookup returns the registration of d's registrable domain.
	Lookup(ctx context.Context, d domain.Domain) (domain.Registration, error)
}

// CompanyRegistry reads what the company register publishes about a
// company.
type CompanyRegistry interface {
	// Source names the registry in reports.
	Source() domain.Source
	// Lookup returns the company n identifies.
	Lookup(ctx context.Context, n domain.SIREN) (domain.Company, error)
}

// NoticeRegistry reads the legal notices published about a company.
type NoticeRegistry interface {
	// Source names the registry in reports.
	Source() domain.Source
	// Notices returns the notices about the company n identifies.
	Notices(ctx context.Context, n domain.SIREN) ([]domain.Notice, error)
}

// BlacklistReader reads a blacklist held in memory.
type BlacklistReader interface {
	// Blacklist returns the blacklist in service and when it was fetched,
	// or ErrNoSnapshot when none is. The blacklist is shared: treat it as
	// read-only.
	Blacklist() (*domain.Blacklist, time.Time, error)
}

// WhitelistReader reads a white list held in memory.
type WhitelistReader interface {
	// Whitelist returns the white list in service and when it was fetched,
	// or ErrNoSnapshot when none is. The white list is shared: treat it as
	// read-only.
	Whitelist() (*domain.Whitelist, time.Time, error)
}
