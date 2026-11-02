// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/ratelimit"
)

// errStoreBusy stands for a key store whose connections are all taken.
var errStoreBusy = errors.New("key store busy")

// keyStore is a fake authenticator backed by a store that answers at most
// capacity lookups, as a connection pool held by slow queries would: it
// accepts testKey, refuses any other key, and fails once full.
type keyStore struct {
	capacity int

	mu      sync.Mutex
	lookups int
}

func (s *keyStore) Authenticate(ctx context.Context, raw string) (domain.KeyRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lookups >= s.capacity {
		return domain.KeyRecord{}, errStoreBusy
	}
	s.lookups++
	if raw != testKey {
		return domain.KeyRecord{}, app.ErrKeyNotFound
	}
	return keyring{}.Authenticate(ctx, raw)
}

func (s *keyStore) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lookups
}

// unknownKey is a key the stores of these tests never issued.
const unknownKey = "ark_test_ffffffffffffffffffffffffffffff000000"

func TestRefusedKeysAreLimitedPerAddress(t *testing.T) {
	t.Parallel()
	for _, route := range []string{"GET /v1/check?q=armen.ltd", "POST /v1/check/batch"} {
		t.Run(route, func(t *testing.T) {
			t.Parallel()
			method, target, _ := strings.Cut(route, " ")
			refusalsAreLimited(t, method, target)
		})
	}
}

// refusalsAreLimited checks that the route at method and target refuses
// two unknown keys, then answers the same address 429 without a lookup,
// and still serves another address.
func refusalsAreLimited(t *testing.T, method, target string) {
	t.Helper()
	store := &keyStore{capacity: 100}
	opts, _ := options(t, true)
	opts.Auth, opts.AuthLimit = store, frozenLimiter(2)
	h := httpapi.NewHandler(opts)
	for i := range 2 {
		res := do(h, withKey(fromAddr(t, method, target, "203.0.113.7:1"), unknownKey))
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("refused key #%d: status %d, want 401", i+1, res.Code)
		}
	}
	res := do(h, authorize(fromAddr(t, method, target, "203.0.113.7:2")))
	if res.Code != http.StatusTooManyRequests {
		t.Fatalf("over the budget: status %d, want 429", res.Code)
	}
	limitHeaders(t, res, `"auth";q=2;w=60`, `"auth";r=0;t=30`, "30")
	if code := decode(t, res)["code"]; code != "too_many_requests" {
		t.Errorf("code = %v, want too_many_requests", code)
	}
	if n := store.count(); n != 2 {
		t.Errorf("store lookups = %d, want 2: the budget is checked before the store", n)
	}
	if res := do(h, authorize(fromAddr(t, method, target, "198.51.100.1:1"))); res.Code != http.StatusOK {
		t.Errorf("another address: status %d, want 200", res.Code)
	}
}

func TestOnlyARefusedKeySpendsTheBudget(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		auth   httpapi.Authenticator
		status int
	}{
		{"valid key", keyring{}, http.StatusOK},
		{"key not checked", keyring{err: errStoreBusy}, http.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts, _ := options(t, true)
			opts.Auth, opts.AuthLimit = tt.auth, frozenLimiter(1)
			h := httpapi.NewHandler(opts)
			for i := range 3 {
				res := do(h, authorize(fromAddr(t, http.MethodGet, "/v1/check?q=armen.ltd", "203.0.113.7:1")))
				if res.Code != tt.status {
					t.Fatalf("request #%d: status %d, want %d", i+1, res.Code, tt.status)
				}
			}
		})
	}
}

