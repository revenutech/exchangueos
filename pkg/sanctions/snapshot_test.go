package sanctions

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var snapTime = time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

func snapEntries() []Entry {
	return []Entry{
		{Source: SourceOFAC, ID: "990001", Name: "Elvaro QUINTESSA", Aliases: []string{"QUINTESSA"}, Kind: KindIndividual,
			Countries: []string{"CO"}, Programs: []string{"SDNTK"}},
		{Source: SourceOFAC, ID: "990002", Name: "Mardok Shipping", Kind: KindEntity},
	}
}

func TestSnapshotPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		source Source
		want   string
	}{
		{SourceOFAC, filepath.Join("/snap", "ofac.json")},
		{SourceUN, filepath.Join("/snap", "un.json")},
		{SourceEU, filepath.Join("/snap", "eu.json")},
		{SourceCOAF, filepath.Join("/snap", "coaf.json")},
	}
	for _, tt := range tests {
		t.Run(string(tt.source), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, SnapshotPath("/snap", tt.source))
		})
	}
}

func TestWriteReadSnapshot_RoundTrip(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "nested") // WriteSnapshot creates the directory
	in := Snapshot{Source: SourceOFAC, FetchedAt: snapTime, Entries: snapEntries()}
	require.NoError(t, WriteSnapshot(dir, in))

	out, err := ReadSnapshot(dir, SourceOFAC)
	require.NoError(t, err)
	assert.Equal(t, in.Source, out.Source)
	assert.True(t, in.FetchedAt.Equal(out.FetchedAt), "fetched_at %v != %v", out.FetchedAt, in.FetchedAt)
	assert.Equal(t, in.Entries, out.Entries)

	// Atomic write: nothing but the snapshot is left in the directory.
	names, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, names, 1)
	assert.Equal(t, "ofac.json", names[0].Name())
}

func TestWriteSnapshot_ReplacesPrevious(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, WriteSnapshot(dir, Snapshot{Source: SourceOFAC, FetchedAt: snapTime, Entries: snapEntries()}))
	later := snapTime.Add(time.Hour)
	require.NoError(t, WriteSnapshot(dir, Snapshot{Source: SourceOFAC, FetchedAt: later, Entries: snapEntries()[:1]}))

	out, err := ReadSnapshot(dir, SourceOFAC)
	require.NoError(t, err)
	assert.True(t, later.Equal(out.FetchedAt))
	assert.Len(t, out.Entries, 1)
}

func TestWriteSnapshot_RefusesInvalid(t *testing.T) {
	t.Parallel()
	other := snapEntries()
	other[1].Source = SourceUN
	broken := snapEntries()
	broken[0].Name = " "
	tests := []struct {
		name string
		snap Snapshot
	}{
		{"unknown source", Snapshot{Source: "XX", FetchedAt: snapTime, Entries: snapEntries()}},
		{"zero fetched_at", Snapshot{Source: SourceOFAC, Entries: snapEntries()}},
		{"no entries", Snapshot{Source: SourceOFAC, FetchedAt: snapTime}},
		{"entry of another list", Snapshot{Source: SourceOFAC, FetchedAt: snapTime, Entries: other}},
		{"invalid entry", Snapshot{Source: SourceOFAC, FetchedAt: snapTime, Entries: broken}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			err := WriteSnapshot(dir, tt.snap)
			require.ErrorIs(t, err, ErrInvalidSnapshot)
			names, rerr := os.ReadDir(dir)
			require.NoError(t, rerr)
			assert.Empty(t, names, "a refused snapshot must leave no file behind")
		})
	}
}

func TestReadSnapshot_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string // written as ofac.json; empty means no file
		want    error
	}{
		{"missing file", "", fs.ErrNotExist},
		{"not json", "<html>", ErrInvalidSnapshot},
		{"other list in the file", `{"source":"UN","fetched_at":"2026-11-10T12:00:00Z","entries":[{"Source":"UN","ID":"1","Name":"X","Kind":"ENTITY"}]}`, ErrInvalidSnapshot},
		{"no fetched_at", `{"source":"OFAC","entries":[{"Source":"OFAC","ID":"1","Name":"X","Kind":"ENTITY"}]}`, ErrInvalidSnapshot},
		{"empty entries", `{"source":"OFAC","fetched_at":"2026-11-10T12:00:00Z","entries":[]}`, ErrInvalidSnapshot},
		{"invalid entry", `{"source":"OFAC","fetched_at":"2026-11-10T12:00:00Z","entries":[{"Source":"OFAC","ID":"","Name":"X","Kind":"ENTITY"}]}`, ErrInvalidSnapshot},
		{"trailing data", `{"source":"OFAC","fetched_at":"2026-11-10T12:00:00Z","entries":[{"Source":"OFAC","ID":"1","Name":"X","Kind":"ENTITY"}]} {}`, ErrInvalidSnapshot},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.content != "" {
				require.NoError(t, os.WriteFile(SnapshotPath(dir, SourceOFAC), []byte(tt.content), 0o600))
			}
			_, err := ReadSnapshot(dir, SourceOFAC)
			require.Error(t, err)
			assert.True(t, errors.Is(err, tt.want), "got %v, want %v", err, tt.want)
		})
	}
}

func TestReadSnapshot_LoadsIntoCache(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, WriteSnapshot(dir, Snapshot{Source: SourceOFAC, FetchedAt: snapTime, Entries: snapEntries()}))
	snap, err := ReadSnapshot(dir, SourceOFAC)
	require.NoError(t, err)

	c := NewCache()
	require.NoError(t, c.Load(snap.Entries))
	assert.Equal(t, 2, c.Len())
	hits, err := c.Match(MatchQuery{Name: "Mardok Shipping", MinScore: 0.9})
	require.NoError(t, err)
	require.NotEmpty(t, hits)
	assert.Equal(t, "990002", hits[0].EntryID)
}
