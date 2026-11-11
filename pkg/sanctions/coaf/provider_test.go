package coaf

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
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

var (
	update        = flag.Bool("update", false, "rewrite testdata/golden.json from the parser output")
	updateFixture = flag.Bool("update-fixture", false, "regenerate testdata/coaf-sintetico.xlsx from fixtureRows")
)

const fixture = "testdata/coaf-sintetico.xlsx"

// fixtureRows is the synthetic list behind testdata/coaf-sintetico.xlsx: invented names, ISO 3166 user-assigned
// country codes (XA, XB), the layout documented in provider.go. Row 4 is empty on purpose, «inline:» and «num:»
// cover the two non-shared cell types.
var fixtureRows = [][]string{
	{"Referência", "Nome", "Tipo", "Outros Nomes", "Nacionalidade", "Regime"},
	{"COAF-0001", "João Fictício da Silva", "Pessoa Física", "João F. Silva; Joao Ficticio", "XA", "CSNU 1267"},
	{"COAF-0002", "Empresa Sintética Ltda", "Pessoa Jurídica", "ESL; Sintética & Filhos", "XA; xb", "CSNU 1988"},
	{},
	{"COAF-0003", "inline:Navio Inventado", "Embarcação", "", "", "CSNU 2231"},
	{"num:1004", "Aeronave Exemplo", "Aeronave", "", "00", "CSNU 2231; CSNU 1267"},
	{"COAF-0005", "Organização Imaginária", "Entidade", "", "XB", "CSNU 1988"},
}

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

func TestFixtureMatchesBuilder(t *testing.T) {
	want := buildXLSX(t, fixtureRows)
	if *updateFixture {
		require.NoError(t, os.WriteFile(fixture, want, 0o600))
	}
	got, err := os.ReadFile(fixture)
	require.NoError(t, err)
	assert.True(t, bytes.Equal(want, got), "testdata/coaf-sintetico.xlsx is stale: rerun with -update-fixture")
}

func TestParse_Golden(t *testing.T) {
	b, err := os.ReadFile(fixture)
	require.NoError(t, err)
	got, err := Parse(bytes.NewReader(b))
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

func TestParse_HeaderByNameNotPosition(t *testing.T) {
	t.Parallel()
	// Columns reversed, header spelled without accents and in other case, plus an unknown column.
	rows := [][]string{
		{"Observação", "REGIME", "nacionalidade", "outros nomes", "TIPO", "  nome ", "Referencia"},
		{"ignored", "CSNU 1267", "XA", "Alias Sintético", "Pessoa Física", "Pessoa Sintética", "R-1"},
	}
	got, err := Parse(bytes.NewReader(buildXLSX(t, rows)))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, sanctions.Entry{
		Source: sanctions.SourceCOAF, ID: "R-1", Name: "Pessoa Sintética", Aliases: []string{"Alias Sintético"},
		Kind: sanctions.KindIndividual, Countries: []string{"XA"}, Programs: []string{"CSNU 1267"},
	}, got[0])
}

func TestParse_OptionalColumnsMayBeAbsent(t *testing.T) {
	t.Parallel()
	got, err := Parse(bytes.NewReader(buildXLSX(t, [][]string{{"ID", "Nome", "Tipo"}, {"9", "Sintético", "Desconhecido"}})))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, sanctions.KindOther, got[0].Kind)
	assert.Nil(t, got[0].Aliases)
}

func TestParse_MissingRequiredColumn(t *testing.T) {
	t.Parallel()
	for i, col := range []string{"referencia", "nome", "tipo"} {
		t.Run(col, func(t *testing.T) {
			t.Parallel()
			head := []string{"Referência", "Nome", "Tipo"}
			head[i] = "Outra"
			_, err := Parse(bytes.NewReader(buildXLSX(t, [][]string{head, {"1", "Sintético", "Entidade"}})))
			assert.ErrorIs(t, err, ErrMissingColumn)
			assert.Contains(t, err.Error(), col)
		})
	}
	_, err := Parse(bytes.NewReader(buildXLSX(t, nil)))
	assert.ErrorIs(t, err, ErrMissingColumn, "a sheet without rows has no header")
}

