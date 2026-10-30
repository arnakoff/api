// SPDX-License-Identifier: AGPL-3.0-only

package replay

import (
	"errors"
	"fmt"
	"sync"

	"github.com/arnakoff/api/internal/app"
)

// ErrNotRecorded means the tape holds no answer to the query: record it
// with an online run.
var ErrNotRecorded = errors.New("no recorded registry answer")

// Recorded answers other than a value.
const (
	answerNotFound   = "not_found"
	answerNotCovered = "not_covered"
)

// entry is one recorded answer: a value, or another answer.
type entry[T any] struct {
	Value  *T     `json:"value,omitempty"`
	Answer string `json:"answer,omitempty"`
}

// track holds the answers of one registry. It is safe for concurrent use.
type track[T any] struct {
	mu      sync.Mutex
	entries map[string]entry[T]
}

// play returns the answer recorded for key.
func (t *track[T]) play(key string) (T, error) {
	t.mu.Lock()
	e, ok := t.entries[key]
	t.mu.Unlock()
	var zero T
	switch {
	case !ok:
		return zero, fmt.Errorf("%w: %s", ErrNotRecorded, key)
	case e.Answer == answerNotFound:
		return zero, fmt.Errorf("replayed %s: %w", key, app.ErrNotFound)
	case e.Answer == answerNotCovered:
		return zero, fmt.Errorf("replayed %s: %w", key, app.ErrNotCovered)
	case e.Value == nil:
		return zero, fmt.Errorf("replayed %s: unknown answer %q", key, e.Answer)
	}
	return *e.Value, nil
}

// record keeps the answer v, err for key, unless err is a failure.
func (t *track[T]) record(key string, v T, err error) {
	var e entry[T]
	switch {
	case err == nil:
		e.Value = &v
	case errors.Is(err, app.ErrNotFound):
		e.Answer = answerNotFound
	case errors.Is(err, app.ErrNotCovered):
		e.Answer = answerNotCovered
	default:
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.entries == nil {
		t.entries = make(map[string]entry[T])
	}
	t.entries[key] = e
}
