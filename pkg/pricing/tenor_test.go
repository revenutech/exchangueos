package pricing_test

import (
	"errors"
	"testing"
	"time"

	"github.com/revenutech/exchangeos/pkg/pricing"
)

// stubCalendar — local in-test BusinessCalendar (avoids modules/ import).
type stubCalendar struct {
	holidays map[string]bool
}

func newStub(holidays ...time.Time) *stubCalendar {
	c := &stubCalendar{holidays: make(map[string]bool, len(holidays))}
	for _, h := range holidays {
		c.holidays[ymd(h)] = true
	}
	return c
}

func ymd(t time.Time) string { return t.UTC().Format("2006-01-02") }

func (c *stubCalendar) IsBusinessDay(t time.Time) bool {
	wd := t.UTC().Weekday()
	if wd == time.Saturday || wd == time.Sunday {
		return false
	}
	return !c.holidays[ymd(t)]
}

func (c *stubCalendar) AddBusinessDays(t time.Time, n int) time.Time {
	if n == 0 {
		return t
	}
	step := 1
	if n < 0 {
		step = -1
		n = -n
	}
	cur := t
	for i := 0; i < n; i++ {
		cur = cur.AddDate(0, 0, step)
		for !c.IsBusinessDay(cur) {
			cur = cur.AddDate(0, 0, step)
		}
	}
	return cur
}

func d(y, m, day int) time.Time {
	return time.Date(y, time.Month(m), day, 0, 0, 0, 0, time.UTC)
}

func TestParseTenor(t *testing.T) {
	for _, s := range []string{"on", "ON ", " 3M", "1y"} {
		if _, err := pricing.ParseTenor(s); err != nil {
			t.Errorf("ParseTenor(%q): %v", s, err)
		}
	}
	if _, err := pricing.ParseTenor("BOGUS"); !errors.Is(err, pricing.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for BOGUS, got %v", err)
	}
}

// Trade Wed 2026-05-20, no holidays.
// ON  = 2026-05-20 (Wed)
// TN  = 2026-05-21 (Thu)
// Spot= 2026-05-22 (Fri)
// SN  = 2026-05-25 (Mon, skips weekend)
func TestValueDate_OnTnSpotSn(t *testing.T) {
	cal := newStub()
	trade := d(2026, 5, 20)

	cases := map[pricing.Tenor]time.Time{
		pricing.TenorON:   d(2026, 5, 20),
		pricing.TenorTN:   d(2026, 5, 21),
		pricing.TenorSpot: d(2026, 5, 22),
		pricing.TenorSN:   d(2026, 5, 25),
	}
	for tn, want := range cases {
		got, err := pricing.ValueDate(tn, trade, cal)
		if err != nil {
			t.Errorf("ValueDate(%s): %v", tn, err)
			continue
		}
		if !got.Equal(want) {
			t.Errorf("ValueDate(%s) = %s, want %s", tn, ymd(got), ymd(want))
		}
	}
}

// Trade on Sat → ON should roll to Mon. Spot = Mon + 2 BD = Wed.
func TestValueDate_TradeOnWeekendRollsToBD(t *testing.T) {
	cal := newStub()
	trade := d(2026, 5, 23) // Saturday
	on, _ := pricing.ValueDate(pricing.TenorON, trade, cal)
	if !on.Equal(d(2026, 5, 25)) {
		t.Fatalf("ON: got %s want 2026-05-25 (Mon)", ymd(on))
	}
	spot, _ := pricing.ValueDate(pricing.TenorSpot, trade, cal)
	if !spot.Equal(d(2026, 5, 27)) {
		t.Fatalf("Spot: got %s want 2026-05-27 (Wed)", ymd(spot))
	}
}

