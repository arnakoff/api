// SPDX-License-Identifier: AGPL-3.0-only

package arcep_test

import (
	"github.com/arnakoff/api/internal/adapter/source/arcep"
	"github.com/arnakoff/api/internal/app"
)

var _ app.SourceFetcher = (*arcep.Fetcher)(nil)
