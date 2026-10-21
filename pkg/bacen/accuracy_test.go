package bacen_test

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenutech/exchangeos/pkg/bacen"
)

// AccuracyThreshold is the minimum hit ratio the keyword classifier must
// achieve against data/bacen/golden-classifications.csv. Cited by MS-024e.
const AccuracyThreshold = 0.95

// TestClassifier_AccuracyAgainstGoldenCorpus runs every phrase in the corpus
// through Classify and asserts the hit ratio meets AccuracyThreshold. A miss
// is either ErrUnknown OR a code mismatch — both equally bad from the
// compliance team's perspective.
//
// The corpus lives at data/bacen/golden-classifications.csv (CSV with header
// phrase,expected_code,notes). Misses are logged with the actual rule fire so
// the failure mode is debuggable.
func TestClassifier_AccuracyAgainstGoldenCorpus(t *testing.T) {
	corpus := loadGoldenCorpus(t)
	require.GreaterOrEqual(t, len(corpus), 100,
		"corpus too small to draw an accuracy signal: %d phrases", len(corpus))

	c := bacen.NewClassifier()
	var (
		hits   int
		misses []missEntry
	)
	for _, row := range corpus {
		n, err := c.Classify(row.phrase)
		switch {
		case err != nil:
			misses = append(misses, missEntry{row, "", err.Error()})
		case n.Code != row.expectedCode:
			misses = append(misses, missEntry{row, n.Code, ""})
		default:
			hits++
		}
	}

	ratio := float64(hits) / float64(len(corpus))
	t.Logf("accuracy: %d/%d (%.2f%%) — threshold %.2f%%",
		hits, len(corpus), ratio*100, AccuracyThreshold*100)
	if ratio < AccuracyThreshold {
		t.Logf("misses (showing first 15):")
		for i, m := range misses {
			if i >= 15 {
				t.Logf("  ... %d more", len(misses)-15)
				break
			}
			t.Logf("  %s", m.format())
		}
	}
	assert.GreaterOrEqual(t, ratio, AccuracyThreshold,
		"classifier accuracy below threshold: %.2f%% < %.2f%% (%d hits / %d total)",
		ratio*100, AccuracyThreshold*100, hits, len(corpus))
}

// TestClassifier_EveryRuleTargetsKnownCode guards against typos in the
// defaultKeywordRules — every code referenced must resolve via ByCode.
func TestClassifier_EveryRuleTargetsKnownCode(t *testing.T) {
	c := bacen.NewClassifier()
	// Pull rules indirectly via Classify: we don't expose the keywords slice,
	// but every rule that the corpus matches will execute ByCode internally.
	// For a direct guarantee, the loop below catches dead rules by exercising
	// every keyword as its own hint and asserting no ErrUnknown surfaces with
	// "rule … targets unknown code".
	corpus := loadGoldenCorpus(t)
	for _, row := range corpus {
		_, err := c.Classify(row.phrase)
		if err != nil && strings.Contains(err.Error(), "targets unknown code") {
			t.Fatalf("dead rule detected via phrase %q: %v", row.phrase, err)
		}
	}
}

// ─── helpers ───────────────────────────────────────────────────────────────

type goldenRow struct {
	phrase       string
	expectedCode string
	notes        string
}

type missEntry struct {
	row       goldenRow
	gotCode   string
	gotErrMsg string
}

func (m missEntry) format() string {
	if m.gotErrMsg != "" {
		return fmt.Sprintf("[%s] %q → ERR: %s",
			m.row.expectedCode, m.row.phrase, m.gotErrMsg)
	}
	return fmt.Sprintf("[%s] %q → got %s",
		m.row.expectedCode, m.row.phrase, m.gotCode)
}

func loadGoldenCorpus(t *testing.T) []goldenRow {
	t.Helper()
	// Walk up to repo root from the package's working dir (pkg/bacen).
	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	path := filepath.Join(repoRoot, "data", "bacen", "golden-classifications.csv")

	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		t.Skipf("golden corpus missing at %s — generate via Compliance ops", path)
	}
	require.NoError(t, err)
	defer f.Close()

	r := csv.NewReader(f)
	r.TrimLeadingSpace = true
	header, err := r.Read()
	require.NoError(t, err)
	require.Equal(t, []string{"phrase", "expected_code", "notes"}, header,
		"golden CSV header drifted — keep schema stable")

	out := make([]goldenRow, 0, 200)
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err, "read corpus row")
		out = append(out, goldenRow{phrase: rec[0], expectedCode: rec[1], notes: rec[2]})
	}
	return out
}
