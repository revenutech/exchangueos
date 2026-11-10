package sanctions

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestMatchQueryNeverLeaksName(t *testing.T) {
	t.Parallel()
	const name = "Joao da Silva"
	q := MatchQuery{Name: name, Country: "BR", MinScore: 0.85}

	var outs []string
	for _, f := range []string{"%v", "%+v", "%s"} {
		outs = append(outs, fmt.Sprintf(f, q))
	}
	var text, js bytes.Buffer
	slog.New(slog.NewTextHandler(&text, nil)).Info("screen", "q", q)
	slog.New(slog.NewJSONHandler(&js, nil)).Info("screen", "q", q)
	outs = append(outs, text.String(), js.String())

	for i, out := range outs {
		if strings.Contains(out, "Joao") || strings.Contains(out, name) {
			t.Errorf("output %d leaks the name: %s", i, out)
		}
		if !strings.Contains(out, "BR") || !strings.Contains(out, "13") {
			t.Errorf("output %d lost country or name length: %s", i, out)
		}
	}
}
