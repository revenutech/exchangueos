// Package un parses the UN Security Council Consolidated List (which includes the 1267/1989/2253 ISIL and
// Al-Qaida regime) in its published XML layout (CONSOLIDATED_LIST/INDIVIDUALS/INDIVIDUAL and ENTITIES/ENTITY)
// into sanctions entries. Fetching is not here: the caller passes a sanctions.Fetch.
package un

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/revenutech/exchangeos/pkg/sanctions"
)

// MaxListBytes caps how much of a download Parse reads. The real consolidated.xml is a few MiB; 256 MiB leaves
// room to grow while refusing a runaway or hostile response before it exhausts memory.
const MaxListBytes int64 = 256 << 20

var (
	// ErrListTooLarge is returned when the input exceeds MaxListBytes.
	ErrListTooLarge = errors.New("un: list exceeds the size cap")
	// ErrNotConsolidatedList is returned when the document root is not CONSOLIDATED_LIST.
	ErrNotConsolidatedList = errors.New("un: document root is not CONSOLIDATED_LIST")
)

// record is one INDIVIDUAL or ENTITY; the two share every field this package reads, under different alias and
// address element names.
type record struct {
	DataID     string   `xml:"DATAID"`
	Reference  string   `xml:"REFERENCE_NUMBER"`
	First      string   `xml:"FIRST_NAME"`
	Second     string   `xml:"SECOND_NAME"`
	Third      string   `xml:"THIRD_NAME"`
	Fourth     string   `xml:"FOURTH_NAME"`
	ListType   string   `xml:"UN_LIST_TYPE"`
	Nationals  []string `xml:"NATIONALITY>VALUE"`
	IndAliases []alias  `xml:"INDIVIDUAL_ALIAS"`
	EntAliases []alias  `xml:"ENTITY_ALIAS"`
	IndAddress []place  `xml:"INDIVIDUAL_ADDRESS"`
	EntAddress []place  `xml:"ENTITY_ADDRESS"`
}

type alias struct {
	Name string `xml:"ALIAS_NAME"`
}

type place struct {
	Country string `xml:"COUNTRY"`
}

// countryCodes maps the country names the UN list uses to ISO 3166-1 alpha-2. The list publishes names, not
// codes; a name missing here is dropped from Entry.Countries (the entry still matches by name). Extend it when a
// name shows up in a real refresh.
var countryCodes = map[string]string{
	"Afghanistan":                           "AF",
	"Central African Republic":              "CF",
	"Democratic People's Republic of Korea": "KP",
	"Democratic Republic of the Congo":      "CD",
	"Haiti":                                 "HT",
	"Indonesia":                             "ID",
	"Iran (Islamic Republic of)":            "IR",
	"Iraq":                                  "IQ",
	"Kuwait":                                "KW",
	"Lebanon":                               "LB",
	"Libya":                                 "LY",
	"Mali":                                  "ML",
	"Pakistan":                              "PK",
	"Philippines":                           "PH",
	"Qatar":                                 "QA",
	"Russian Federation":                    "RU",
	"Saudi Arabia":                          "SA",
	"Somalia":                               "SO",
	"South Sudan":                           "SS",
	"Sudan":                                 "SD",
	"Syrian Arab Republic":                  "SY",
	"Türkiye":                               "TR",
	"United Arab Emirates":                  "AE",
	"Yemen":                                 "YE",
}

// Parse reads one consolidated.xml. It streams the records, so memory follows the record, not the document; it
// reads at most MaxListBytes. The entry id is REFERENCE_NUMBER (QDi.001…), or DATAID when the list leaves it
// empty. Records are returned as published: one without id or name fails later, in ListProvider.Refresh, with
// sanctions.ErrInvalidEntry. Countries come from nationalities and addresses, not the place of birth.
func Parse(r io.Reader) ([]sanctions.Entry, error) { return parse(r, MaxListBytes) }

// New wires the UN provider: fetch → Parse → idx.
func New(fetch sanctions.Fetch, idx sanctions.Index) (*sanctions.ListProvider, error) {
	return sanctions.NewListProvider(sanctions.SourceUN, fetch, Parse, idx)
}

func parse(r io.Reader, limit int64) ([]sanctions.Entry, error) {
	cr := &capReader{r: r, left: limit}
	dec := xml.NewDecoder(cr)
	fail := func(err error) error {
		if cr.over {
			return ErrListTooLarge
		}
		return fmt.Errorf("un: %w", err)
	}
	var out []sanctions.Entry
	rooted := false
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fail(err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if !rooted {
			if se.Name.Local != "CONSOLIDATED_LIST" {
				return nil, fmt.Errorf("%w: <%s>", ErrNotConsolidatedList, se.Name.Local)
			}
			rooted = true
			continue
		}
		var kind sanctions.EntryKind
		switch se.Name.Local {
		case "INDIVIDUAL":
			kind = sanctions.KindIndividual
		case "ENTITY":
			kind = sanctions.KindEntity
		default:
			continue
		}
		var x record
		if err := dec.DecodeElement(&x, &se); err != nil {
			return nil, fail(err)
		}
		out = append(out, x.entry(kind))
	}
	if cr.over {
		return nil, ErrListTooLarge
	}
	if !rooted {
		return nil, ErrNotConsolidatedList
	}
	return out, nil
}

func (x record) entry(kind sanctions.EntryKind) sanctions.Entry {
	id := strings.TrimSpace(x.Reference)
	if id == "" {
		id = strings.TrimSpace(x.DataID)
	}
	e := sanctions.Entry{Source: sanctions.SourceUN, ID: id, Name: joinNames(x.First, x.Second, x.Third, x.Fourth), Kind: kind}
	for _, a := range append(x.IndAliases, x.EntAliases...) {
		if n := strings.TrimSpace(a.Name); n != "" {
			e.Aliases = append(e.Aliases, n)
		}
	}
	if p := strings.TrimSpace(x.ListType); p != "" {
		e.Programs = []string{p}
	}
	seen := map[string]bool{}
	add := func(name string) {
		if c, ok := countryCodes[strings.TrimSpace(name)]; ok && !seen[c] {
			seen[c] = true
			e.Countries = append(e.Countries, c)
		}
	}
	for _, n := range x.Nationals {
		add(n)
	}
	for _, p := range append(x.IndAddress, x.EntAddress...) {
		add(p.Country)
	}
	sort.Strings(e.Countries)
	return e
}

func joinNames(parts ...string) string {
	var kept []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, " ")
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