func TestAFloodOfUnknownKeysDoesNotFailAValidKey(t *testing.T) {
	t.Parallel()
	const budget = 30
	store := &keyStore{capacity: budget + 1}
	opts, _ := options(t, true)
	opts.Auth, opts.AuthLimit = store, frozenLimiter(budget)
	h := httpapi.NewHandler(opts)
	for range 10 * store.capacity {
		do(h, withKey(fromAddr(t, http.MethodGet, "/v1/check?q=armen.ltd", "203.0.113.7:1"), unknownKey))
	}
	res := do(h, authorize(fromAddr(t, http.MethodGet, "/v1/check?q=armen.ltd", "198.51.100.1:1")))
	if res.Code != http.StatusOK {
		t.Errorf("valid key after the flood: status %d, want 200", res.Code)
	}
}

func TestARequestWithoutAKeySpendsNoBudget(t *testing.T) {
	t.Parallel()
	opts, _ := options(t, true)
	opts.AuthLimit = frozenLimiter(1)
	h := httpapi.NewHandler(opts)
	do(h, fromAddr(t, http.MethodGet, "/v1/check?q=armen.ltd", "203.0.113.7:1"))
	if res := do(h, authorize(fromAddr(t, http.MethodGet, "/v1/check?q=armen.ltd", "203.0.113.7:1"))); res.Code != http.StatusOK {
		t.Errorf("valid key after a request without one: status %d, want 200", res.Code)
	}
}

func TestTheRefusalBudgetDefaultsToThirtyAMinute(t *testing.T) {
	t.Parallel()
	opts, _ := options(t, true)
	opts.Auth, opts.AuthLimit = keyring{err: app.ErrKeyNotFound}, nil
	h := httpapi.NewHandler(opts)
	var res *httptest.ResponseRecorder
	for range 31 {
		res = do(h, authorize(fromAddr(t, http.MethodGet, "/v1/check?q=armen.ltd", "203.0.113.7:1")))
	}
	if got, want := res.Header().Get("RateLimit-Policy"), `"auth";q=30;w=60`; res.Code != http.StatusTooManyRequests || got != want {
		t.Errorf("31st refused key: status %d, RateLimit-Policy %q, want 429 and %q", res.Code, got, want)
	}
}

func TestARefusedKeysAnswerHoldsToTheContract(t *testing.T) {
	t.Parallel()
	ct, err := contract()
	if err != nil {
		t.Fatal(err)
	}
	opts, _ := options(t, true)
	opts.Auth, opts.AuthLimit = keyring{err: app.ErrKeyNotFound}, frozenLimiter(1)
	h := httpapi.NewHandler(opts)
	for _, status := range []int{http.StatusUnauthorized, http.StatusTooManyRequests} {
		req := authorize(fromAddr(t, http.MethodGet, "/v1/check?q=armen.ltd", "203.0.113.7:1"))
		res := do(h, req)
		if res.Code != status {
			t.Fatalf("status %d, want %d", res.Code, status)
		}
		if err := ct.Check(t.Context(), req, res.Result()); err != nil {
			t.Errorf("the %d answer breaks the contract: %v", res.Code, err)
		}
	}
}

func TestAFullRefusalBudgetStillChecksNewClients(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	store := &keyStore{capacity: 100}
	opts, _ := options(t, true)
	opts.Auth = store
	opts.AuthLimit = ratelimit.New(ratelimit.Policy{Quota: 1, Window: time.Minute}, 1, func() time.Time { return now })
	h := httpapi.NewHandler(opts)
	do(h, withKey(fromAddr(t, http.MethodGet, "/v1/check?q=armen.ltd", "203.0.113.7:1"), unknownKey))
	if res := do(h, authorize(fromAddr(t, http.MethodGet, "/v1/check?q=armen.ltd", "198.51.100.1:1"))); res.Code != http.StatusOK {
		t.Errorf("new client on a full budget: status %d, want 200", res.Code)
	}
	if res := do(h, withKey(fromAddr(t, http.MethodGet, "/v1/check?q=armen.ltd", "198.51.100.2:1"), unknownKey)); res.Code != http.StatusUnauthorized {
		t.Errorf("new client with an unknown key on a full budget: status %d, want 401", res.Code)
	}
}
