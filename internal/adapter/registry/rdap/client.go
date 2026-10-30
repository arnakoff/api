// SPDX-License-Identifier: AGPL-3.0-only

package rdap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/cache"
	"github.com/arnakoff/api/internal/platform/httpclient"
	"github.com/arnakoff/api/internal/platform/throttle"
)

// BootstrapURL is the IANA bootstrap file for domain names (RFC 9224).
const BootstrapURL = "https://data.iana.org/rdap/dns.json"

// Cache lifetimes. They are choices: IANA publishes the bootstrap file with
// "Cache-Control: max-age=86400" (checked on 30/10/2026), and a domain's
// registration date does not change.
const (
	bootstrapTTL = 24 * time.Hour
	answerTTL    = 24 * time.Hour
)

// bootstrapKey is the only key of the bootstrap cache.
const bootstrapKey = "dns"

// BootstrapHosts returns the hosts the bootstrap client reaches, for its
// allowlist.
func BootstrapHosts() []string {
	return []string{"data.iana.org"}
}

// Getter fetches a URL; *httpclient.Client is one.
type Getter interface {
	Get(ctx context.Context, rawURL string) ([]byte, error)
}

// Source names the registry in reports.
const Source domain.Source = "RDAP"

// Source returns Source.
func (*Client) Source() domain.Source { return Source }

// Client reads the registration dates of domains from the RDAP server of
// their registry. It is safe for concurrent use.
type Client struct {
	iana         Getter
	bootstrapURL string
	dial         func(hosts []string) (Getter, error)
	registries   *cache.Cache[registries]
	answers      *cache.Cache[domain.Registration]
	throttle     *throttle.Throttle
}

// registries is a bootstrap file and the client allowed to reach the
// servers it names.
type registries struct {
	dir    directory
	client Getter
}

// New returns a Client that reads the bootstrap file at bootstrapURL with
// iana, and queries registries with the client dial builds for the hosts the
// file names: the allowlist follows the file, and a user input only ever
// selects one of its servers.
func New(iana Getter, bootstrapURL string, dial func(hosts []string) (Getter, error)) (*Client, error) {
	regs, err := cache.New[registries](cache.Config{
		MaxEntries: 1,
		TTL:        bootstrapTTL,
		Negative:   func(error) bool { return true },
	})
	if err != nil {
		return nil, fmt.Errorf("rdap bootstrap cache: %w", err)
	}
	answers, err := cache.New[domain.Registration](cache.Config{TTL: answerTTL, Negative: lasting})
	if err != nil {
		return nil, fmt.Errorf("rdap answer cache: %w", err)
	}
	return &Client{
		iana:         iana,
		bootstrapURL: bootstrapURL,
		dial:         dial,
		registries:   regs,
		answers:      answers,
		throttle:     throttle.New(time.Now),
	}, nil
}

// lasting reports whether err is an answer worth caching for a while.
func lasting(err error) bool {
	return errors.Is(err, ErrNotFound) || errors.Is(err, ErrNoServer)
}

// Lookup returns the registration dates of d's registrable domain, from the
// cache or its registry. It returns ErrNoServer, ErrNotFound, ErrThrottled,
// ErrBootstrap, ErrResponse or the error of the HTTP client.
func (c *Client) Lookup(ctx context.Context, d domain.Domain) (domain.Registration, error) {
	name := d.Registrable()
	return c.answers.Get(ctx, name, func(ctx context.Context) (domain.Registration, error) {
		return c.query(ctx, name)
	})
}

// query asks the registry of name for it.
func (c *Client) query(ctx context.Context, name string) (domain.Registration, error) {
	regs, err := c.registries.Get(ctx, bootstrapKey, c.loadRegistries)
	if err != nil {
		return domain.Registration{}, err
	}
	base, ok := regs.dir.server(name)
	if !ok {
		return domain.Registration{}, fmt.Errorf("rdap %s: %w", name, ErrNoServer)
	}
	u, err := url.Parse(base + "domain/" + name)
	if err != nil {
		return domain.Registration{}, fmt.Errorf("rdap %s: %w", name, err)
	}
	if !c.throttle.Allows(u.Host) {
		return domain.Registration{}, fmt.Errorf("rdap %s: %w", name, ErrThrottled)
	}
	body, err := regs.client.Get(ctx, u.String())
	if err != nil {
		return domain.Registration{}, c.queryError(name, u.Host, err)
	}
	reg, err := parseRegistration(body)
	if err != nil {
		return domain.Registration{}, fmt.Errorf("rdap %s: %w", name, err)
	}
	reg.URL = u.String()
	return reg, nil
}

// queryError reads a 404 as ErrNotFound and keeps away from a host that
// asked to wait.
func (c *Client) queryError(name, host string, err error) error {
	if se, ok := errors.AsType[*httpclient.StatusError](err); ok && se.StatusCode == http.StatusNotFound {
		return fmt.Errorf("rdap %s: %w", name, ErrNotFound)
	}
	c.throttle.Heed(host, err)
	return fmt.Errorf("rdap %s: %w", name, err)
}

// loadRegistries fetches the bootstrap file and builds the registry client
// for the hosts it names.
func (c *Client) loadRegistries(ctx context.Context) (registries, error) {
	body, err := c.iana.Get(ctx, c.bootstrapURL)
	if err != nil {
		return registries{}, fmt.Errorf("rdap bootstrap: %w", err)
	}
	dir, err := parseDirectory(body)
	if err != nil {
		return registries{}, fmt.Errorf("rdap bootstrap: %w", err)
	}
	client, err := c.dial(dir.hosts())
	if err != nil {
		return registries{}, fmt.Errorf("rdap registry client: %w", err)
	}
	return registries{dir: dir, client: client}, nil
}
