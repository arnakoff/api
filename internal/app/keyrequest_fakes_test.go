// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// loginStore is an in-memory app.LoginRequestStore. A non-nil err is the
// answer to every call.
type loginStore struct {
	mu   sync.Mutex
	reqs []loginRow
	err  error
}

// loginRow is a stored login request and when its link was used.
type loginRow struct {
	domain.LoginRequest
	usedAt time.Time
}

func (s *loginStore) CreateLoginRequest(_ context.Context, r domain.LoginRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	s.reqs = append(s.reqs, loginRow{LoginRequest: r})
	return nil
}

func (s *loginStore) UseLoginRequest(_ context.Context, d domain.TokenDigest, at time.Time) (domain.Email, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return domain.Email{}, s.err
	}
	i := slices.IndexFunc(s.reqs, func(r loginRow) bool { return r.Digest == d })
	if i < 0 || !s.reqs[i].usedAt.IsZero() || !at.Before(s.reqs[i].ExpiresAt) {
		return domain.Email{}, app.ErrLoginLinkInvalid
	}
	s.reqs[i].usedAt = at
	return s.reqs[i].Email, nil
}

func (s *loginStore) PurgeLoginRequests(_ context.Context, cutoff time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return 0, s.err
	}
	before := len(s.reqs)
	s.reqs = slices.DeleteFunc(s.reqs, func(r loginRow) bool {
		end := r.ExpiresAt
		if !r.usedAt.IsZero() {
			end = r.usedAt
		}
		return end.Before(cutoff)
	})
	return int64(before - len(s.reqs)), nil
}

func (s *loginStore) requests() []loginRow {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.reqs)
}

// ReplaceAccountKey revokes, at r.CreatedAt, every active key of the
// account of r, then stores r.
func (s *keyStore) ReplaceAccountKey(ctx context.Context, r domain.KeyRecord) (domain.KeyRecord, error) {
	s.mu.Lock()
	if s.err != nil {
		s.mu.Unlock()
		return domain.KeyRecord{}, s.err
	}
	for i, old := range s.recs {
		if old.Active() && !old.Email.IsZero() && old.Email.Account() == r.Email.Account() {
			s.recs[i].RevokedAt = r.CreatedAt
		}
	}
	s.mu.Unlock()
	return s.CreateKey(ctx, r)
}

// mailer is a fake app.Mailer: it fails with err, or records each mail.
type mailer struct {
	mu   sync.Mutex
	sent []mail
	err  error
}

// mail is a message the fake mailer was asked to send.
type mail struct {
	to            domain.Email
	subject, body string
}

func (m *mailer) Send(_ context.Context, to domain.Email, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, mail{to: to, subject: subject, body: body})
	return nil
}

func (m *mailer) mails() []mail {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.sent)
}

// purgeStore is the app.Purger of the key store and the login store.
type purgeStore struct {
	*keyStore
	*loginStore
}
