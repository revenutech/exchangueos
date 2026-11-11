package coaf

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The minimal SpreadsheetML parts a reader needs, written with archive/zip only (no spreadsheet dependency).
// Cells are shared strings unless prefixed «inline:» (t="inlineStr") or «num:» (a number, no t attribute); an empty
// cell is not written, as Excel does.

const (
	contentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
		`<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>` +
		`<Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>` +
		`</Types>`
	rootRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
		`</Relationships>`
	workbookXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
		`<sheets><sheet name="Lista" sheetId="1" r:id="rId1"/></sheets></workbook>`
	workbookRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>` +
		`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>` +
		`</Relationships>`
)

type part struct{ name, body string }

// fixedTime keeps the generated archive byte-for-byte reproducible.
var fixedTime = time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)

// zipParts writes the parts in order, stored (no compression) so the bytes do not depend on the deflate encoder.
func zipParts(t testing.TB, parts []part) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, p := range parts {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: p.name, Method: zip.Store, Modified: fixedTime})
		require.NoError(t, err)
		_, err = w.Write([]byte(p.body))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func esc(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func colName(i int) string {
	s := ""
	for i++; i > 0; i = (i - 1) / 26 {
		s = string(rune('A'+(i-1)%26)) + s
	}
	return s
}

// sheetParts renders rows into the sheet and shared-strings parts.
func sheetParts(rows [][]string) (sheet, sst string) {
	var strs []string
	index := map[string]int{}
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
		`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for r, row := range rows {
		fmt.Fprintf(&sb, `<row r="%d">`, r+1)
		for c, v := range row {
			ref := fmt.Sprintf("%s%d", colName(c), r+1)
			switch {
			case v == "":
				continue
			case strings.HasPrefix(v, "inline:"):
				fmt.Fprintf(&sb, `<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`, ref, esc(strings.TrimPrefix(v, "inline:")))
			case strings.HasPrefix(v, "num:"):
				fmt.Fprintf(&sb, `<c r="%s"><v>%s</v></c>`, ref, esc(strings.TrimPrefix(v, "num:")))
			default:
				i, ok := index[v]
				if !ok {
					i = len(strs)
					index[v] = i
					strs = append(strs, v)
				}
				fmt.Fprintf(&sb, `<c r="%s" t="s"><v>%d</v></c>`, ref, i)
			}
		}
		sb.WriteString(`</row>`)
	}
	sb.WriteString(`</sheetData></worksheet>`)
	var ss strings.Builder
	fmt.Fprintf(&ss, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`+"\n"+
		`<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="%d" uniqueCount="%d">`, len(strs), len(strs))
	for _, s := range strs {
		fmt.Fprintf(&ss, `<si><t xml:space="preserve">%s</t></si>`, esc(s))
	}
	ss.WriteString(`</sst>`)
	return sb.String(), ss.String()
}

// xlsxParts is a complete minimal workbook for rows.
func xlsxParts(rows [][]string) []part {
	sheet, sst := sheetParts(rows)
	return []part{
		{"[Content_Types].xml", contentTypes},
		{"_rels/.rels", rootRels},
		{"xl/workbook.xml", workbookXML},
		{"xl/_rels/workbook.xml.rels", workbookRels},
		{"xl/worksheets/sheet1.xml", sheet},
		{"xl/sharedStrings.xml", sst},
	}
}

func buildXLSX(t testing.TB, rows [][]string) []byte { return zipParts(t, xlsxParts(rows)) }

// replacePart swaps (or, with an empty body, drops) one part.
func replacePart(parts []part, name, body string) []part {
	out := make([]part, 0, len(parts))
	for _, p := range parts {
		if p.name == name {
			if body != "" {
				out = append(out, part{name, body})
			}
			continue
		}
		out = append(out, p)
	}
	return out
}
