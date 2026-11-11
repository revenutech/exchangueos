// Package ofac parses the OFAC Specially Designated Nationals list in its published XML layout (sdn.xml:
// sdnList/sdnEntry) into sanctions entries. Fetching is not here: the caller passes a sanctions.Fetch.
package ofac

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/revenutech/exchangeos/pkg/sanctions"
)

// MaxListBytes caps how much of a download Parse reads. The real sdn.xml is in the tens of MiB; 256 MiB leaves
// room to grow while refusing a runaway or hostile response before it exhausts memory.
const MaxListBytes int64 = 256 << 20

var (
	// ErrListTooLarge is returned when the input exceeds MaxListBytes.
	ErrListTooLarge = errors.New("ofac: list exceeds the size cap")
	// ErrNotSDNList is returned when the document root is not sdnList (an error page, another list).
	ErrNotSDNList = errors.New("ofac: document root is not sdnList")
)

// The SDN element structure. Tags carry no namespace, so they match the published namespace and none.
type sdnList struct {
	XMLName xml.Name   `xml:"sdnList"`
	Entries []sdnEntry `xml:"sdnEntry"`
}

type sdnEntry struct {
	UID           string   `xml:"uid"`
	FirstName     string   `xml:"firstName,omitempty"`
	LastName      string   `xml:"lastName"`
	SDNType       string   `xml:"sdnType"`
	Programs      []string `xml:"programList>program"`
	AKAs          []aka    `xml:"akaList>aka"`
	Addresses     []place  `xml:"addressList>address"`
	Nationalities []place  `xml:"nationalityList>nationality"`
	Citizenships  []place  `xml:"citizenshipList>citizenship"`
}

type aka struct {
	FirstName string `xml:"firstName,omitempty"`
	LastName  string `xml:"lastName"`
}

type place struct {
	Country string `xml:"country"`
}

// countryCodes maps the country names the SDN list uses to ISO 3166-1 alpha-2. The list publishes names, not
// codes; a name missing here is dropped from Entry.Countries (the entry still matches by name). Extend it when a
// name shows up in a real refresh.
var countryCodes = map[string]string{
	"Afghanistan":                       "AF",
	"Argentina":                         "AR",
	"Belarus":                           "BY",
	"Brazil":                            "BR",
	"Burma":                             "MM",
	"China":                             "CN",
	"Colombia":                          "CO",
	"Congo, Democratic Republic of the": "CD",
	"Cuba":                              "CU",
	"Hong Kong":                         "HK",
	"Iran":                              "IR",
	"Iraq":                              "IQ",
	"Korea, North":                      "KP",
	"Lebanon":                           "LB",
	"Mexico":                            "MX",
	"Nicaragua":                         "NI",
	"North Korea":                       "KP",
	"Panama":                            "PA",
	"Paraguay":                          "PY",
	"Region: Crimea":                    "UA",
	"Russia":                            "RU",
	"Syria":                             "SY",
	"Turkey":                            "TR",
	"Ukraine":                           "UA",
	"United Arab Emirates":              "AE",
	"United Kingdom":                    "GB",
	"United States":                     "US",
	"Venezuela":                         "VE",
	"Yemen":                             "YE",
}

var kinds = map[string]sanctions.EntryKind{
	"individual": sanctions.KindIndividual,
	"entity":     sanctions.KindEntity,
	"vessel":     sanctions.KindVessel,
	"aircraft":   sanctions.KindAircraft,
}

// Parse reads one sdn.xml. It streams the entries, so memory follows the entry, not the document; it reads at
// most MaxListBytes. Entries are returned as published: one without uid or name fails later, in
// ListProvider.Refresh, with sanctions.ErrInvalidEntry.
func Parse(r io.Reader) ([]sanctions.Entry, error) { return parse(r, MaxListBytes) }

// New wires the OFAC provider: fetch → Parse → idx.
func New(fetch sanctions.Fetch, idx sanctions.Index) (*sanctions.ListProvider, error) {
	return sanctions.NewListProvider(sanctions.SourceOFAC, fetch, Parse, idx)
}

func parse(r io.Reader, limit int64) ([]sanctions.Entry, error) {
	cr := &capReader{r: r, left: limit}
	dec := xml.NewDecoder(cr)
	var out []sanctions.Entry
	rooted := false
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if cr.over {
				return nil, ErrListTooLarge
			}
			return nil, fmt.Errorf("ofac: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if !rooted {
			if se.Name.Local != "sdnList" {
				return nil, fmt.Errorf("%w: <%s>", ErrNotSDNList, se.Name.Local)
			}
			rooted = true
			continue
		}
		if se.Name.Local != "sdnEntry" {
			continue
		}
		var x sdnEntry
		if err := dec.DecodeElement(&x, &se); err != nil {
			if cr.over {
				return nil, ErrListTooLarge
			}
			return nil, fmt.Errorf("ofac: entry: %w", err)
		}
		out = append(out, x.entry())
	}
	if cr.over {
		return nil, ErrListTooLarge
	}
	if !rooted {
		return nil, ErrNotSDNList
	}
	return out, nil
}

func (x sdnEntry) entry() sanctions.Entry {
	e := sanctions.Entry{
		Source: sanctions.SourceOFAC,
		ID:     strings.TrimSpace(x.UID),
		Name:   fullName(x.FirstName, x.LastName),
		Kind:   sanctions.KindOther,
	}
	if k, ok := kinds[strings.ToLower(strings.TrimSpace(x.SDNType))]; ok {
		e.Kind = k
	}
	for _, a := range x.AKAs {
		if n := fullName(a.FirstName, a.LastName); n != "" {
			e.Aliases = append(e.Aliases, n)
		}
	}
	for _, p := range x.Programs {
		if p = strings.TrimSpace(p); p != "" {
			e.Programs = append(e.Programs, p)
		}
	}
	// Countries: addresses, nationalities and citizenships (not the vessel flag), as unique sorted codes.
	seen := map[string]bool{}
	for _, list := range [][]place{x.Addresses, x.Nationalities, x.Citizenships} {
		for _, p := range list {
			if c, ok := countryCodes[strings.TrimSpace(p.Country)]; ok && !seen[c] {
				seen[c] = true
				e.Countries = append(e.Countries, c)
			}
		}
	}
	sort.Strings(e.Countries)
	return e
}

func fullName(first, last string) string {
	return strings.TrimSpace(strings.TrimSpace(first) + " " + strings.TrimSpace(last))
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
