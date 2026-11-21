package un

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenutech/exchangeos/pkg/sanctions"
)

var update = flag.Bool("update", false, "rewrite testdata/golden.json from the parse of testdata/consolidated.xml")

func readFixture(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "consolidated.xml"))
	require.NoError(t, err)
	return b
}

func TestParse_Golden(t *testing.T) {
	got, err := Parse(bytes.NewReader(readFixture(t)))
	require.NoError(t, err)
	golden := filepath.Join("testdata", "golden.json")
	if *update {
		b, err := json.MarshalIndent(got, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(golden, append(b, '\n'), 0o600))
	}
	b, err := os.ReadFile(golden)
	require.NoError(t, err)
	var want []sanctions.Entry
	require.NoError(t, json.Unmarshal(b, &want))
	assert.Equal(t, want, got)
	for _, e := range got {
		require.NoError(t, e.Validate(), e.ID)
	}
}

func TestParse_Malformed(t *testing.T) {
	t.Parallel()
	for name, body := range map[string]string{
		"truncated":       `<CONSOLIDATED_LIST><INDIVIDUALS><INDIVIDUAL><DATAID>1</DATAID><FIRST_NAME>X`,
		"not xml":         `{"CONSOLIDATED_LIST": []}`,
		"another root":    `<html><body>Service unavailable</body></html>`,
		"mismatched tags": `<CONSOLIDATED_LIST><INDIVIDUALS><INDIVIDUAL><DATAID>1</INDIVIDUAL></INDIVIDUALS></CONSOLIDATED_LIST>`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(strings.NewReader(body))
			assert.Error(t, err)
		})
	}
}

func TestParse_TooLarge(t *testing.T) {
	t.Parallel()
	body := `<CONSOLIDATED_LIST><ENTITIES>` +
		strings.Repeat(`<ENTITY><DATAID>1</DATAID><FIRST_NAME>SYNTHETIC</FIRST_NAME></ENTITY>`, 20) +
		`</ENTITIES></CONSOLIDATED_LIST>`
	_, err := parse(strings.NewReader(body), 64)
	require.ErrorIs(t, err, ErrListTooLarge)
	got, err := parse(strings.NewReader(body), int64(len(body)))
	require.NoError(t, err, "a list of exactly the cap is accepted")
	assert.Len(t, got, 20)
	assert.Equal(t, int64(256<<20), MaxListBytes)
}

type memIndex struct {
	mu      sync.Mutex
	entries []sanctions.Entry
}

func (m *memIndex) Load(entries []sanctions.Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = entries
	return nil
}

func (m *memIndex) Match(sanctions.MatchQuery) ([]sanctions.Match, error) { return nil, nil }

func fetchBytes(b []byte) sanctions.Fetch {
	return func(context.Context) (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(b)), nil }
}

func TestNew_RefreshLoadsTheList(t *testing.T) {
	t.Parallel()
	idx := &memIndex{}
	p, err := New(fetchBytes(readFixture(t)), idx)
	require.NoError(t, err)
	assert.Equal(t, sanctions.SourceUN, p.Source())
	require.NoError(t, p.Refresh(context.Background()))
	assert.Len(t, idx.entries, 4)
}

func TestNew_InvalidEntriesAreRefused(t *testing.T) {
	t.Parallel()
	for name, body := range map[string]string{
		"without reference or dataid": `<CONSOLIDATED_LIST><ENTITIES><ENTITY><FIRST_NAME>SYNTHETIC</FIRST_NAME></ENTITY></ENTITIES></CONSOLIDATED_LIST>`,
		"without name":                `<CONSOLIDATED_LIST><INDIVIDUALS><INDIVIDUAL><DATAID>1</DATAID><REFERENCE_NUMBER>QDi.991</REFERENCE_NUMBER></INDIVIDUAL></INDIVIDUALS></CONSOLIDATED_LIST>`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			idx := &memIndex{}
			p, err := New(fetchBytes([]byte(body)), idx)
			require.NoError(t, err)
			assert.ErrorIs(t, p.Refresh(context.Background()), sanctions.ErrInvalidEntry)
			assert.Empty(t, idx.entries)
		})
	}
}

func TestNew_EmptyListIsRefused(t *testing.T) {
	t.Parallel()
	body := `<CONSOLIDATED_LIST dateGenerated="2026-11-10T00:00:00.000Z"><INDIVIDUALS/><ENTITIES/></CONSOLIDATED_LIST>`
	p, err := New(fetchBytes([]byte(body)), &memIndex{})
	require.NoError(t, err)
	assert.ErrorIs(t, p.Refresh(context.Background()), sanctions.ErrEmptyList)
}

func TestNew_RequiresFetchAndIndex(t *testing.T) {
	t.Parallel()
	_, err := New(nil, &memIndex{})
	assert.Error(t, err)
	_, err = New(fetchBytes(nil), nil)
	assert.Error(t, err)
}
