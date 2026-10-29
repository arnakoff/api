// SPDX-License-Identifier: AGPL-3.0-only

package store

import (
	"fmt"
	"strings"

	"github.com/arnakoff/api/internal/domain"
)

// fileExt ends the name of every snapshot file.
const fileExt = ".snapshot"

// fileName returns the name of the file of source (see the package doc), or
// ErrSourceName when source has no letter or digit. The name never holds a
// path separator or a dot before its extension, so it stays in the store's
// directory.
func fileName(source domain.Source) (string, error) {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(string(source)) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			dash = false
			continue
		}
		dash = true
	}
	if b.Len() == 0 {
		return "", fmt.Errorf("%w: %q", ErrSourceName, source)
	}
	return b.String() + fileExt, nil
}
