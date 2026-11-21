// Package coaf reads the COAF publication of the UN Security Council (CSNU) sanctions list, distributed as an
// XLSX workbook. The workbook is read with the standard library only (archive/zip + encoding/xml), by decision of
// the owner on 2026-11-10: no spreadsheet dependency.
//
// Layout: the first sheet of xl/workbook.xml (resolved through xl/_rels/workbook.xml.rels); the first non-empty row
// is the header; each following non-empty row is one designation. Columns are found by header name, compared
// without case, accents or surrounding blanks, so their order does not matter and extra columns are ignored:
//
//	Referência (or ID)  required  → Entry.ID
//	Nome                required  → Entry.Name
//	Tipo                required  → Entry.Kind (Pessoa Física / Indivíduo, Pessoa Jurídica / Entidade / Empresa,
//	                                Embarcação / Navio, Aeronave; anything else is OTHER)
//	Outros Nomes        optional  → Entry.Aliases, separated by «;»
//	Nacionalidade       optional  → Entry.Countries, ISO 3166-1 alpha-2 separated by «;» or «,»; other tokens dropped
//	Regime              optional  → Entry.Programs, separated by «;»
//
// Cells: shared strings (t="s", rich-text runs concatenated, phonetic runs ignored), inline strings
// (t="inlineStr"), and raw values (t="str", numbers). Rows without id or name are kept so ListProvider refuses the
// snapshot with sanctions.ErrInvalidEntry instead of loading half a list.
//
// Zip-bomb guards: the archive is read up to MaxInput bytes, may hold at most MaxEntries entries, and the parts
// actually read (workbook, its rels, the sheet, the shared strings) share a budget of MaxUncompressed inflated
// bytes, checked against the declared size before inflating and enforced again while inflating.
package coaf

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/revenutech/exchangeos/pkg/sanctions"
)

const (
	// MaxInput is the largest archive accepted, compressed (the real list is well under 1 MiB).
	MaxInput int64 = 32 << 20
	// MaxUncompressed is the inflated-byte budget shared by every part the parser reads.
	MaxUncompressed int64 = 64 << 20
	// MaxEntries is the most entries the archive may contain (a plain workbook has about a dozen).
	MaxEntries = 128
	// maxColumn is XFD, the last column SpreadsheetML allows.
	maxColumn = 16384
)

var (
	// ErrNotXLSX is returned when the input is not a readable XLSX workbook (bad zip, no workbook, malformed XML
	// or cells).
	ErrNotXLSX = errors.New("coaf: not a readable xlsx workbook")
	// ErrMissingSheet is returned when the workbook does not lead to a first sheet that exists in the archive.
	ErrMissingSheet = errors.New("coaf: first sheet missing")
	// ErrTooLarge is returned when a zip-bomb guard trips (input size, entry count or inflated bytes).
	ErrTooLarge = errors.New("coaf: workbook exceeds size limits")
	// ErrMissingColumn is returned when the header lacks a required column (or there is no header).
	ErrMissingColumn = errors.New("coaf: required column missing")
)

type limits struct {
	input, uncompressed int64
	entries             int
}

var defaultLimits = limits{input: MaxInput, uncompressed: MaxUncompressed, entries: MaxEntries}

// New wires the COAF provider: fetch the XLSX, Parse it, load it into idx.
func New(fetch sanctions.Fetch, idx sanctions.Index) (*sanctions.ListProvider, error) {
	return sanctions.NewListProvider(sanctions.SourceCOAF, fetch, Parse, idx)
}

// Parse reads the COAF XLSX into one Entry per data row.
func Parse(r io.Reader) ([]sanctions.Entry, error) { return parse(r, defaultLimits) }

func parse(r io.Reader, lim limits) ([]sanctions.Entry, error) {
	rows, err := readFirstSheet(r, lim)
	if err != nil {
		return nil, err
	}
	return entries(rows)
}

// --- workbook ---

