// SPDX-License-Identifier: AGPL-3.0-only

// Package csvfile reads the semicolon- or comma-separated files upstream
// sources publish, record by record, after checking their header. Its
// errors never quote a value of the file: it may hold personal data.
package csvfile
