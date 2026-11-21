// Package eu reads the EU consolidated list of persons, groups and entities subject to financial sanctions, as
// published in the Financial Sanctions Files (FSF) CSV.
//
// Format: UTF-8 (a leading BOM is tolerated), semicolon-separated, one header row, quoted fields may contain «;»
// and doubled quotes. One sanctioned entity spans several rows (one per name alias, citizenship, address…), all
// sharing Entity_LogicalId; rows are grouped by it, in order of first appearance.
//
// Columns read, by header name (position does not matter, extra columns are ignored):
//
//	Entity_LogicalId                       required  → Entry.ID
//	NameAlias_WholeName                    required  → first non-empty is Entry.Name, the others Entry.Aliases
//	Entity_SubjectType                     required  → Entry.Kind («person» / «enterprise»)
//	Entity_SubjectType_ClassificationCode  optional  → Entry.Kind when present («P» / «E»); wins over SubjectType
//	Entity_Regulation_Programme            optional  → Entry.Programs
//	Citizenship_CountryIso2Code            optional  → Entry.Countries
//	Address_CountryIso2Code                optional  → Entry.Countries
//
// Country codes are upper-cased and kept only when they are two letters («00», the FSF «unknown», is dropped).
// Duplicates are removed keeping the first occurrence. Rows without id or name are not dropped: they become an
// Entry that ListProvider refuses with sanctions.ErrInvalidEntry, so a damaged download never loads half a list.
package eu

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/revenutech/exchangeos/pkg/sanctions"
)

// MaxListBytes caps how much of a download Parse reads. The real FSF CSV is in the tens of MiB; 256 MiB leaves
// room to grow while refusing a runaway or hostile response before it exhausts memory. With LazyQuotes an
// unclosed quote turns the rest of the input into one field, so the cap is what bounds a damaged download.
const MaxListBytes int64 = 256 << 20

var (
	// ErrMissingColumn is returned when the header lacks a required column (or there is no header).
	ErrMissingColumn = errors.New("eu: required column missing")
	// ErrListTooLarge is returned when the input exceeds MaxListBytes.
	ErrListTooLarge = errors.New("eu: list exceeds the size cap")
)

const (
	colID       = "Entity_LogicalId"
	colName     = "NameAlias_WholeName"
	colSubject  = "Entity_SubjectType"
	colClass    = "Entity_SubjectType_ClassificationCode"
	colProgram  = "Entity_Regulation_Programme"
	colCitizen  = "Citizenship_CountryIso2Code"
	colAddrCtry = "Address_CountryIso2Code"
)

var bom = []byte{0xEF, 0xBB, 0xBF}

// New wires the EU provider: fetch the FSF CSV, Parse it, load it into idx.
func New(fetch sanctions.Fetch, idx sanctions.Index) (*sanctions.ListProvider, error) {
	return sanctions.NewListProvider(sanctions.SourceEU, fetch, Parse, idx)
}

// Parse reads the FSF consolidated CSV into one Entry per Entity_LogicalId, reading at most MaxListBytes.
func Parse(r io.Reader) ([]sanctions.Entry, error) { return parse(r, MaxListBytes) }

func parse(r io.Reader, limit int64) (entries []sanctions.Entry, err error) {
	lim := &capReader{r: r, left: limit}
	defer func() {
		if lim.over {
			entries, err = nil, ErrListTooLarge
		}
	}()
	cr := newCSVReader(lim)
	col, err := readHeader(cr)
	if err != nil {
		return nil, err
	}

	var order []string
	byID := map[string]*sanctions.Entry{}
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("eu: row: %w", err)
		}
		id := col.field(rec, colID)
		e, seen := byID[id]
		if !seen {
			e = &sanctions.Entry{Source: sanctions.SourceEU, ID: id, Kind: kindOf(col.field(rec, colClass), col.field(rec, colSubject))}
			byID[id] = e
			order = append(order, id)
		}
		col.merge(e, rec)
	}
	out := make([]sanctions.Entry, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	return out, nil
}

// newCSVReader skips a leading BOM and reads the FSF dialect: «;»-separated, lazy quotes, rows of any width.
func newCSVReader(r io.Reader) *csv.Reader {
	br := bufio.NewReader(r)
	if b, err := br.Peek(len(bom)); err == nil && bytes.Equal(b, bom) {
		_, _ = br.Discard(len(bom))
	}
	cr := csv.NewReader(br)
	cr.Comma = ';'
	cr.LazyQuotes = true
	cr.FieldsPerRecord = -1
	cr.ReuseRecord = true
	return cr
}

// columns maps a trimmed header name to its column; the last column wins when a name repeats.
type columns map[string]int

// readHeader reads the header row and checks it has the required columns.
func readHeader(cr *csv.Reader) (columns, error) {
	head, err := cr.Read()
	if errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: no header", ErrMissingColumn)
	}
	if err != nil {
		return nil, fmt.Errorf("eu: header: %w", err)
	}
	col := columns{}
	for i, h := range head {
		col[strings.TrimSpace(h)] = i
	}
	for _, req := range []string{colID, colName, colSubject} {
		if _, ok := col[req]; !ok {
			return nil, fmt.Errorf("%w: %s", ErrMissingColumn, req)
		}
	}
	return col, nil
}

// field is the trimmed cell of rec under the column name; empty when the column or the cell is absent.
func (col columns) field(rec []string, name string) string {
	i, ok := col[name]
	if !ok || i >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[i])
}

// merge adds the name alias, programme and countries of one row of e's entity to e.
func (col columns) merge(e *sanctions.Entry, rec []string) {
	if n := col.field(rec, colName); n != "" {
		switch {
		case e.Name == "":
			e.Name = n
		case n != e.Name:
			e.Aliases = appendUnique(e.Aliases, n)
		}
	}
	if p := col.field(rec, colProgram); p != "" {
		e.Programs = appendUnique(e.Programs, p)
	}
	for _, c := range []string{col.field(rec, colCitizen), col.field(rec, colAddrCtry)} {
		if c = strings.ToUpper(c); isISO2(c) {
			e.Countries = appendUnique(e.Countries, c)
		}
	}
}

func kindOf(class, subject string) sanctions.EntryKind {
	switch strings.ToUpper(class) {
	case "P":
		return sanctions.KindIndividual
	case "E":
		return sanctions.KindEntity
	}
	switch strings.ToLower(subject) {
	case "person":
		return sanctions.KindIndividual
	case "enterprise":
		return sanctions.KindEntity
	}
	return sanctions.KindOther
}

func isISO2(c string) bool {
	return len(c) == 2 && c[0] >= 'A' && c[0] <= 'Z' && c[1] >= 'A' && c[1] <= 'Z'
}

func appendUnique(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}

// capReader reads at most left bytes and fails, remembering why, if the input has more.
type capReader struct {
	r    io.Reader
	left int64
	over bool
}

func (c *capReader) Read(p []byte) (int, error) {
	if c.left <= 0 {
		var one [1]byte
		n, err := c.r.Read(one[:])
		if n > 0 {
			c.over = true
			return 0, ErrListTooLarge
		}
		return 0, err
	}
	if int64(len(p)) > c.left {
		p = p[:c.left]
	}
	n, err := c.r.Read(p)
	c.left -= int64(n)
	return n, err
}