type zipBudget struct {
	files  map[string]*zip.File
	remain int64
}

// read returns the inflated part, false when it is not in the archive.
func (z *zipBudget) read(name string) ([]byte, bool, error) {
	f, ok := z.files[name]
	if !ok {
		return nil, false, nil
	}
	if f.UncompressedSize64 > uint64(z.remain) { //nolint:gosec // remain is never negative
		return nil, true, fmt.Errorf("%w: %s declares %d bytes inflated", ErrTooLarge, name, f.UncompressedSize64)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, true, fmt.Errorf("%w: open %s: %w", ErrNotXLSX, name, err)
	}
	defer func() { _ = rc.Close() }()
	b, err := io.ReadAll(io.LimitReader(rc, z.remain+1))
	if err != nil && int64(len(b)) <= z.remain {
		return nil, true, fmt.Errorf("%w: inflate %s: %w", ErrNotXLSX, name, err)
	}
	if int64(len(b)) > z.remain {
		return nil, true, fmt.Errorf("%w: %s inflates past the budget", ErrTooLarge, name)
	}
	z.remain -= int64(len(b))
	return b, true, nil
}

type xWorkbook struct {
	Sheets []struct {
		Attrs []xml.Attr `xml:",any,attr"`
	} `xml:"sheets>sheet"`
}

type xRels struct {
	Rels []struct {
		ID     string `xml:"Id,attr"`
		Target string `xml:"Target,attr"`
	} `xml:"Relationship"`
}

type xSI struct {
	T string `xml:"t"`
	R []struct {
		T string `xml:"t"`
	} `xml:"r"`
}

func (s xSI) text() string {
	var b strings.Builder
	_, _ = b.WriteString(s.T) // a strings.Builder write never fails
	for _, r := range s.R {
		_, _ = b.WriteString(r.T)
	}
	return b.String()
}

type xSST struct {
	SI []xSI `xml:"si"`
}

type xSheet struct {
	Rows []struct {
		C []struct {
			R  string `xml:"r,attr"`
			T  string `xml:"t,attr"`
			V  string `xml:"v"`
			IS xSI    `xml:"is"`
		} `xml:"c"`
	} `xml:"sheetData>row"`
}

// readFirstSheet reads the cells of the workbook's first sheet, inflating at most lim.uncompressed bytes in all.
func readFirstSheet(r io.Reader, lim limits) ([][]string, error) {
	z, err := openArchive(r, lim)
	if err != nil {
		return nil, err
	}
	target, err := firstSheetTarget(z)
	if err != nil {
		return nil, err
	}
	sheetXML, ok, err := z.read(path.Clean(target))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: %s not in the archive", ErrMissingSheet, target)
	}
	shared, err := sharedStrings(z)
	if err != nil {
		return nil, err
	}
	var sh xSheet
	if err := xml.Unmarshal(sheetXML, &sh); err != nil {
		return nil, fmt.Errorf("%w: sheet: %w", ErrNotXLSX, err)
	}
	return sheetRows(sh, shared)
}

// openArchive reads at most lim.input bytes of r as a zip archive of at most lim.entries entries, and indexes them
// under a budget of lim.uncompressed inflated bytes.
func openArchive(r io.Reader, lim limits) (*zipBudget, error) {
	data, err := io.ReadAll(io.LimitReader(r, lim.input+1))
	if err != nil {
		return nil, fmt.Errorf("coaf: read: %w", err)
	}
	if int64(len(data)) > lim.input {
		return nil, fmt.Errorf("%w: archive larger than %d bytes", ErrTooLarge, lim.input)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotXLSX, err)
	}
	if len(zr.File) > lim.entries {
		return nil, fmt.Errorf("%w: %d entries, at most %d", ErrTooLarge, len(zr.File), lim.entries)
	}
	z := &zipBudget{files: make(map[string]*zip.File, len(zr.File)), remain: lim.uncompressed}
	for _, f := range zr.File {
		z.files[f.Name] = f
	}
	return z, nil
}

