package ofac

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
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

var update = flag.Bool("update", false, "rewrite testdata/golden.json from the parse of testdata/sdn.xml")

func readFixture(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "sdn.xml"))
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

// toSDN writes entries back in the SDN element structure. The whole primary name goes to lastName (the list
// does not let us split it back), and countries go out as the names the list uses.
func toSDN(t *testing.T, entries []sanctions.Entry) []byte {
	t.Helper()
	names := map[string]string{}
	for name, code := range countryCodes {
		names[code] = name
	}
	kinds := map[sanctions.EntryKind]string{
		sanctions.KindIndividual: "Individual", sanctions.KindEntity: "Entity",
		sanctions.KindVessel: "Vessel", sanctions.KindAircraft: "Aircraft",
	}
	doc := sdnList{}
	for _, e := range entries {
		x := sdnEntry{UID: e.ID, LastName: e.Name, SDNType: kinds[e.Kind]}
		x.Programs = append(x.Programs, e.Programs...)
		for _, a := range e.Aliases {
			x.AKAs = append(x.AKAs, aka{LastName: a})
		}
		for _, c := range e.Countries {
			n, ok := names[c]
			require.True(t, ok, "no list name for %s", c)
			x.Addresses = append(x.Addresses, place{Country: n})
		}
		doc.Entries = append(doc.Entries, x)
	}
	b, err := xml.Marshal(doc)
	require.NoError(t, err)
	return b
}

func TestParse_RoundTrip(t *testing.T) {
	first, err := Parse(bytes.NewReader(readFixture(t)))
	require.NoError(t, err)
	second, err := Parse(bytes.NewReader(toSDN(t, first)))
	require.NoError(t, err)
	assert.Equal(t, first, second)
}

func TestParse_Malformed(t *testing.T) {
	t.Parallel()
	for name, body := range map[string]string{
		"truncated":       `<sdnList><sdnEntry><uid>1</uid><lastName>X`,
		"not xml":         `{"sdnList": []}`,
		"another root":    `<html><body>Service unavailable</body></html>`,
		"mismatched tags": `<sdnList><sdnEntry><uid>1</sdnEntry></sdnList>`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(strings.NewReader(body))
			assert.Error(t, err)
		})
	}
}

func TestParse_UnknownTypeIsOther(t *testing.T) {
	t.Parallel()
	got, err := Parse(strings.NewReader(`<sdnList><sdnEntry><uid>1</uid><lastName>SYNTHETIC</lastName><sdnType>Spaceship</sdnType></sdnEntry></sdnList>`))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, sanctions.KindOther, got[0].Kind)
}

func TestParse_TooLarge(t *testing.T) {
	t.Parallel()
	body := `<sdnList>` + strings.Repeat(`<sdnEntry><uid>1</uid><lastName>SYNTHETIC</lastName><sdnType>Entity</sdnType></sdnEntry>`, 20) + `</sdnList>`
	_, err := parse(strings.NewReader(body), 64)
	require.ErrorIs(t, err, ErrListTooLarge)
	got, err := parse(strings.NewReader(body), int64(len(body)))
	require.NoError(t, err, "a list of exactly the cap is accepted")
	assert.Len(t, got, 20)
	assert.Equal(t, int64(256<<20), MaxListBytes)
}

// memIndex is the smallest sanctions.Index: it keeps what was loaded.
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
	assert.Equal(t, sanctions.SourceOFAC, p.Source())
	require.NoError(t, p.Refresh(context.Background()))
	assert.Len(t, idx.entries, 4)
}

func TestNew_InvalidEntriesAreRefused(t *testing.T) {
	t.Parallel()
	for name, body := range map[string]string{
		"without uid":  `<sdnList><sdnEntry><lastName>SYNTHETIC</lastName><sdnType>Entity</sdnType></sdnEntry></sdnList>`,
		"without name": `<sdnList><sdnEntry><uid>1</uid><sdnType>Individual</sdnType></sdnEntry></sdnList>`,
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
	body := `<sdnList><publshInformation><Publish_Date>11/10/2026</Publish_Date><Record_Count>0</Record_Count></publshInformation></sdnList>`
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
