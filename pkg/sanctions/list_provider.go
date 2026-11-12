package sanctions

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

var (
	// ErrInvalidEntry is returned when a parsed list entry cannot be indexed.
	ErrInvalidEntry = errors.New("sanctions: invalid list entry")
	// ErrEmptyList is returned when a refresh parses zero entries: an empty sanctions list is a broken download,
	// never a reason to stop matching.
	ErrEmptyList = errors.New("sanctions: list parsed to zero entries")
)

// EntryKind is what a list entry designates.
type EntryKind string

const (
	KindIndividual EntryKind = "INDIVIDUAL"
	KindEntity     EntryKind = "ENTITY"
	KindVessel     EntryKind = "VESSEL"
	KindAircraft   EntryKind = "AIRCRAFT"
	KindOther      EntryKind = "OTHER"
)

func validSource(s Source) bool {
	switch s {
	case SourceOFAC, SourceUN, SourceEU, SourceCOAF:
		return true
	}
	return false
}

// Entry is one designation on a sanctions list, normalised across OFAC, UN, EU and COAF. The names are those
// published by the list itself (public data), not data captured from our counterparties.
type Entry struct {
	Source    Source
	ID        string    // the list's own identifier (OFAC uid, UN reference number, EU logical id, COAF row id)
	Name      string    // primary name as published
	Aliases   []string  // a.k.a. names, as published
	Kind      EntryKind //
	Countries []string  // ISO 3166-1 alpha-2, when the list gives them
	Programs  []string  // sanctions programs / regimes, as published
}

// Validate reports whether the entry can be indexed.
func (e Entry) Validate() error {
	switch {
	case !validSource(e.Source):
		return fmt.Errorf("%w: unknown source %q", ErrInvalidEntry, e.Source)
	case strings.TrimSpace(e.ID) == "":
		return fmt.Errorf("%w: %s entry without id", ErrInvalidEntry, e.Source)
	case strings.TrimSpace(e.Name) == "":
		return fmt.Errorf("%w: %s entry %s without name", ErrInvalidEntry, e.Source, e.ID)
	}
	switch e.Kind {
	case KindIndividual, KindEntity, KindVessel, KindAircraft, KindOther:
		return nil
	}
	return fmt.Errorf("%w: %s entry %s with unknown kind %q", ErrInvalidEntry, e.Source, e.ID, e.Kind)
}

// Index is the in-memory matcher a ListProvider loads each snapshot into. The cache of MS-024d 1.2 implements it.
// Load replaces the whole content atomically; Match must be safe for concurrent use with Load.
type Index interface {
	Load(entries []Entry) error
	Match(q MatchQuery) ([]Match, error)
}

// Fetch opens the list as published (HTTP download in production, a fixture in tests).
type Fetch func(ctx context.Context) (io.ReadCloser, error)

// Parse turns one published list into entries. It is the only list-specific part of a provider.
type Parse func(r io.Reader) ([]Entry, error)

// ListProvider is the SanctionsProvider shared by every list: fetch → parse → validate → load into the index.
// A failed refresh (fetch, parse, an invalid entry, an entry of another source, or an empty list) leaves the
// previous snapshot serving Match.
type ListProvider struct {
	source Source
	fetch  Fetch
	parse  Parse
	index  Index
	mu     sync.Mutex // serialises Refresh; Match goes straight to the index
}

// NewListProvider wires a provider for one list.
func NewListProvider(source Source, fetch Fetch, parse Parse, index Index) (*ListProvider, error) {
	switch {
	case !validSource(source):
		return nil, fmt.Errorf("sanctions: unknown source %q", source)
	case fetch == nil, parse == nil, index == nil:
		return nil, errors.New("sanctions: fetch, parse and index are required")
	}
	return &ListProvider{source: source, fetch: fetch, parse: parse, index: index}, nil
}

// Source returns the list this provider serves.
func (p *ListProvider) Source() Source { return p.source }

// Refresh downloads, parses and validates the list, then swaps it into the index.
func (p *ListProvider) Refresh(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	rc, err := p.fetch(ctx)
	if err != nil {
		return fmt.Errorf("sanctions: fetch %s: %w", p.source, err)
	}
	defer func() { _ = rc.Close() }()
	entries, err := p.parse(rc)
	if err != nil {
		return fmt.Errorf("sanctions: parse %s: %w", p.source, err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("%w: %s", ErrEmptyList, p.source)
	}
	for _, e := range entries {
		if e.Source != p.source {
			return fmt.Errorf("%w: %s provider got an entry of %q", ErrInvalidEntry, p.source, e.Source)
		}
		if err := e.Validate(); err != nil {
			return err
		}
	}
	if err := p.index.Load(entries); err != nil {
		return fmt.Errorf("sanctions: load %s: %w", p.source, err)
	}
	return nil
}

// Match validates the query and asks the index.
func (p *ListProvider) Match(_ context.Context, q MatchQuery) ([]Match, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	return p.index.Match(q)
}
