package sanctions

import (
	"cmp"
	"fmt"
	"math/bits"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

// tokenFloor is the token-level Jaro similarity a query token needs with a list token for the entries holding that
// token to become candidates. A lower MinScore lowers it, so a permissive query is never pruned harder than it asked.
// It is plain Jaro on purpose: Winkler's prefix boost would let every token of the same prefix block through
// («bakori»/«bakozen» is 0.85 Jaro-Winkler, 0.75 Jaro), and the candidate set would grow to a large part of the list.
const tokenFloor = 0.8

// tokenLenRatio is the shortest/longest rune-length ratio a list token needs against the query token to be a hit.
// Without it a short token hits every longer token that extends it (Jaro «bari»/«barimal» is 0.86), and a two-token
// query scores a large part of the list. A typo, an insertion or a deletion keeps the ratio near 1. Like tokenFloor,
// a lower MinScore lowers it.
const tokenLenRatio = 0.75

// blockLen is the prefix/suffix length (in runes) of the blocking keys of the token dictionary.
const blockLen = 3

// Cache is the in-memory Index of one sanctions list (MS-024d 1.2): an inverted index from tokens to entries, and a
// Jaro-Winkler scorer over the primary name and the aliases.
//
// Candidates are found without a full scan: each distinct list token is filed under its first and last three runes;
// a query token is compared only with the list tokens that share one of those keys, and a token at least
// tokenFloor-similar (Jaro) to the query token, and of comparable length (tokenLenRatio), is a hit. A name (primary or alias) is scored only when it holds hits
// for at least half the query tokens, rounded up: one of one or two, two of three or four, and so on. What this
// blocking misses: a typo in both the first and the last three runes of a query token, and a name where more than
// half of the query tokens have no similar token at all.
//
// A name scores the larger of the Jaro-Winkler of the normalised strings and of their tokens sorted (so
// «FAKERTON, Zebulon» meets «Zebulon Fakerton»); an entry scores its best name.
//
// Load builds the next snapshot aside and swaps it in one atomic store; Match reads whichever snapshot is current,
// without locks. The zero value is an empty cache ready for use.
type Cache struct {
	mu   sync.Mutex // serialises Load
	snap atomic.Pointer[snapshot]
}

var _ Index = (*Cache)(nil)

// NewCache returns an empty cache.
func NewCache() *Cache { return &Cache{} }

type variant struct {
	norm, sorted []rune
	inOrder      bool // the tokens are already sorted: sorted == norm
}

type cachedEntry struct {
	source    Source
	id, name  string
	countries []string
	variants  []variant
}

// posting is one name (variant) of one entry holding a token.
type posting struct {
	entry   int32
	variant int32
}

type snapshot struct {
	entries  []cachedEntry
	dict     [][]rune           // distinct list tokens
	postings [][]posting        // dict index → names holding the token, ascending, distinct
	blocks   map[string][]int32 // blocking key → dict indexes
}

// maxQueryTokens bounds the per-name hit mask; a longer query is screened on its first 64 distinct tokens.
const maxQueryTokens = 64

// Len reports how many entries the current snapshot holds.
func (c *Cache) Len() int {
	if s := c.snap.Load(); s != nil {
		return len(s.entries)
	}
	return 0
}

// Load replaces the cache content with entries. An empty list, an invalid entry, a duplicate (source, id) or an entry
// whose names normalise to nothing is refused, and the previous snapshot keeps serving. The entries are copied.
func (c *Cache) Load(entries []Entry) error {
	if len(entries) == 0 {
		return ErrEmptyList
	}
	s := &snapshot{entries: make([]cachedEntry, 0, len(entries)), blocks: make(map[string][]int32)}
	seen := make(map[string]struct{}, len(entries))
	dictIdx := make(map[string]int32)
	for i, e := range entries {
		if err := e.Validate(); err != nil {
			return err
		}
		key := string(e.Source) + "\x00" + e.ID
		if _, dup := seen[key]; dup {
			return fmt.Errorf("%w: duplicate %s entry %s", ErrInvalidEntry, e.Source, e.ID)
		}
		seen[key] = struct{}{}
		ce := cachedEntry{source: e.Source, id: e.ID, name: e.Name, countries: slices.Clone(e.Countries)}
		for _, n := range append([]string{e.Name}, e.Aliases...) {
			toks := strings.Fields(normalize(n))
			if len(toks) == 0 {
				continue
			}
			pv := posting{entry: int32(i), variant: int32(len(ce.variants))}
			ce.variants = append(ce.variants, newVariant(toks))
			for _, t := range toks {
				d, ok := dictIdx[t]
				if !ok {
					d = int32(len(s.dict))
					dictIdx[t] = d
					r := []rune(t)
					s.dict = append(s.dict, r)
					s.postings = append(s.postings, nil)
					for _, k := range blockKeys(r) {
						s.blocks[k] = append(s.blocks[k], d)
					}
				}
				if p := s.postings[d]; len(p) == 0 || p[len(p)-1] != pv {
					s.postings[d] = append(p, pv)
				}
			}
		}
		if len(ce.variants) == 0 {
			return fmt.Errorf("%w: %s entry %s has no name left after normalisation", ErrInvalidEntry, e.Source, e.ID)
		}
		s.entries = append(s.entries, ce)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.snap.Store(s)
	return nil
}

// scratch holds the per-query maps; Match reuses them through scratchPool instead of allocating them per call.
type scratch struct {
	hits  map[posting]uint64 // name → bit mask of the query tokens it hit
	tried map[int32]struct{} // dict tokens already compared with the current query token
	best  map[int32]float64  // entry → best name score
}

var scratchPool = sync.Pool{New: func() any {
	return &scratch{hits: map[posting]uint64{}, tried: map[int32]struct{}{}, best: map[int32]float64{}}
}}

func (sc *scratch) release() {
	clear(sc.hits)
	clear(sc.tried)
	clear(sc.best)
	scratchPool.Put(sc)
}

func lenRatio(a, b int) float64 { return float64(min(a, b)) / float64(max(a, b)) }

// distinct keeps the first occurrence of each token, up to maxQueryTokens.
func distinct(toks []string) []string {
	out := make([]string, 0, min(len(toks), maxQueryTokens))
	for _, t := range toks {
		if len(out) == maxQueryTokens {
			break
		}
		if !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out
}

func newVariant(toks []string) variant {
	sorted := slices.Clone(toks)
	slices.Sort(sorted)
	return variant{
		norm: []rune(strings.Join(toks, " ")), sorted: []rune(strings.Join(sorted, " ")),
		inOrder: slices.Equal(toks, sorted),
	}
}

// blockKeys files a token under its first and last blockLen runes (the token itself when shorter).
func blockKeys(t []rune) []string {
	if len(t) <= blockLen {
		return []string{string(t)}
	}
	return []string{"<" + string(t[:blockLen]), ">" + string(t[len(t)-blockLen:])}
}

// Match returns the entries whose similarity to q reaches q.MinScore, best first, ties by source then entry id.
// When q.Country is set, entries listing countries must list it; entries without countries always pass.
func (c *Cache) Match(q MatchQuery) ([]Match, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	toks := strings.Fields(normalize(q.Name))
	if len(toks) == 0 {
		return nil, fmt.Errorf("%w: name has nothing to compare after normalisation", ErrInvalidQuery)
	}
	s := c.snap.Load()
	if s == nil {
		return nil, nil
	}
	floor, ratio := min(tokenFloor, q.MinScore), min(tokenLenRatio, q.MinScore)
	qtoks := distinct(toks)
	need := (len(qtoks) + 1) / 2
	sc := scratchPool.Get().(*scratch)
	defer sc.release()
	hits, tried, best := sc.hits, sc.tried, sc.best
	for qi, t := range qtoks {
		tr, bit := []rune(t), uint64(1)<<qi
		for _, k := range blockKeys(tr) {
			for _, d := range s.blocks[k] {
				if _, done := tried[d]; done {
					continue
				}
				tried[d] = struct{}{}
				dt := s.dict[d]
				if lenRatio(len(tr), len(dt)) >= ratio && jaroRunes(tr, dt) >= floor {
					for _, p := range s.postings[d] {
						hits[p] |= bit
					}
				}
			}
		}
		clear(tried)
	}
	qv := newVariant(toks)
	multi := len(toks) > 1
	for p, mask := range hits {
		if bits.OnesCount64(mask) < need {
			continue
		}
		ce := &s.entries[p.entry]
		if q.Country != "" && len(ce.countries) > 0 && !slices.ContainsFunc(ce.countries, func(c string) bool {
			return strings.EqualFold(c, q.Country)
		}) {
			continue
		}
		v := &ce.variants[p.variant]
		score := jaroWinklerRunes(qv.norm, v.norm)
		if multi && score < 1 && !(qv.inOrder && v.inOrder) {
			score = max(score, jaroWinklerRunes(qv.sorted, v.sorted))
		}
		if prev, ok := best[p.entry]; !ok || score > prev {
			best[p.entry] = score
		}
	}
	var out []Match
	for e, score := range best {
		if score >= q.MinScore {
			ce := &s.entries[e]
			out = append(out, Match{Source: ce.source, EntryID: ce.id, Name: ce.name, Score: score})
		}
	}
	slices.SortFunc(out, func(a, b Match) int {
		if c := cmp.Compare(b.Score, a.Score); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Source, b.Source); c != 0 {
			return c
		}
		return cmp.Compare(a.EntryID, b.EntryID)
	})
	return out, nil
}
