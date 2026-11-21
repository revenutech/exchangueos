// Package sanctions defines the contract for sanctions-list providers used by
// counterparty screening (OFAC SDN, UN 1267, EU restrictive measures, COAF).
package sanctions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
//
// PII: Name is the counterparty's name. String, GoString, LogValue and MarshalJSON never print it, only the country,
// the threshold and the name length. They use value receivers on purpose, so a pointer or a dereferenced copy is
// redacted as well. MarshalJSON is a redacted view, not a wire format: a MatchQuery does not round-trip through JSON.
//
// fmt calls these methods only on exported fields. A MatchQuery kept in an unexported field of another struct is
// printed raw, Name included, by %v, %+v or %#v of the outer struct (and encoding/json skips unexported fields). Never
// keep a MatchQuery in an unexported field of a struct that gets formatted.
type MatchQuery struct {
	// Name is the party name as captured, before normalisation. Required.
	Name string
	// Country is an optional ISO 3166-1 alpha-2 code that narrows candidates.
	Country string
	// MinScore is the similarity threshold in [0, 1]; zero returns every candidate.
	MinScore float64
}

func (q MatchQuery) redacted() string {
	return fmt.Sprintf("MatchQuery{Country:%s MinScore:%v name_len:%d}", q.Country, q.MinScore, len(q.Name))
}

// String never includes Name: the party name is PII. fmt uses it for %v, %+v and %s.
func (q MatchQuery) String() string { return q.redacted() }

// GoString covers %#v; it never includes Name.
func (q MatchQuery) GoString() string { return q.redacted() }

// MarshalJSON is the redacted view (country, min_score, name_len); it never includes Name.
func (q MatchQuery) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Country  string  `json:"country"`
		MinScore float64 `json:"min_score"`
		NameLen  int     `json:"name_len"`
	}{q.Country, q.MinScore, len(q.Name)})
}

// LogValue never includes Name: the party name is PII.
func (q MatchQuery) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("country", q.Country),
		slog.Float64("min_score", q.MinScore),
		slog.Int("name_len", len(q.Name)),
	)
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
