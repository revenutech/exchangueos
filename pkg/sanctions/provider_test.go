package sanctions_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenutech/exchangeos/pkg/sanctions"
)

type fakeProvider struct{ entries []sanctions.Match }

func (f *fakeProvider) Source() sanctions.Source       { return sanctions.SourceOFAC }
func (f *fakeProvider) Refresh(context.Context) error { return nil }
func (f *fakeProvider) Match(_ context.Context, q sanctions.MatchQuery) ([]sanctions.Match, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	var out []sanctions.Match
	for _, e := range f.entries {
		if e.Name == q.Name && e.Score >= q.MinScore {
			out = append(out, e)
		}
	}
	return out, nil
}

var _ sanctions.SanctionsProvider = (*fakeProvider)(nil)

func TestMatchQueryValidate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		q    sanctions.MatchQuery
		ok   bool
	}{
		{"valid", sanctions.MatchQuery{Name: "ACME TRADING", MinScore: 0.85}, true},
		{"zero score accepts all", sanctions.MatchQuery{Name: "ACME"}, true},
		{"empty name", sanctions.MatchQuery{Name: "  ", MinScore: 0.9}, false},
		{"score above one", sanctions.MatchQuery{Name: "ACME", MinScore: 1.1}, false},
		{"negative score", sanctions.MatchQuery{Name: "ACME", MinScore: -0.1}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.q.Validate()
			if tc.ok {
				assert.NoError(t, err)
				return
			}
			assert.True(t, errors.Is(err, sanctions.ErrInvalidQuery))
		})
	}
}

func TestProviderContract(t *testing.T) {
	t.Parallel()
	var p sanctions.SanctionsProvider = &fakeProvider{entries: []sanctions.Match{
		{Source: sanctions.SourceOFAC, EntryID: "SDN-1", Name: "ACME TRADING", Score: 0.97},
	}}
	require.NoError(t, p.Refresh(context.Background()))
	got, err := p.Match(context.Background(), sanctions.MatchQuery{Name: "ACME TRADING", MinScore: 0.9})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "SDN-1", got[0].EntryID)
	_, err = p.Match(context.Background(), sanctions.MatchQuery{})
	assert.ErrorIs(t, err, sanctions.ErrInvalidQuery)
}