// firstSheetTarget resolves the archive path of the workbook's first sheet through the workbook relationships. The
// result is not cleaned yet.
func firstSheetTarget(z *zipBudget) (string, error) {
	wbXML, ok, err := z.read("xl/workbook.xml")
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("%w: no xl/workbook.xml", ErrNotXLSX)
	}
	var wb xWorkbook
	if err := xml.Unmarshal(wbXML, &wb); err != nil {
		return "", fmt.Errorf("%w: workbook: %w", ErrNotXLSX, err)
	}
	if len(wb.Sheets) == 0 {
		return "", fmt.Errorf("%w: workbook lists no sheet", ErrMissingSheet)
	}
	rid := ""
	for _, a := range wb.Sheets[0].Attrs {
		if a.Name.Local == "id" {
			rid = a.Value
		}
	}
	relsXML, ok, err := z.read("xl/_rels/workbook.xml.rels")
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("%w: no workbook relationships", ErrMissingSheet)
	}
	var rels xRels
	if err := xml.Unmarshal(relsXML, &rels); err != nil {
		return "", fmt.Errorf("%w: workbook relationships: %w", ErrNotXLSX, err)
	}
	target := ""
	for _, rel := range rels.Rels {
		if rid != "" && rel.ID == rid {
			target = rel.Target
		}
	}
	if target == "" {
		return "", fmt.Errorf("%w: relationship %q not found", ErrMissingSheet, rid)
	}
	if strings.HasPrefix(target, "/") {
		return strings.TrimPrefix(target, "/"), nil
	}
	return path.Join("xl", target), nil
}

// sharedStrings reads the shared string table; nil when the archive has none.
func sharedStrings(z *zipBudget) ([]string, error) {
	sstXML, ok, err := z.read("xl/sharedStrings.xml")
	if err != nil || !ok {
		return nil, err
	}
	var sst xSST
	if err := xml.Unmarshal(sstXML, &sst); err != nil {
		return nil, fmt.Errorf("%w: shared strings: %w", ErrNotXLSX, err)
	}
	shared := make([]string, len(sst.SI))
	for i, si := range sst.SI {
		shared[i] = si.text()
	}
	return shared, nil
}

// sheetRows lays the cells of sh out by column, resolving shared and inline strings.
func sheetRows(sh xSheet, shared []string) ([][]string, error) {
	rows := make([][]string, 0, len(sh.Rows))
	for _, row := range sh.Rows {
		var cells []string
		next := 0
		for _, c := range row.C {
			col := next
			if c.R != "" {
				var err error
				if col, err = columnIndex(c.R); err != nil {
					return nil, err
				}
			}
			next = col + 1
			v, err := cellValue(c.T, c.V, c.R, c.IS, shared)
			if err != nil {
				return nil, err
			}
			for len(cells) <= col {
				cells = append(cells, "")
			}
			cells[col] = v
		}
		rows = append(rows, cells)
	}
	return rows, nil
}

// cellValue is the text of a cell of type t, value v and inline string is, at reference ref.
func cellValue(t, v, ref string, is xSI, shared []string) (string, error) {
	switch t {
	case "s":
		i, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil || i < 0 || i >= len(shared) {
			return "", fmt.Errorf("%w: cell %s: shared string %q out of range", ErrNotXLSX, ref, v)
		}
		return shared[i], nil
	case "inlineStr":
		return is.text(), nil
	default:
		return v, nil
	}
}

// columnIndex turns the letters of a cell reference («AB12») into a zero-based column.
func columnIndex(ref string) (int, error) {
	n, i := 0, 0
	for ; i < len(ref) && ref[i] >= 'A' && ref[i] <= 'Z'; i++ {
		n = n*26 + int(ref[i]-'A'+1)
		if n > maxColumn {
			return 0, fmt.Errorf("%w: cell %q beyond column XFD", ErrNotXLSX, ref)
		}
	}
	if i == 0 || i == len(ref) {
		return 0, fmt.Errorf("%w: bad cell reference %q", ErrNotXLSX, ref)
	}
	if _, err := strconv.Atoi(ref[i:]); err != nil {
		return 0, fmt.Errorf("%w: bad cell reference %q", ErrNotXLSX, ref)
	}
	return n - 1, nil
}

