package application_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenutech/exchangeos/modules/compliance/application"
	"github.com/revenutech/exchangeos/modules/compliance/domain"
	"github.com/revenutech/exchangeos/pkg/sanctions"
)

// screenedName is synthetic; the tests also use it to check that no error carries it.
const screenedName = "Synthetic Counterparty Bank"

// fakeListProvider stands in for one live sanctions list.
type fakeListProvider struct {
	src     sanctions.Source
	matches []sanctions.Match
	err     error

	mu      sync.Mutex
	queries []sanctions.MatchQuery
}

func (f *fakeListProvider) Source() sanctions.Source      { return f.src }
func (f *fakeListProvider) Refresh(context.Context) error { return nil }
func (f *fakeListProvider) Match(_ context.Context, q sanctions.MatchQuery) ([]sanctions.Match, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queries = append(f.queries, q)
	if f.err != nil {
		return nil, f.err
	}
	return append([]sanctions.Match(nil), f.matches...), nil
}

func (f *fakeListProvider) calls() []sanctions.MatchQuery {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]sanctions.MatchQuery(nil), f.queries...)
}

var _ sanctions.SanctionsProvider = (*fakeListProvider)(nil)

// fakeNames resolves every counterparty to one synthetic name.
type fakeNames struct {
	name string
	err  error

	mu    sync.Mutex
	asked []struct{ bic, lei string }
}

func (f *fakeNames) CounterpartyName(_ context.Context, bic, lei string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.asked = append(f.asked, struct{ bic, lei string }{bic, lei})
	return f.name, f.err
}

func hit(src sanctions.Source, id string, score float64) sanctions.Match {
	return sanctions.Match{Source: src, EntryID: id, Name: "Listed Synthetic Entity", Score: score}
}

// fourLists returns one fake provider per required list, in a fixed order, with the given matches.
func fourLists(ofac, un, eu, coaf []sanctions.Match) []*fakeListProvider {
	return []*fakeListProvider{
		{src: sanctions.SourceOFAC, matches: ofac},
		{src: sanctions.SourceUN, matches: un},
		{src: sanctions.SourceEU, matches: eu},
		{src: sanctions.SourceCOAF, matches: coaf},
	}
}

func asProviders(fs []*fakeListProvider) []sanctions.SanctionsProvider {
	out := make([]sanctions.SanctionsProvider, len(fs))
	for i, f := range fs {
		out[i] = f
	}
	return out
}

func newProviderScreener(t *testing.T, names application.CounterpartyNameResolver, fs []*fakeListProvider) *application.ProviderScreener {
	t.Helper()
	s, err := application.NewProviderScreener(names, asProviders(fs), application.DefaultSanctionsMinScore)
	require.NoError(t, err)
	return s
}

func screenInput() domain.NewScreeningInput {
	return domain.NewScreeningInput{TenantID: uuid.New(), CounterpartyBIC: "SYNTBRSP", LEI: "5299000SYNTHETIC0001"}
}

func TestRN_FX_039_ScreenCounterparty_QueriesAllFourLists_HighestScoreWins(t *testing.T) {
	t.Parallel()
	fs := fourLists(
		[]sanctions.Match{hit(sanctions.SourceOFAC, "ofac-1", 0.91)},
		[]sanctions.Match{hit(sanctions.SourceUN, "un-7", 0.97)},
		nil,
		[]sanctions.Match{hit(sanctions.SourceCOAF, "coaf-3", 0.88)},
	)
	names := &fakeNames{name: screenedName}
	svc, repo, _ := newSvcWith(t, newProviderScreener(t, names, fs))

	in := screenInput()
	res, err := svc.ScreenCounterparty(context.Background(), in)
	require.NoError(t, err)

	assert.Equal(t, []string{"UN:un-7", "OFAC:ofac-1", "COAF:coaf-3"}, res.Hits(), "best score first, across lists")
	assert.Equal(t, domain.RiskHigh, res.RiskLevel())
	require.Len(t, repo.Saved, 1)

	require.Len(t, names.asked, 1)
	assert.Equal(t, in.CounterpartyBIC, names.asked[0].bic)
	assert.Equal(t, in.LEI, names.asked[0].lei)
	for _, f := range fs {
		qs := f.calls()
		require.Len(t, qs, 1, "%s must be queried exactly once", f.src)
		assert.Equal(t, screenedName, qs[0].Name, "%s got another name", f.src)
		assert.Equal(t, application.DefaultSanctionsMinScore, qs[0].MinScore)
		assert.Empty(t, qs[0].Country, "the country is not used to narrow a sanctions screen")
	}
}

