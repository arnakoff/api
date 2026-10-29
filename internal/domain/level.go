// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "strconv"

// Level is the severity of a signal. Levels are ordered: a greater Level is
// more severe, so levels compare with the usual operators. The zero value is
// not a valid level.
type Level int

// Signal levels, from the least to the most severe.
const (
	// LevelInfo is a neutral fact worth knowing.
	LevelInfo Level = iota + 1
	// LevelAttention is a heuristic that calls for a check before acting.
	LevelAttention
	// LevelAlert is a fact an authority published, such as a blacklist entry.
	LevelAlert
)

// String returns the level's stable French code: "info", "attention" or
// "alerte".
func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "info"
	case LevelAttention:
		return "attention"
	case LevelAlert:
		return "alerte"
	}
	return "Level(" + strconv.Itoa(int(l)) + ")"
}
