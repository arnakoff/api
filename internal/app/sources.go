// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"errors"
	"fmt"

	"github.com/arnakoff/api/internal/domain"
)

// ErrUndescribedSource is returned by NewSourceList when a synced source
// has no SourceInfo.
var ErrUndescribedSource = errors.New("synced source without info")

// ListedSource is one source as GET /v1/sources shows it.
type ListedSource struct {
	Info domain.SourceInfo
	// Synced tells whether Sync downloads the source; false for a registry
	// queried on demand or a reference list built into the binary.
	Synced bool
	// Sync is the state of the data in service of a synced source.
	Sync SourceSync
}

// SourceList lists the sources a check consults: the synced ones, with the
// state of their data in service, then the others. It is safe for
// concurrent use.
type SourceList struct {
	sync   *Sync
	synced []domain.SourceInfo // in the order of s.feeds
	others []domain.SourceInfo
}

// NewSourceList returns the list of the sources of s, described by synced,
// one info each, then of others. It returns ErrUnknownSource for an info
// of no source of s, ErrDuplicateSource for a source described twice and
// ErrUndescribedSource for a source of s without info.
func NewSourceList(s *Sync, synced []domain.SourceInfo, others ...domain.SourceInfo) (*SourceList, error) {
	infos := make([]domain.SourceInfo, len(s.feeds))
	for _, info := range synced {
		i := s.index(info.Name)
		switch {
		case i < 0:
			return nil, fmt.Errorf("%w: %s", ErrUnknownSource, info.Name)
		case infos[i].Name != "":
			return nil, fmt.Errorf("%w: %s", ErrDuplicateSource, info.Name)
		}
		infos[i] = info
	}
	for i, info := range infos {
		if info.Name == "" {
			return nil, fmt.Errorf("%w: %s", ErrUndescribedSource, s.feeds[i].spec.Source)
		}
	}
	return &SourceList{sync: s, synced: infos, others: others}, nil
}

// Sources returns every source, the synced ones first, in the order of the
// sync.
func (l *SourceList) Sources() []ListedSource {
	states := l.sync.States()
	out := make([]ListedSource, 0, len(states)+len(l.others))
	for i, st := range states {
		out = append(out, ListedSource{Info: l.synced[i], Synced: true, Sync: st})
	}
	for _, info := range l.others {
		out = append(out, ListedSource{Info: info})
	}
	return out
}
