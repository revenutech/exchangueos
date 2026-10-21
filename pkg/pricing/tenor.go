package pricing

import (
	"fmt"
	"strings"
	"time"
)

// Tenor — standard FX tenor codes.
type Tenor string

const (
	TenorON   Tenor = "ON"   // overnight (T+0)
	TenorTN   Tenor = "TN"   // tom-next (T+1)
	TenorSN   Tenor = "SN"   // spot-next (spot + 1 BD)
	TenorSpot Tenor = "SPOT" // T+2 BD (default FX spot)
	Tenor1W   Tenor = "1W"
	Tenor2W   Tenor = "2W"
	Tenor3W   Tenor = "3W"
	Tenor1M   Tenor = "1M"
	Tenor2M   Tenor = "2M"
	Tenor3M   Tenor = "3M"
	Tenor6M   Tenor = "6M"
	Tenor9M   Tenor = "9M"
	Tenor1Y   Tenor = "1Y"
	Tenor18M  Tenor = "18M"
	Tenor2Y   Tenor = "2Y"
)

// BusinessCalendar is the minimal calendar interface ValueDate requires.
// Implementations live in modules/refdata/domain (refdata.Calendar already
// satisfies this contract), but pkg/pricing/ does not import modules/ — the
// caller passes any conforming impl.
type BusinessCalendar interface {
	IsBusinessDay(time.Time) bool
	AddBusinessDays(time.Time, int) time.Time
}

// ParseTenor parses canonical tenor strings (case-insensitive, leading/trailing space tolerated).
func ParseTenor(s string) (Tenor, error) {
	t := Tenor(strings.ToUpper(strings.TrimSpace(s)))
	switch t {
	case TenorON, TenorTN, TenorSN, TenorSpot,
		Tenor1W, Tenor2W, Tenor3W,
		Tenor1M, Tenor2M, Tenor3M, Tenor6M, Tenor9M,
		Tenor1Y, Tenor18M, Tenor2Y:
		return t, nil
	default:
		return "", fmt.Errorf("%w: unknown tenor %q", ErrInvalidInput, s)
	}
}

// ValueDate returns the FX value date for `tenor` given a `tradeDate` and a `BusinessCalendar`.
//
// Rules:
//
//   - ON   → tradeDate itself, advanced to the next BD if not already one.
//   - TN   → ON + 1 BD.
//   - Spot → ON + 2 BD (industry standard for most pairs).
//   - SN   → Spot + 1 BD.
//   - Week tenors (1W/2W/3W) → Spot + N calendar weeks, Modified-Following.
//   - Month tenors (1M/2M/3M/6M/9M/18M) → Spot + N calendar months, Modified-Following.
//   - Year tenors (1Y/2Y) → Spot + N×12 calendar months, Modified-Following.
//
// Modified-Following: if the resulting day is not a business day, advance to the next
// business day; if that next BD crosses into the following month, fall back to the
// PREVIOUS business day instead.
func ValueDate(tenor Tenor, tradeDate time.Time, cal BusinessCalendar) (time.Time, error) {
	if cal == nil {
		return time.Time{}, fmt.Errorf("%w: calendar is required", ErrInvalidInput)
	}
	td := truncateToDate(tradeDate)

	// "Today" must itself be a BD before we count Spot/ON.
	on := td
	if !cal.IsBusinessDay(on) {
		on = cal.AddBusinessDays(on, 1)
	}

	spot := cal.AddBusinessDays(on, 2)

	switch tenor {
	case TenorON:
		return on, nil
	case TenorTN:
		return cal.AddBusinessDays(on, 1), nil
	case TenorSpot:
		return spot, nil
	case TenorSN:
		return cal.AddBusinessDays(spot, 1), nil
	case Tenor1W:
		return modifiedFollowing(spot.AddDate(0, 0, 7), cal), nil
	case Tenor2W:
		return modifiedFollowing(spot.AddDate(0, 0, 14), cal), nil
	case Tenor3W:
		return modifiedFollowing(spot.AddDate(0, 0, 21), cal), nil
	case Tenor1M:
		return modifiedFollowing(addMonths(spot, 1), cal), nil
	case Tenor2M:
		return modifiedFollowing(addMonths(spot, 2), cal), nil
	case Tenor3M:
		return modifiedFollowing(addMonths(spot, 3), cal), nil
	case Tenor6M:
		return modifiedFollowing(addMonths(spot, 6), cal), nil
	case Tenor9M:
		return modifiedFollowing(addMonths(spot, 9), cal), nil
	case Tenor18M:
		return modifiedFollowing(addMonths(spot, 18), cal), nil
	case Tenor1Y:
		return modifiedFollowing(addMonths(spot, 12), cal), nil
	case Tenor2Y:
		return modifiedFollowing(addMonths(spot, 24), cal), nil
	default:
		return time.Time{}, fmt.Errorf("%w: unsupported tenor %q", ErrInvalidInput, tenor)
	}
}

// modifiedFollowing returns the next business day on or after d, unless that BD
// crosses into the next month — in which case it returns the previous business day.
func modifiedFollowing(d time.Time, cal BusinessCalendar) time.Time {
	d = truncateToDate(d)
	if cal.IsBusinessDay(d) {
		return d
	}
	originalMonth := d.Month()

	// Walk forward.
	forward := d
	for !cal.IsBusinessDay(forward) {
		forward = forward.AddDate(0, 0, 1)
	}
	if forward.Month() == originalMonth {
		return forward
	}

	// Crossed month — walk backward instead.
	back := d
	for !cal.IsBusinessDay(back) {
		back = back.AddDate(0, 0, -1)
	}
	return back
}

// addMonths adds n calendar months to d, clamping to the last day of the target
// month rather than spilling into the following one.
//
// time.Time.AddDate normalises overflow, which is wrong for FX tenors:
// 2026-08-31 plus one month becomes 2026-10-01 because September has no 31st,
// and 2026-01-31 plus one month becomes 2026-03-03. A 1M trade dealt on the
// 31st values on the last day of the following month.
//
// The overflow also defeated modifiedFollowing entirely: it compares the rolled
// date's month against the unadjusted date's month to decide whether to fall
// back, and an unadjusted date that had already spilled into the next month made
// that comparison meaningless.
//
// This is the "clamp to month end" part of the convention. The stricter
// end-of-month rule — spot on the last BUSINESS day of a month values on the
// last business day of the target month, e.g. 2026-08-28 (Fri, last BD of
// August) rolling to 2026-09-30 rather than 2026-09-28 — is a separate,
// counterparty-agreed flag and is deliberately not applied here.
func addMonths(d time.Time, n int) time.Time {
	day := d.Day()
	// Anchor on day 1 so the month arithmetic itself can never overflow.
	target := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, n, 0)
	if last := daysInMonth(target.Year(), target.Month()); day > last {
		day = last
	}
	return time.Date(target.Year(), target.Month(), day, 0, 0, 0, 0, time.UTC)
}

// daysInMonth returns the number of days in the given month. Day 0 of the
// following month is the last day of this one.
func daysInMonth(y int, m time.Month) int {
	return time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func truncateToDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