// --- rows → entries ---

var unaccent = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c",
)

func norm(s string) string {
	return strings.Join(strings.Fields(unaccent.Replace(strings.ToLower(s))), " ")
}

const (
	colID      = "referencia"
	colName    = "nome"
	colKind    = "tipo"
	colAliases = "outros nomes"
	colCountry = "nacionalidade"
	colProgram = "regime"
)

var headerSynonyms = map[string]string{"id": colID, "referencia/id": colID}

func blank(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func entries(rows [][]string) ([]sanctions.Entry, error) {
	h := 0
	for h < len(rows) && blank(rows[h]) {
		h++
	}
	if h == len(rows) {
		return nil, fmt.Errorf("%w: no header row", ErrMissingColumn)
	}
	col, err := headerColumns(rows[h])
	if err != nil {
		return nil, err
	}
	var out []sanctions.Entry
	for _, row := range rows[h+1:] {
		if blank(row) {
			continue
		}
		out = append(out, sanctions.Entry{
			Source:    sanctions.SourceCOAF,
			ID:        col.field(row, colID),
			Name:      col.field(row, colName),
			Kind:      kindOf(col.field(row, colKind)),
			Aliases:   split(col.field(row, colAliases), ";"),
			Programs:  split(col.field(row, colProgram), ";"),
			Countries: countries(col.field(row, colCountry)),
		})
	}
	return out, nil
}

// columns maps a normalised header name to its column; the first column wins when a name repeats.
type columns map[string]int

// headerColumns reads the header row and checks it has the required columns.
func headerColumns(header []string) (columns, error) {
	col := columns{}
	for i, name := range header {
		k := norm(name)
		if s, ok := headerSynonyms[k]; ok {
			k = s
		}
		if _, dup := col[k]; !dup {
			col[k] = i
		}
	}
	for _, req := range []string{colID, colName, colKind} {
		if _, ok := col[req]; !ok {
			return nil, fmt.Errorf("%w: %s", ErrMissingColumn, req)
		}
	}
	return col, nil
}

// field is the trimmed cell of row under the column name; empty when the column or the cell is absent.
func (col columns) field(row []string, name string) string {
	i, ok := col[name]
	if !ok || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

// countries keeps the two-letter codes of s (cut at «;» or «,»), upper-cased, without duplicates; nil when none.
func countries(s string) []string {
	var out []string
	for _, c := range split(s, ";,") {
		if c = strings.ToUpper(c); len(c) == 2 && c[0] >= 'A' && c[0] <= 'Z' && c[1] >= 'A' && c[1] <= 'Z' {
			out = appendUnique(out, c)
		}
	}
	return out
}

func kindOf(tipo string) sanctions.EntryKind {
	switch norm(tipo) {
	case "pessoa fisica", "individuo", "pessoa":
		return sanctions.KindIndividual
	case "pessoa juridica", "entidade", "empresa":
		return sanctions.KindEntity
	case "embarcacao", "navio":
		return sanctions.KindVessel
	case "aeronave":
		return sanctions.KindAircraft
	}
	return sanctions.KindOther
}

// split cuts s at any of seps, trims each part and drops empty ones and duplicates; nil when nothing is left.
func split(s, seps string) []string {
	var out []string
	for _, p := range strings.FieldsFunc(s, func(r rune) bool { return strings.ContainsRune(seps, r) }) {
		if p = strings.TrimSpace(p); p != "" {
			out = appendUnique(out, p)
		}
	}
	return out
}

func appendUnique(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}
