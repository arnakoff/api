// SPDX-License-Identifier: AGPL-3.0-only

package cache

import (
	"errors"
	"fmt"
	"time"

	"github.com/arnakoff/api/internal/platform/defaults"
)

// ErrInvalidConfig means New was given a Config it cannot use.
var ErrInvalidConfig = errors.New("invalid cache config")

// Defaults applied by New to a zero Config field. They are choices: the
// registries behind a cache (RDAP, SIRENE, BODACC) change a record at most
// daily, and an answer such as "not found" may stop being true sooner.
const (
	DefaultMaxEntries  = 10_000
	DefaultTTL         = time.Hour
	DefaultNegativeTTL = 5 * time.Minute
)

// Config configures a Cache. Every zero numeric field takes its Default
// value; a negative one is invalid.
type Config struct {
	// MaxEntries is the most entries the cache holds.
	MaxEntries int

	// TTL is how long a loaded value is served.
	TTL time.Duration

	// NegativeTTL is how long an error accepted by Negative is served.
	NegativeTTL time.Duration

	// Negative reports whether a load error is a lasting answer worth
	// keeping for NegativeTTL. Nil keeps no error.
	Negative func(error) bool
}

// withDefaults returns cfg with its zero fields set to their Default value,
// or ErrInvalidConfig if a field is negative.
func (cfg Config) withDefaults() (Config, error) {
	errs := []error{
		defaults.Set("max entries", &cfg.MaxEntries, DefaultMaxEntries),
		defaults.Set("TTL", &cfg.TTL, DefaultTTL),
		defaults.Set("negative TTL", &cfg.NegativeTTL, DefaultNegativeTTL),
	}
	if err := errors.Join(errs...); err != nil {
		return Config{}, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}
	return cfg, nil
}
