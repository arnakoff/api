// SPDX-License-Identifier: AGPL-3.0-only

package bodacc

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/cache"
	"github.com/arnakoff/api/internal/platform/throttle"
)

// BaseURL is the BODACC's open data portal (docs/sources.md).
const BaseURL = "https://bodacc-datadila.opendatasoft.com"

// recordsPath is the records endpoint of the annonces-commerciales dataset.
const recordsPath = "api/explore/v2.1/catalog/datasets/annonces-commerciales/records"

// answerTTL is how long an answer is served from the cache, a choice: the
// BODACC publishes once a day.
const answerTTL = 24 * time.Hour

// maxNotices is the most notices asked for, the API's largest limit (it
// refuses "limit=101"). The most recent come first, so a correction or a
// cancellation is never cut off from the notice it points to.
const maxNotices = 100

// selectedFields are the only fields fetched: none names a person.
var selectedFields = []string{
	"id", "registre", "dateparution", "familleavis", "typeavis",
	"publicationavis", "parution", "numeroannonce", "parutionavisprecedent",
}

// Hosts returns the hosts the client reaches, for its allowlist.
func Hosts() []string {
	return []string{"bodacc-datadila.opendatasoft.com"}
}

// Getter fetches a URL; *httpclient.Client is one.
type Getter interface {
	Get(ctx context.Context, rawURL string) ([]byte, error)
}

// Source names the registry in reports.
const Source domain.Source = "BODACC"

// Source returns Source.
func (*Client) Source() domain.Source { return Source }

// Client reads company notices from the BODACC. It is safe for concurrent
// use.
type Client struct {
	get      Getter
	records  string
	host     string
	families string
	answers  *cache.Cache[[]domain.Notice]
	throttle *throttle.Throttle
}

// New returns a Client that queries the API at baseURL with get.
func New(get Getter, baseURL string) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("bodacc base URL: %w", err)
	}
	answers, err := cache.New[[]domain.Notice](cache.Config{TTL: answerTTL})
	if err != nil {
		return nil, fmt.Errorf("bodacc answer cache: %w", err)
	}
	return &Client{
		get:      get,
		records:  u.JoinPath(recordsPath).String(),
		host:     u.Host,
		families: familyList(),
		answers:  answers,
		throttle: throttle.New(time.Now),
	}, nil
}

// Notices returns the notices about the company with SIREN n, most recent
// first, from the cache or the API. No notice is an empty list. It returns
// ErrThrottled, ErrResponse or the error of the HTTP client.
func (c *Client) Notices(ctx context.Context, n domain.SIREN) ([]domain.Notice, error) {
	notices, err := c.answers.Get(ctx, n.String(), func(ctx context.Context) ([]domain.Notice, error) {
		return c.query(ctx, n.String())
	})
	return slices.Clone(notices), err
}

// query asks the API for the notices about the company with SIREN n.
func (c *Client) query(ctx context.Context, n string) ([]domain.Notice, error) {
	if !c.throttle.Allows(c.host) {
		return nil, fmt.Errorf("bodacc %s: %w", n, ErrThrottled)
	}
	body, err := c.get.Get(ctx, c.recordsURL(n))
	if err != nil {
		c.throttle.Heed(c.host, err)
		return nil, fmt.Errorf("bodacc %s: %w", n, err)
	}
	notices, err := parseNotices(body, n)
	if err != nil {
		return nil, fmt.Errorf("bodacc %s: %w", n, err)
	}
	return notices, nil
}

// recordsURL asks for the notices of the families read about SIREN n, a
// validated string of nine digits, so it cannot alter the filter.
func (c *Client) recordsURL(n string) string {
	q := url.Values{
		"where":    {`registre="` + n + `" and familleavis in ` + c.families},
		"select":   {strings.Join(selectedFields, ",")},
		"order_by": {"dateparution desc"},
		"limit":    {strconv.Itoa(maxNotices)},
	}
	return c.records + "?" + q.Encode()
}

// familyList returns the families read as an ODSQL list, in a stable
// order: ("collective","conciliation",…).
func familyList() string {
	quoted := make([]string, 0, len(kinds))
	for _, f := range slices.Sorted(maps.Keys(kinds)) {
		quoted = append(quoted, strconv.Quote(f))
	}
	return "(" + strings.Join(quoted, ",") + ")"
}
