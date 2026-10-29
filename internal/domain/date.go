// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "time"

// frenchDateLayout is the JJ/MM/AAAA layout used in every French message.
const frenchDateLayout = "02/01/2006"

// FormatDate formats t as a French date, JJ/MM/AAAA, in t's own location.
// The caller picks the location (Europe/Paris for user-facing dates): the
// domain loads no time zone data.
func FormatDate(t time.Time) string {
	return t.Format(frenchDateLayout)
}
