// SPDX-License-Identifier: AGPL-3.0-only

package api_test

import "testing"

// Throwaway test for issue #6: proves a red CI blocks the merge. Never merged.
func TestCIRedCheck(t *testing.T) {
	t.Parallel()
	t.Fatal("intentional failure")
}
