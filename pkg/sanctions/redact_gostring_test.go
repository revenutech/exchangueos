package sanctions

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// syntheticName is invented; it is the marker the redaction tests look for.
const syntheticName = "Zebulon Quixfeather"

// rawMatchQuery has MatchQuery's fields and none of its methods: the negative control. Converting a MatchQuery to
// it must leak the name, which proves the detector below would catch a leak in MatchQuery itself.
type rawMatchQuery struct {
	Name     string
	Country  string
	MinScore float64
}

type wrapsQuery struct {
	Q MatchQuery
}

type wrapsQueryPtr struct {
	Q *MatchQuery
}

func leaksName(out string) bool {
	return strings.Contains(out, syntheticName) || strings.Contains(out, "Zebulon") || strings.Contains(out, "Quixfeather")
}

func TestMatchQuery_GoStringAndJSONNeverLeakName(t *testing.T) {
	t.Parallel()
	q := MatchQuery{Name: syntheticName, Country: "BR", MinScore: 0.85}
	mustJSON := func(v any) string {
		b, err := json.Marshal(v)
		require.NoError(t, err)
		return string(b)
	}
	cases := []struct {
		name string
		out  string
	}{
		{"%#v value", fmt.Sprintf("%#v", q)},
		{"%#v pointer", fmt.Sprintf("%#v", &q)},
		{"%#v nested value", fmt.Sprintf("%#v", wrapsQuery{Q: q})},
		{"%#v nested pointer", fmt.Sprintf("%#v", wrapsQueryPtr{Q: &q})},
		{"%#v slice", fmt.Sprintf("%#v", []MatchQuery{q})},
		{"json value", mustJSON(q)},
		{"json pointer", mustJSON(&q)},
		{"json nested value", mustJSON(wrapsQuery{Q: q})},
		{"json nested pointer", mustJSON(wrapsQueryPtr{Q: &q})},
		{"json slice", mustJSON([]MatchQuery{q})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.False(t, leaksName(tc.out), "output leaks the name: %s", tc.out)
			assert.Contains(t, tc.out, "BR", "country lost: %s", tc.out)
			assert.Contains(t, tc.out, "19", "name length lost: %s", tc.out)
		})
	}
}

func TestMatchQuery_JSONRedactedShape(t *testing.T) {
	t.Parallel()
	b, err := json.Marshal(MatchQuery{Name: syntheticName, Country: "BR", MinScore: 0.85})
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))
	assert.Equal(t, map[string]any{"country": "BR", "min_score": 0.85, "name_len": float64(19)}, got)
}

// TestMatchQuery_RedactionNegativeControl shows the detector is not vacuous: the same fields without MatchQuery's
// methods do print the name, under both %#v and json.Marshal.
func TestMatchQuery_RedactionNegativeControl(t *testing.T) {
	t.Parallel()
	raw := rawMatchQuery(MatchQuery{Name: syntheticName, Country: "BR", MinScore: 0.85})
	b, err := json.Marshal(raw)
	require.NoError(t, err)
	cases := []struct {
		name string
		out  string
	}{
		{"%#v", fmt.Sprintf("%#v", raw)},
		{"json", string(b)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.True(t, leaksName(tc.out), "the control should leak the name, got: %s", tc.out)
		})
	}
}
