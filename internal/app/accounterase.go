// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"fmt"

	"github.com/arnakoff/api/internal/domain"
)

// AccountEraser deletes what is kept about an account.
type AccountEraser interface {
	// EraseAccount deletes, in one transaction, every API key of the
	// account of e, active or revoked, and every login request to it,
	// whatever case the address was typed in, and returns how many of
	// each it deleted. An account with nothing kept deletes nothing.
	EraseAccount(ctx context.Context, e domain.Email) (PurgeCount, error)
}

// AccountErase answers a request for erasure (RGPD, article 17) of the
// holder of an account: its keys and its login requests are deleted at
// once, not kept until domain.RevokedKeyRetention. A running API refuses
// an erased key as soon as the store tells it, as a revoked one.
type AccountErase struct {
	store AccountEraser
}

// NewAccountErase returns an AccountErase deleting from store.
func NewAccountErase(store AccountEraser) *AccountErase {
	return &AccountErase{store: store}
}

// Erase deletes the keys and the login requests of the account of e, and
// returns how many of each it deleted. It may be run again: an erased
// account deletes nothing more.
func (a *AccountErase) Erase(ctx context.Context, e domain.Email) (PurgeCount, error) {
	n, err := a.store.EraseAccount(ctx, e)
	if err != nil {
		return PurgeCount{}, fmt.Errorf("erase account %s: %w", e.Redacted(), err)
	}
	return n, nil
}