func TestParse_BadZip(t *testing.T) {
	t.Parallel()
	_, err := Parse(strings.NewReader("Referência;Nome;Tipo\n1;a;b\n"))
	assert.ErrorIs(t, err, ErrNotXLSX)
	_, err = Parse(bytes.NewReader(zipParts(t, []part{{"readme.txt", "not a workbook"}})))
	assert.ErrorIs(t, err, ErrNotXLSX, "a zip without xl/workbook.xml")
}

func TestParse_MissingSheet(t *testing.T) {
	t.Parallel()
	base := xlsxParts(fixtureRows)
	cases := map[string][]part{
		"sheet part absent": replacePart(base, "xl/worksheets/sheet1.xml", ""),
		"no sheets":         replacePart(base, "xl/workbook.xml", `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheets/></workbook>`),
		"no workbook rels":  replacePart(base, "xl/_rels/workbook.xml.rels", ""),
		"unknown r:id":      replacePart(base, "xl/workbook.xml", strings.Replace(workbookXML, `r:id="rId1"`, `r:id="rId9"`, 1)),
	}
	for name, parts := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(bytes.NewReader(zipParts(t, parts)))
			assert.ErrorIs(t, err, ErrMissingSheet)
		})
	}
}

func TestParse_AbsoluteSheetTargetAndRichText(t *testing.T) {
	t.Parallel()
	parts := xlsxParts([][]string{{"Referência", "Nome", "Tipo"}, {"1", "placeholder", "Entidade"}})
	parts = replacePart(parts, "xl/_rels/workbook.xml.rels", strings.Replace(workbookRels, `Target="worksheets/sheet1.xml"`, `Target="/xl/worksheets/sheet1.xml"`, 1))
	// Shared string 4 («placeholder») as rich text in two runs, with a phonetic run that must be ignored.
	sst := `<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
		`<si><t>Referência</t></si><si><t>Nome</t></si><si><t>Tipo</t></si><si><t>1</t></si>` +
		`<si><r><t>Rich </t></r><r><t>Synthetic</t></r><rPh><t>IGNORED</t></rPh></si><si><t>Entidade</t></si></sst>`
	parts = replacePart(parts, "xl/sharedStrings.xml", sst)
	got, err := Parse(bytes.NewReader(zipParts(t, parts)))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Rich Synthetic", got[0].Name)
}

func TestParse_MalformedCells(t *testing.T) {
	t.Parallel()
	base := xlsxParts([][]string{{"Referência", "Nome", "Tipo"}, {"1", "Sintético", "Entidade"}})
	sheet := func(cells string) string {
		return `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>` +
			`<row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c><c r="C1" t="s"><v>2</v></c></row>` +
			`<row r="2">` + cells + `</row></sheetData></worksheet>`
	}
	for name, cells := range map[string]string{
		"shared index out of range": `<c r="A2" t="s"><v>99</v></c>`,
		"shared index not a number": `<c r="A2" t="s"><v>x</v></c>`,
		"column beyond XFD":         `<c r="XFE2" t="inlineStr"><is><t>x</t></is></c>`,
		"bad cell reference":        `<c r="2A" t="inlineStr"><is><t>x</t></is></c>`,
		"not xml":                   `<c r="A2"`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(bytes.NewReader(zipParts(t, replacePart(base, "xl/worksheets/sheet1.xml", sheet(cells)))))
			assert.ErrorIs(t, err, ErrNotXLSX)
		})
	}
}

func TestParse_CellsWithoutReferenceAreSequential(t *testing.T) {
	t.Parallel()
	base := xlsxParts([][]string{{"Referência", "Nome", "Tipo"}})
	sheet := `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>` +
		`<row><c t="s"><v>0</v></c><c t="s"><v>1</v></c><c t="s"><v>2</v></c></row>` +
		`<row><c t="inlineStr"><is><t>7</t></is></c><c t="str"><v>Sintético</v></c><c t="inlineStr"><is><t>Entidade</t></is></c></row>` +
		`</sheetData></worksheet>`
	got, err := Parse(bytes.NewReader(zipParts(t, replacePart(base, "xl/worksheets/sheet1.xml", sheet))))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "7", got[0].ID)
	assert.Equal(t, "Sintético", got[0].Name)
	assert.Equal(t, sanctions.KindEntity, got[0].Kind)
}

// --- zip bomb guards ---

func TestParse_TooManyEntries(t *testing.T) {
	t.Parallel()
	parts := xlsxParts(fixtureRows)
	for i := 0; len(parts) <= MaxEntries; i++ {
		parts = append(parts, part{fmt.Sprintf("xl/media/pad%03d.bin", i), "x"})
	}
	_, err := Parse(bytes.NewReader(zipParts(t, parts)))
	assert.ErrorIs(t, err, ErrTooLarge)
}

// A real bomb against the default cap: a shared-strings part that inflates past MaxUncompressed while the
// archive stays well under a megabyte. It is refused from the declared size, before anything is inflated.
func TestParse_ZipBombDeclaredSize(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, p := range xlsxParts(fixtureRows) {
		if p.name == "xl/sharedStrings.xml" {
			continue
		}
		w, err := zw.Create(p.name)
		require.NoError(t, err)
		_, err = io.WriteString(w, p.body)
		require.NoError(t, err)
	}
	w, err := zw.Create("xl/sharedStrings.xml")
	require.NoError(t, err)
	chunk := bytes.Repeat([]byte(" "), 1<<20)
	for written := int64(0); written <= MaxUncompressed; written += int64(len(chunk)) {
		_, err = w.Write(chunk)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	require.Less(t, buf.Len(), 1<<20, "the bomb must be small on the wire")
	_, err = Parse(bytes.NewReader(buf.Bytes()))
	assert.ErrorIs(t, err, ErrTooLarge)
}

// A part whose declared size understates what it inflates to is cut at the budget while reading.
func TestParse_UncompressedBudgetWhileReading(t *testing.T) {
	t.Parallel()
	pad := strings.Repeat("<!-- synthetic padding -->", 200) // ~5 KiB
	parts := replacePart(xlsxParts(fixtureRows), "xl/sharedStrings.xml", "")
	_, sst := sheetParts(fixtureRows)
	parts = append(parts, part{"xl/sharedStrings.xml", strings.Replace(sst, "<sst ", pad+"<sst ", 1)})
	b := zipParts(t, parts)
	_, err := parse(bytes.NewReader(b), limits{input: MaxInput, uncompressed: 4 << 10, entries: MaxEntries})
	assert.ErrorIs(t, err, ErrTooLarge)
	_, err = parse(bytes.NewReader(b), limits{input: MaxInput, uncompressed: 64 << 10, entries: MaxEntries})
	assert.NoError(t, err, "the same archive passes with room in the budget")
}

func TestParse_InputCap(t *testing.T) {
	t.Parallel()
	b := buildXLSX(t, fixtureRows)
	_, err := parse(bytes.NewReader(b), limits{input: int64(len(b) - 1), uncompressed: MaxUncompressed, entries: MaxEntries})
	assert.ErrorIs(t, err, ErrTooLarge)
}

// --- through ListProvider ---

func TestProvider_RefreshLoadsFixture(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(fixture)
	require.NoError(t, err)
	idx := &memIndex{}
	p, err := New(fetchBytes(b), idx)
	require.NoError(t, err)
	assert.Equal(t, sanctions.SourceCOAF, p.Source())
	require.NoError(t, p.Refresh(context.Background()))
	assert.Len(t, idx.entries, 5)
}

func TestProvider_EntityWithoutIDOrNameIsInvalid(t *testing.T) {
	t.Parallel()
	for name, row := range map[string][]string{
		"without id":   {"", "Pessoa Sintética", "Pessoa Física"},
		"without name": {"R-2", "", "Pessoa Física"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rows := [][]string{{"Referência", "Nome", "Tipo"}, {"R-1", "Válido Sintético", "Entidade"}, row}
			idx := &memIndex{}
			p, err := New(fetchBytes(buildXLSX(t, rows)), idx)
			require.NoError(t, err)
			assert.ErrorIs(t, p.Refresh(context.Background()), sanctions.ErrInvalidEntry)
			assert.Empty(t, idx.entries)
		})
	}
}

func TestProvider_EmptyList(t *testing.T) {
	t.Parallel()
	p, err := New(fetchBytes(buildXLSX(t, [][]string{{"Referência", "Nome", "Tipo"}, {}, {}})), &memIndex{})
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
