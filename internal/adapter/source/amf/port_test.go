// SPDX-License-Identifier: AGPL-3.0-only

package amf_test

import (
	"github.com/arnakoff/api/internal/adapter/source/amf"
	"github.com/arnakoff/api/internal/app"
)

var _ app.SourceFetcher = (*amf.Fetcher)(nil)
