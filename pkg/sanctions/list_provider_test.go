package sanctions

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeIndex records what it was loaded with and answers Match from it by exact name.
type fakeIndex struct {
	mu      sync.Mutex
	entries []Entry
	loadErr error
}

func (f *fakeIndex) Load(entries []Entry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.loadErr != nil {
		return f.loadErr
	}
	f.entries = append([]Entry(nil), entries...)
	return nil
}

func (f *fakeIndex) Match(q MatchQuery) ([]Match, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Match
	for _, e := range f.entries {
		if e.Name == q.Name {
			out = append(out, Match{Source: e.Source, EntryID: e.ID, Name: e.Name, Score: 1})
		}
	}
	return out, nil
}

func fetchString(s string) Fetch {
	return func(context.Context) (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(s)), nil }
}

// parseLines: one «id|name» per line — a stand-in for a real list format.
func parseLines(src Source) Parse {
	return func(r io.Reader) ([]Entry, error) {
		b, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		var out []Entry
		for _, ln := range strings.Split(strings.TrimSpace(string(b)), "\n") {
			id, name, ok := strings.Cut(ln, "|")
			if !ok {
				return nil, errors.New("bad line")
			}
			out = append(out, Entry{Source: src, ID: id, Name: name, Kind: KindIndividual})
		}
		return out, nil
	}
}

func TestEntry_Validate(t *testing.T) {
	t.Parallel()
	ok := Entry{Source: SourceOFAC, ID: "1", Name: "Synthetic Person", Kind: KindIndividual}
	require.NoError(t, ok.Validate())
	cases := map[string]Entry{
		"unknown source": {Source: "XX", ID: "1", Name: "n", Kind: KindIndividual},
		"empty id":       {Source: SourceOFAC, ID: " ", Name: "n", Kind: KindIndividual},
		"empty name":     {Source: SourceOFAC, ID: "1", Name: "", Kind: KindIndividual},
		"unknown kind":   {Source: SourceOFAC, ID: "1", Name: "n", Kind: "PLANET"},
	}
	for name, e := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, e.Validate(), ErrInvalidEntry)
		})
	}
}

func TestListProvider_RefreshThenMatch(t *testing.T) {
	t.Parallel()
	idx := &fakeIndex{}
	p, err := NewListProvider(SourceUN, fetchString("a|Alpha Synthetic\nb|Beta Synthetic"), parseLines(SourceUN), idx)
	require.NoError(t, err)
	assert.Equal(t, SourceUN, p.Source())
	require.NoError(t, p.Refresh(context.Background()))
	got, err := p.Match(context.Background(), MatchQuery{Name: "Beta Synthetic"})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "b", got[0].EntryID)
}

func TestListProvider_RefreshFailureKeepsPreviousSnapshot(t *testing.T) {
	t.Parallel()
	idx := &fakeIndex{}
	calls := 0
	fetch := func(context.Context) (io.ReadCloser, error) {
		calls++
		if calls == 1 {
			return io.NopCloser(strings.NewReader("a|Alpha Synthetic")), nil
		}
		return io.NopCloser(strings.NewReader("garbage without separator")), nil
	}
	p, err := NewListProvider(SourceEU, fetch, parseLines(SourceEU), idx)
	require.NoError(t, err)
	require.NoError(t, p.Refresh(context.Background()))
	require.Error(t, p.Refresh(context.Background()))
	got, err := p.Match(context.Background(), MatchQuery{Name: "Alpha Synthetic"})
	require.NoError(t, err)
	assert.Len(t, got, 1, "the previous snapshot keeps serving")
}

func TestListProvider_RefusesEntriesOfAnotherSourceOrInvalid(t *testing.T) {
	t.Parallel()
	for name, body := range map[string]string{"other source": "a|Alpha", "empty name": "a|"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			src := SourceOFAC
			parse := parseLines(SourceUN) // wrong source on purpose
			if name == "empty name" {
				parse = parseLines(SourceOFAC)
			}
			idx := &fakeIndex{}
			p, err := NewListProvider(src, fetchString(body), parse, idx)
			require.NoError(t, err)
			assert.ErrorIs(t, p.Refresh(context.Background()), ErrInvalidEntry)
			assert.Empty(t, idx.entries, "nothing is loaded from a bad snapshot")
		})
	}
}

func TestListProvider_EmptyListIsAnError(t *testing.T) {
	t.Parallel()
	p, err := NewListProvider(SourceCOAF, fetchString(""), func(io.Reader) ([]Entry, error) { return nil, nil }, &fakeIndex{})
	require.NoError(t, err)
	assert.ErrorIs(t, p.Refresh(context.Background()), ErrEmptyList)
}

func TestListProvider_MatchValidatesQuery(t *testing.T) {
	t.Parallel()
	p, err := NewListProvider(SourceOFAC, fetchString("a|A"), parseLines(SourceOFAC), &fakeIndex{})
	require.NoError(t, err)
	_, err = p.Match(context.Background(), MatchQuery{Name: " "})
	assert.ErrorIs(t, err, ErrInvalidQuery)
}

func TestNewListProvider_RequiresParts(t *testing.T) {
	t.Parallel()
	_, err := NewListProvider(SourceOFAC, nil, parseLines(SourceOFAC), &fakeIndex{})
	assert.Error(t, err)
	_, err = NewListProvider(SourceOFAC, fetchString(""), nil, &fakeIndex{})
	assert.Error(t, err)
	_, err = NewListProvider(SourceOFAC, fetchString(""), parseLines(SourceOFAC), nil)
	assert.Error(t, err)
	_, err = NewListProvider("XX", fetchString(""), parseLines(SourceOFAC), &fakeIndex{})
	assert.Error(t, err)
}

var _ SanctionsProvider = (*ListProvider)(nil)
