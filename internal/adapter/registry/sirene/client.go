// SPDX-License-Identifier: AGPL-3.0-only

package sirene

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"golang.org/x/time/rate"

	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
	"github.com/arnakoff/api/internal/platform/cache"
	"github.com/arnakoff/api/internal/platform/httpclient"
	"github.com/arnakoff/api/internal/platform/throttle"
)

// BaseURL is the API Recherche d'entreprises (docs/sources.md).
const BaseURL = "https://recherche-entreprises.api.gouv.fr"

// MaxRequestsPerSecond is the API's published limit per IP address: "au
// maximum 7 requêtes par seconde" (https://recherche-entreprises.api.gouv.fr/docs/,
// checked on 30/10/2026 and 06/11/2026). It is a ceiling, not a guaranteed
// rate: the API may also answer 429 with a Retry-After, which the client
// heeds. Instances that may share one outgoing address share it (New).
const MaxRequestsPerSecond rate.Limit = 7

// answerTTL is how long an answer is served from the cache, a choice: a
// company's name, creation date and head office state change rarely.
const answerTTL = 24 * time.Hour

// maxRateWait bounds the wait for the request rate, a choice: a second of
// requests queued ahead, seven for a single instance. Beyond it the lookup
// fails rather than pile up, and a cache load does not see the caller's
// deadline.
const maxRateWait = time.Second

// perPage is the number of results asked for. A SIREN search is direct,
// but the result matching the SIREN is still picked among a few.
const perPage = "5"

// Hosts returns the hosts the client reaches, for its allowlist.
func Hosts() []string {
	return []string{"recherche-entreprises.api.gouv.fr"}
}

// HTTPConfig returns the configuration of the HTTP client a Client needs:
// its hosts and a single attempt. The request rate and the API's
// Retry-After are checked once per query, so a retry inside the HTTP
// client would get past both; a refused query is retried by the next
// lookup instead (#343).
func HTTPConfig() httpclient.Config {
	return httpclient.Config{AllowedHosts: Hosts(), Attempts: 1}
}

// Getter fetches a URL; *httpclient.Client is one.
type Getter interface {
	Get(ctx context.Context, rawURL string) ([]byte, error)
}

// Source names the registry in reports.
const Source = signal.SIRENESource

// Source returns Source.
func (*Client) Source() domain.Source { return Source }

// Client reads companies from the API Recherche d'entreprises. It is safe
// for concurrent use.
type Client struct {
	get      Getter
	search   string
	host     string
	answers  *cache.Cache[domain.Company]
	limiter  *rate.Limiter
	throttle *throttle.Throttle
}

// New returns a Client that queries the API at baseURL with get, at most
// its share of MaxRequestsPerSecond among instances, the most processes
// querying the API at once: the platform does not tell whether they share
// one outgoing IP address, so together they keep to the limit of one.
func New(get Getter, baseURL string, instances int) (*Client, error) {
	if instances < 1 {
		return nil, fmt.Errorf("sirene instances %d: %w", instances, errNoInstance)
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("sirene base URL: %w", err)
	}
	answers, err := cache.New[domain.Company](cache.Config{
		TTL:      answerTTL,
		Negative: func(err error) bool { return errors.Is(err, ErrNotFound) },
	})
	if err != nil {
		return nil, fmt.Errorf("sirene answer cache: %w", err)
	}
	return &Client{
		get:      get,
		search:   u.JoinPath("search").String(),
		host:     u.Host,
		answers:  answers,
		limiter:  rate.NewLimiter(MaxRequestsPerSecond/rate.Limit(instances), 1),
		throttle: throttle.New(time.Now),
	}, nil
}

// Lookup returns the company with SIREN n, from the cache or the API. It
// waits at most maxRateWait for the request rate. It returns
// ErrNotFound, ErrThrottled, ErrResponse or the error of the HTTP client.
func (c *Client) Lookup(ctx context.Context, n domain.SIREN) (domain.Company, error) {
	return c.answers.Get(ctx, n.String(), func(ctx context.Context) (domain.Company, error) {
		return c.query(ctx, n.String())
	})
}

// query asks the API for the company with SIREN n.
func (c *Client) query(ctx context.Context, n string) (domain.Company, error) {
	if !c.throttle.Allows(c.host) {
		return domain.Company{}, fmt.Errorf("sirene %s: %w", n, ErrThrottled)
	}
	if err := c.waitForRate(ctx); err != nil {
		return domain.Company{}, fmt.Errorf("sirene %s: %w: %w", n, ErrThrottled, err)
	}
	search := c.searchURL(n)
	body, err := c.get.Get(ctx, search)
	if err != nil {
		c.throttle.Heed(c.host, err)
		return domain.Company{}, fmt.Errorf("sirene %s: %w", n, err)
	}
	company, err := parseCompany(body, n)
	if err != nil {
		return domain.Company{}, fmt.Errorf("sirene %s: %w", n, err)
	}
	company.URL = search
	return company, nil
}

// waitForRate waits for the request rate, at most maxRateWait. It returns
// at once when the wait would be longer.
func (c *Client) waitForRate(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, maxRateWait)
	defer cancel()
	return c.limiter.Wait(ctx)
}

// searchURL asks for the head office only: the minimal answer leaves out
// the dirigeants, who are people, so they are never fetched.
func (c *Client) searchURL(n string) string {
	q := url.Values{
		"q":        {n},
		"minimal":  {"true"},
		"include":  {"siege"},
		"per_page": {perPage},
	}
	return c.search + "?" + q.Encode()
}
