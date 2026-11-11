package sanctions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Vectors from Winkler (1990), «String Comparator Metrics and Enhanced Decision Rules in the Fellegi-Sunter Model of
// Record Linkage», and the usual textbook examples derived from it.
func TestJaroWinkler_LiteratureVectors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		a, b string
		want float64
	}{
		{"MARTHA", "MARHTA", 0.961},
		{"DWAYNE", "DUANE", 0.840},
		{"DIXON", "DICKSONX", 0.813},
		{"JELLYFISH", "SMELLYFISH", 0.896},
		{"SAME", "SAME", 1},
		{"ABC", "XYZ", 0},
		{"", "", 1},
		{"A", "", 0},
	}
	for _, c := range cases {
		t.Run(c.a+"/"+c.b, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, c.want, jaroWinkler(c.a, c.b), 0.001)
			assert.InDelta(t, c.want, jaroWinkler(c.b, c.a), 0.001, "symmetric")
		})
	}
}

func TestJaroWinkler_Unicode(t *testing.T) {
	t.Parallel()
	// Runes, not bytes: a two-byte rune counts as one character.
	assert.InDelta(t, jaroWinkler("MARTHA", "MARHTA"), jaroWinkler("MÄRTHA", "MÄRHTA"), 0.0001)
}

func TestNormalize(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"José  da Silva":          "jose da silva",
		"AL-QAIDA, (a.k.a.)":      "al qaida a k a",
		"Ærøskøbing Straße":       "aeroskobing strasse",
		"  Łódź\tŻółć  ":          "lodz zolc",
		"ÇAĞRI Öztürk":            "cagri ozturk",
		"Synthetic Trading 2 LLC": "synthetic trading 2 llc",
		"...":                     "",
	}
	for in, want := range cases {
		assert.Equal(t, want, normalize(in), in)
	}
}

func BenchmarkJaroWinkler(b *testing.B) {
	for b.Loop() {
		_ = jaroWinkler("synthetic trading company", "synthetik tradng company")
	}
}
