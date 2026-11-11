package sanctions

import (
	"strings"
	"unicode"
)

// Jaro-Winkler similarity, written here to keep the module free of new dependencies.
//
// Jaro (1989), «Advances in Record-Linkage Methodology as Applied to Matching the 1985 Census of Tampa, Florida»,
// JASA 84(406); Winkler (1990), «String Comparator Metrics and Enhanced Decision Rules in the Fellegi-Sunter Model of
// Record Linkage», Proc. Section on Survey Research Methods, ASA:
//
//	jaro = (m/|a| + m/|b| + (m - t)/m) / 3
//	jw   = jaro + ℓ·p·(1 - jaro)
//
// m = characters that match within a window of max(|a|,|b|)/2 - 1 positions; t = half the matched characters that
// are out of order; ℓ = common prefix length, capped at 4; p = 0.1 (Winkler's scaling factor).
// The result is a similarity in [0, 1], never money (see Match.Score).
const (
	winklerPrefixCap = 4
	winklerScale     = 0.1
)

func jaroWinkler(a, b string) float64 { return jaroWinklerRunes([]rune(a), []rune(b)) }

func jaroWinklerRunes(a, b []rune) float64 {
	j := jaroRunes(a, b)
	if j == 0 || j == 1 {
		return j
	}
	prefix := 0
	for prefix < min(len(a), len(b), winklerPrefixCap) && a[prefix] == b[prefix] {
		prefix++
	}
	return j + float64(prefix)*winklerScale*(1-j)
}

// jaroRunes is the Jaro similarity alone, without Winkler's prefix boost.
func jaroRunes(a, b []rune) float64 {
	la, lb := len(a), len(b)
	if la == 0 && lb == 0 {
		return 1
	}
	if la == 0 || lb == 0 {
		return 0
	}
	window := max(la, lb)/2 - 1
	if window < 0 {
		window = 0
	}
	var bufA, bufB [64]bool
	ma, mb := flags(bufA[:], la), flags(bufB[:], lb)
	m := 0
	for i, r := range a {
		lo, hi := max(0, i-window), min(lb, i+window+1)
		for j := lo; j < hi; j++ {
			if !mb[j] && b[j] == r {
				ma[i], mb[j] = true, true
				m++
				break
			}
		}
	}
	if m == 0 {
		return 0
	}
	half, k := 0, 0
	for i, r := range a {
		if !ma[i] {
			continue
		}
		for !mb[k] {
			k++
		}
		if r != b[k] {
			half++
		}
		k++
	}
	fm := float64(m)
	return (fm/float64(la) + fm/float64(lb) + (fm-float64(half)/2)/fm) / 3
}

// flags returns n cleared match flags, on the caller's stack buffer when it is large enough.
func flags(buf []bool, n int) []bool {
	if n <= len(buf) {
		return buf[:n]
	}
	return make([]bool, n)
}

// fold maps the Latin letters with diacritics (and the few ligatures) found in list names to plain ASCII, after
// lower-casing. Combining marks of decomposed input are dropped separately in normalize.
var fold = func() map[rune]string {
	groups := map[string]string{
		"àáâãäåāăą": "a", "çćĉċč": "c", "ďđð": "d", "èéêëēĕėęě": "e", "ĝğġģ": "g", "ĥħ": "h",
		"ìíîïĩīĭįı": "i", "ĵ": "j", "ķ": "k", "ĺļľŀł": "l", "ñńņňŉ": "n", "òóôõöøōŏő": "o", "ŕŗř": "r",
		"śŝşšș": "s", "ţťŧț": "t", "ùúûüũūŭůűų": "u", "ŵ": "w", "ýÿŷ": "y", "źżž": "z",
		"ß": "ss", "æ": "ae", "œ": "oe", "þ": "th",
	}
	out := make(map[rune]string, 128)
	for runes, plain := range groups {
		for _, r := range runes {
			out[r] = plain
		}
	}
	return out
}()

// normalize case-folds, strips diacritics, and turns every run of non-letters/non-digits into one space.
// Letters of other scripts (Arabic, Cyrillic, CJK) are kept as they are.
func normalize(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	space := false
	write := func(str string) {
		if space && sb.Len() > 0 {
			sb.WriteByte(' ')
		}
		space = false
		sb.WriteString(str)
	}
	for _, r := range s {
		r = unicode.ToLower(r)
		switch {
		case unicode.Is(unicode.Mn, r):
			// combining mark of a decomposed letter: dropped
		case fold[r] != "":
			write(fold[r])
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			write(string(r))
		default:
			space = true
		}
	}
	return sb.String()
}
