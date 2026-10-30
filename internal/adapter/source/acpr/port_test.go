// SPDX-License-Identifier: AGPL-3.0-only

package acpr_test

import (
	"github.com/arnakoff/api/internal/adapter/source/acpr"
	"github.com/arnakoff/api/internal/app"
)

var _ app.SourceFetcher = (*acpr.Fetcher)(nil)