// Trade Wed 2026-05-20, Spot = Fri 2026-05-22.
// 1W = Spot + 7d = Fri 2026-05-29 (BD, no roll).
// 1M = Spot + 1 month = Mon 2026-06-22 (BD, no roll).
// 3M = Spot + 3 months = Mon 2026-08-24 (Fri 2026-08-22 is in Aug; Aug 22 is Sat,
//
//	Aug 23 is Sun → forward to Mon Aug 24 still in same month).
//
// 6M = Spot + 6 months = Mon 2026-11-23 (Sun Nov 22 → forward to Mon Nov 23).
// 1Y = Spot + 12 months = Mon 2027-05-24 (Sat May 22, Sun May 23 → forward to Mon May 24).
func TestValueDate_StandardTenors(t *testing.T) {
	cal := newStub()
	trade := d(2026, 5, 20)

	cases := []struct {
		tenor pricing.Tenor
		want  time.Time
	}{
		{pricing.Tenor1W, d(2026, 5, 29)},
		{pricing.Tenor1M, d(2026, 6, 22)},
		{pricing.Tenor3M, d(2026, 8, 24)},
		{pricing.Tenor6M, d(2026, 11, 23)},
		{pricing.Tenor1Y, d(2027, 5, 24)},
	}
	for _, tc := range cases {
		got, err := pricing.ValueDate(tc.tenor, trade, cal)
		if err != nil {
			t.Errorf("ValueDate(%s): %v", tc.tenor, err)
			continue
		}
		if !got.Equal(tc.want) {
			t.Errorf("ValueDate(%s) = %s, want %s", tc.tenor, ymd(got), ymd(tc.want))
		}
	}
}

// Modified-Following: when forward roll would cross month, must go back to last BD of month.
// Construct: target falls on weekend at end of month, AND next month starts with a weekend
// or holiday → next BD is in following month → must fall back to previous BD.
//
// Example: spot = Wed 2026-04-29; 1M = spot + 1 month = Fri 2026-05-29. May 30 = Sat,
// May 31 = Sun, June 1 = Mon (BD in next month). May 29 itself is a Friday, so no issue here.
// We need to engineer a date that DOES cross the month. Use a holiday on the last
// business day to force the roll.
//
// Construct: spot = Mon 2026-08-31 (last day of Aug, a Monday). 1M = Wed 2026-09-30 (BD).
// To test the previous-BD fallback, mark Sept 30 (Wed) AND Oct 1 (Thu) AND Oct 2 (Fri) as
// holidays → forward roll would land on Mon Oct 5 (next month) → must fall back to last
// BD in Sept, which is Tue Sept 29.
func TestValueDate_ModifiedFollowing_FallbackPreviousBD(t *testing.T) {
	cal := newStub(
		d(2026, 9, 30), // Wed holiday
		d(2026, 10, 1), // Thu holiday
		d(2026, 10, 2), // Fri holiday
	)
	// Force spot to be 2026-08-31 (Mon). Pick trade = 2026-08-27 (Thu) → spot = 2026-08-31 (Mon).
	trade := d(2026, 8, 27)
	got, err := pricing.ValueDate(pricing.Tenor1M, trade, cal)
	if err != nil {
		t.Fatalf("ValueDate: %v", err)
	}
	want := d(2026, 9, 29) // fall back to Tue Sept 29 (last BD in Sept)
	if !got.Equal(want) {
		t.Fatalf("ModFollowing fallback: got %s want %s", ymd(got), ymd(want))
	}
}

func TestValueDate_NilCalendar_Rejected(t *testing.T) {
	_, err := pricing.ValueDate(pricing.TenorSpot, d(2026, 5, 20), nil)
	if !errors.Is(err, pricing.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestValueDate_UnknownTenor_Rejected(t *testing.T) {
	cal := newStub()
	_, err := pricing.ValueDate(pricing.Tenor("BOGUS"), d(2026, 5, 20), cal)
	if !errors.Is(err, pricing.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

// End-to-end through ValueDate: a month tenor off a month-end spot must not
// spill past the target month.
func TestValueDate_MonthTenorFromMonthEndDoesNotSpill(t *testing.T) {
	cal := newStub() // no holidays; weekends only

	// 2026-01-27 (Tue) → spot 2026-01-29 (Thu). 1M → 2026-02-28 (Sat, not a BD)
	// → Modified-Following walks forward to Mon 2026-03-02, which crosses the
	// month, so it falls back to Fri 2026-02-27.
	got, err := pricing.ValueDate(pricing.Tenor1M, d(2026, 1, 27), cal)
	if err != nil {
		t.Fatalf("ValueDate: %v", err)
	}
	if want := d(2026, 2, 27); !got.Equal(want) {
		t.Errorf("1M from month-end spot = %s, want %s", ymd(got), ymd(want))
	}
	if got.Month() != time.February {
		t.Errorf("value date landed in %s, want February", got.Month())
	}
}
