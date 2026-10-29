// SPDX-License-Identifier: AGPL-3.0-only

// Package defaults fills the zero fields of a configuration struct with
// their default value, so every Config in platform follows the same rule:
// zero means default, negative is invalid.
package defaults

import "fmt"

// Set sets a zero *field to def and rejects a negative one with an error
// naming the field. The caller wraps the error with its own sentinel.
func Set[T ~int | ~int64](name string, field *T, def T) error {
	switch {
	case *field < 0:
		return fmt.Errorf("negative %s %v", name, *field)
	case *field == 0:
		*field = def
	}
	return nil
}