func TestRN_FX_039_ScreenCounterparty_HitOrdering(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		fs   []*fakeListProvider
		want []string
	}{
		{
			name: "all lists clear",
			fs:   fourLists(nil, nil, nil, nil),
			want: []string{},
		},
		{
			name: "single list, several entries",
			fs: fourLists(nil, nil,
				[]sanctions.Match{hit(sanctions.SourceEU, "eu-2", 0.86), hit(sanctions.SourceEU, "eu-1", 0.99)}, nil),
			want: []string{"EU:eu-1", "EU:eu-2"},
		},
		{
			name: "equal scores break by source then entry id",
			fs: fourLists(
				[]sanctions.Match{hit(sanctions.SourceOFAC, "b", 0.9), hit(sanctions.SourceOFAC, "a", 0.9)},
				nil, nil,
				[]sanctions.Match{hit(sanctions.SourceCOAF, "z", 0.9)},
			),
			want: []string{"COAF:z", "OFAC:a", "OFAC:b"},
		},
		{
			name: "a duplicate entry keeps its best score",
			fs: fourLists(
				[]sanctions.Match{hit(sanctions.SourceOFAC, "dup", 0.86), hit(sanctions.SourceOFAC, "dup", 0.95)},
				[]sanctions.Match{hit(sanctions.SourceUN, "un-1", 0.9)},
				nil, nil,
			),
			want: []string{"OFAC:dup", "UN:un-1"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newProviderScreener(t, &fakeNames{name: screenedName}, tc.fs)
			hits, err := s.Screen(context.Background(), "SYNTBRSP", "")
			require.NoError(t, err)
			assert.Equal(t, tc.want, append([]string{}, hits...))
			for _, f := range tc.fs {
				assert.Len(t, f.calls(), 1, "%s must be queried", f.src)
			}
		})
	}
}

// A list that cannot answer makes the whole screening fail: a counterparty is never cleared on three lists out of
// four. Nothing is persisted, and the error names the list, never the counterparty.
func TestRN_FX_039_ScreenCounterparty_ProviderErrorFailsClosed(t *testing.T) {
	t.Parallel()
	boom := errors.New("synthetic list outage")
	for _, failing := range []sanctions.Source{sanctions.SourceOFAC, sanctions.SourceUN, sanctions.SourceEU, sanctions.SourceCOAF} {
		t.Run(string(failing), func(t *testing.T) {
			t.Parallel()
			fs := fourLists(nil, nil, nil, nil)
			for _, f := range fs {
				f.matches = []sanctions.Match{hit(f.src, "x", 0.99)}
				if f.src == failing {
					f.err = boom
				}
			}
			svc, repo, _ := newSvcWith(t, newProviderScreener(t, &fakeNames{name: screenedName}, fs))

			res, err := svc.ScreenCounterparty(context.Background(), screenInput())
			require.Error(t, err)
			assert.Nil(t, res)
			assert.ErrorIs(t, err, application.ErrScreeningFailed)
			assert.ErrorIs(t, err, boom)
			assert.Contains(t, err.Error(), string(failing))
			assert.NotContains(t, err.Error(), screenedName)
			assert.Empty(t, repo.Saved, "a failed screening must not be recorded")
		})
	}
}

func TestRN_FX_039_ScreenCounterparty_UnresolvedNameFailsClosed(t *testing.T) {
	t.Parallel()
	notFound := errors.New("synthetic directory miss")
	cases := []struct {
		name  string
		names *fakeNames
		want  error
	}{
		{"resolver error", &fakeNames{err: notFound}, notFound},
		{"empty name", &fakeNames{name: ""}, application.ErrScreeningFailed},
		{"blank name", &fakeNames{name: "   "}, application.ErrScreeningFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fs := fourLists(nil, nil, nil, nil)
			svc, repo, _ := newSvcWith(t, newProviderScreener(t, tc.names, fs))

			_, err := svc.ScreenCounterparty(context.Background(), screenInput())
			assert.ErrorIs(t, err, application.ErrScreeningFailed)
			assert.ErrorIs(t, err, tc.want)
			assert.Empty(t, repo.Saved)
			for _, f := range fs {
				assert.Empty(t, f.calls(), "no list is queried without a name")
			}
		})
	}
}

func TestScreenCounterparty_CancelledContextFailsClosed(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fs := fourLists(nil, nil, nil, nil)
	s := newProviderScreener(t, &fakeNames{name: screenedName}, fs)

	_, err := s.Screen(ctx, "SYNTBRSP", "")
	assert.ErrorIs(t, err, application.ErrScreeningFailed)
	assert.ErrorIs(t, err, context.Canceled)
	for _, f := range fs {
		assert.Empty(t, f.calls())
	}
}

func TestNewProviderScreener_Validation(t *testing.T) {
	t.Parallel()
	names := &fakeNames{name: screenedName}
	all := func() []sanctions.SanctionsProvider { return asProviders(fourLists(nil, nil, nil, nil)) }
	cases := []struct {
		name      string
		names     application.CounterpartyNameResolver
		providers func() []sanctions.SanctionsProvider
		minScore  float64
	}{
		{"nil resolver", nil, all, 0.85},
		{"no providers", names, func() []sanctions.SanctionsProvider { return nil }, 0.85},
		{"missing COAF", names, func() []sanctions.SanctionsProvider { return all()[:3] }, 0.85},
		{"duplicate OFAC", names, func() []sanctions.SanctionsProvider {
			return append(all(), &fakeListProvider{src: sanctions.SourceOFAC})
		}, 0.85},
		{"nil provider", names, func() []sanctions.SanctionsProvider { return append(all(), nil) }, 0.85},
		{"zero threshold", names, all, 0},
		{"threshold above one", names, all, 1.01},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s, err := application.NewProviderScreener(tc.names, tc.providers(), tc.minScore)
			assert.ErrorIs(t, err, application.ErrInvalidInput)
			assert.Nil(t, s)
		})
	}

	s, err := application.NewProviderScreener(names, all(), application.DefaultSanctionsMinScore)
	require.NoError(t, err)
	assert.NotNil(t, s)
}

var _ application.SanctionsScreener = (*application.ProviderScreener)(nil)
