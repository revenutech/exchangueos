// Package sanctions defines the contract for sanctions-list providers used by
// counterparty screening (OFAC SDN, UN 1267, EU restrictive measures, COAF).
package sanctions

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidQuery is returned by MatchQuery.Validate and by Match when the
// query cannot be evaluated.
var ErrInvalidQuery = errors.New("sanctions: invalid match query")

// Source identifies the sanctions list an entry comes from.
type Source string

const (
	SourceOFAC Source = "OFAC"
	SourceUN   Source = "UN"
	SourceEU   Source = "EU"
	SourceCOAF Source = "COAF"
)

// MatchQuery is a screening request against one list.
type MatchQuery struct {
	// Name is the party name as captured, before normalisation. Required.
	Name string
	// Country is an optional ISO 3166-1 alpha-2 code that narrows candidates.
	Country string
	// MinScore is the similarity threshold in [0, 1]; zero returns every candidate.
	MinScore float64
}

// Validate reports whether the query can be evaluated.
func (q MatchQuery) Validate() error {
	if strings.TrimSpace(q.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidQuery)
	}
	if q.MinScore < 0 || q.MinScore > 1 {
		return fmt.Errorf("%w: min score %v outside [0, 1]", ErrInvalidQuery, q.MinScore)
	}
	return nil
}

// Match is one list entry whose similarity to the query reached MinScore.
// Score is a similarity in [0, 1], not a monetary value.
type Match struct {
	Source  Source
	EntryID string
	Name    string
	Score   float64
}

// SanctionsProvider is implemented once per list.
type SanctionsProvider interface {
	// Source returns the list this provider serves.
	Source() Source
	// Refresh downloads the list and atomically swaps the in-memory snapshot.
	// On failure the previous snapshot keeps serving Match.
	Refresh(ctx context.Context) error
	// Match returns the entries whose similarity to q is at least q.MinScore,
	// best score first. It returns ErrInvalidQuery when q.Validate fails.
	Match(ctx context.Context, q MatchQuery) ([]Match, error)
}
