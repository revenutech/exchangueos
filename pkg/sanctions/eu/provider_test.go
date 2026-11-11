package eu

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

var update = flag.Bool("update", false, "rewrite testdata/golden.json from the parser output")

const fixture = "testdata/eu-fsf-sintetico.csv"

// header is the minimal official header: the three required columns plus the optional ones the parser reads.
const header = "Entity_LogicalId;Entity_SubjectType;Entity_SubjectType_ClassificationCode;Entity_Regulation_Programme;" +
	"NameAlias_WholeName;Citizenship_CountryIso2Code;Address_CountryIso2Code\n"

type memIndex struct {
	mu      sync.Mutex
	entries []sanctions.Entry
}

func (m *memIndex) Load(e []sanctions.Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append([]sanctions.Entry(nil), e...)
	return nil
}

func (m *memIndex) Match(sanctions.MatchQuery) ([]sanctions.Match, error) { return nil, nil }

func fetchBytes(b []byte) sanctions.Fetch {
	return func(context.Context) (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(b)), nil }
}

func TestParse_Golden(t *testing.T) {
	f, err := os.Open(fixture)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	got, err := Parse(f)
	require.NoError(t, err)
	out, err := json.MarshalIndent(got, "", "  ")
	require.NoError(t, err)
	golden := filepath.Join("testdata", "golden.json")
	if *update {
		require.NoError(t, os.WriteFile(golden, append(out, '\n'), 0o600))
	}
	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(out))
}

func TestParse_WithoutBOMAndWithShortRows(t *testing.T) {
	t.Parallel()
	// The last row stops before the country columns: missing trailing fields read as empty.
	body := header + "1;person;P;SYN;Synthetic Person;xa;\n2;enterprise;E;SYN;Synthetic Org\n"
	got, err := Parse(strings.NewReader(body))
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, sanctions.KindIndividual, got[0].Kind)
	assert.Equal(t, []string{"XA"}, got[0].Countries)
	assert.Equal(t, sanctions.KindEntity, got[1].Kind)
	assert.Nil(t, got[1].Countries)
}

func TestParse_KindFallsBackToSubjectTypeThenOther(t *testing.T) {
	t.Parallel()
	body := header + "1;person;;SYN;A Synthetic;;\n2;ENTERPRISE;;SYN;B Synthetic;;\n3;unknown-type;;SYN;C Synthetic;;\n"
	got, err := Parse(strings.NewReader(body))
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, sanctions.KindIndividual, got[0].Kind)
	assert.Equal(t, sanctions.KindEntity, got[1].Kind)
	assert.Equal(t, sanctions.KindOther, got[2].Kind)
}

func TestParse_MissingRequiredColumn(t *testing.T) {
	t.Parallel()
	for _, col := range []string{"Entity_LogicalId", "NameAlias_WholeName", "Entity_SubjectType"} {
		t.Run(col, func(t *testing.T) {
			t.Parallel()
			h := strings.Replace(header, col+";", "Renamed_"+col+";", 1)
			_, err := Parse(strings.NewReader(h + "1;person;P;SYN;Synthetic;;\n"))
			assert.ErrorIs(t, err, ErrMissingColumn)
			assert.Contains(t, err.Error(), col)
		})
	}
	_, err := Parse(strings.NewReader(""))
	assert.ErrorIs(t, err, ErrMissingColumn, "an empty file has no header")
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("connection reset") }

func TestParse_ReadErrorIsReturned(t *testing.T) {
	t.Parallel()
	_, err := Parse(io.MultiReader(strings.NewReader(header+"1;person;P;SYN;Synthetic;"), errReader{}))
	assert.Error(t, err)
}

func TestProvider_RefreshLoadsFixture(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(fixture)
	require.NoError(t, err)
	idx := &memIndex{}
	p, err := New(fetchBytes(b), idx)
	require.NoError(t, err)
	assert.Equal(t, sanctions.SourceEU, p.Source())
	require.NoError(t, p.Refresh(context.Background()))
	assert.Len(t, idx.entries, 3)
}

func TestProvider_EntityWithoutIDOrNameIsInvalid(t *testing.T) {
	t.Parallel()
	for name, row := range map[string]string{
		"without id":   ";person;P;SYN;Synthetic Person;;\n",
		"without name": "7;person;P;SYN;;;\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			idx := &memIndex{}
			p, err := New(fetchBytes([]byte(header+"1;person;P;SYN;Valid Synthetic;;\n"+row)), idx)
			require.NoError(t, err)
			assert.ErrorIs(t, p.Refresh(context.Background()), sanctions.ErrInvalidEntry)
			assert.Empty(t, idx.entries)
		})
	}
}

func TestProvider_EmptyList(t *testing.T) {
	t.Parallel()
	p, err := New(fetchBytes([]byte("\xef\xbb\xbf"+header)), &memIndex{})
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
