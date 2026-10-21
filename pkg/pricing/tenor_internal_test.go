package pricing

import (
	"testing"
	"time"
)

func d(y int, m int, day int) time.Time {
	return time.Date(y, time.Month(m), day, 0, 0, 0, 0, time.UTC)
}

func ymd(t time.Time) string { return t.UTC().Format("2006-01-02") }

// --- Month-end clamping ---
//
// Month and year tenors used time.Time.AddDate, which normalises overflow: a
// spot of 2026-08-31 rolled to 2026-10-01 instead of 2026-09-30, and 2026-01-31
// rolled to 2026-03-03. Nothing covered a month boundary, so it went unnoticed —
// and it also silently defeated the Modified-Following fallback, whose
// month-comparison operates on the unadjusted date.

func TestAddMonths_ClampsToLastDayOfTargetMonth(t *testing.T) {
	tests := []struct {
		name   string
		from   time.Time
		months int
		want   time.Time
	}{
		{"31 Jan +1M → 28 Feb", d(2026, 1, 31), 1, d(2026, 2, 28)},
		{"31 Jan +1M leap year → 29 Feb", d(2028, 1, 31), 1, d(2028, 2, 29)},
		{"31 Aug +1M → 30 Sep", d(2026, 8, 31), 1, d(2026, 9, 30)},
		{"31 May +1M → 30 Jun", d(2026, 5, 31), 1, d(2026, 6, 30)},
		{"31 Mar +3M → 30 Jun", d(2026, 3, 31), 3, d(2026, 6, 30)},
		{"29 Feb +12M → 28 Feb", d(2028, 2, 29), 12, d(2029, 2, 28)},
		{"31 Dec +1M crosses the year", d(2026, 12, 31), 1, d(2027, 1, 31)},
		{"30 Nov +3M → 28 Feb", d(2026, 11, 30), 3, d(2027, 2, 28)},
		// Days that exist in the target month are untouched.
		{"15 Jan +1M → 15 Feb", d(2026, 1, 15), 1, d(2026, 2, 15)},
		{"30 Jan +1M → 28 Feb (still clamped)", d(2026, 1, 30), 1, d(2026, 2, 28)},
		{"31 Jan +24M → 31 Jan", d(2026, 1, 31), 24, d(2028, 1, 31)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := addMonths(tt.from, tt.months); !got.Equal(tt.want) {
				t.Errorf("addMonths(%s, %d) = %s, want %s",
					ymd(tt.from), tt.months, ymd(got), ymd(tt.want))
			}
		})
	}
}

// addMonths must never land in a month other than the arithmetic target — the
// exact failure mode of AddDate.
func TestAddMonths_NeverOverflowsIntoTheFollowingMonth(t *testing.T) {
	for year := 2026; year <= 2029; year++ {
		for month := 1; month <= 12; month++ {
			for day := 28; day <= 31; day++ {
				last := daysInMonth(year, time.Month(month))
				if day > last {
					continue
				}
				from := d(year, month, day)
				for _, n := range []int{1, 2, 3, 6, 9, 12, 18, 24} {
					got := addMonths(from, n)

					wantYear := year + (month-1+n)/12
					wantMonth := time.Month((month-1+n)%12 + 1)
					if got.Year() != wantYear || got.Month() != wantMonth {
						t.Fatalf("addMonths(%s, %d) = %s — landed in %s %d, want %s %d",
							ymd(from), n, ymd(got), got.Month(), got.Year(), wantMonth, wantYear)
					}
				}
			}
		}
	}
}
