package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/refdata/domain"
)

// ─── Currency ─────────────────────────────────────────────────────────────

func TestCurrency_Valid(t *testing.T) {
	c, err := domain.NewCurrency("usd", "United States Dollar", 2, true, false)
	if err != nil {
		t.Fatalf("NewCurrency: %v", err)
	}
	if c.Code() != "USD" {
		t.Errorf("code normalised: got %s want USD", c.Code())
	}
	if !c.IsActive() || !c.IsCLSEligible() || c.IsCFETSEligible() {
		t.Errorf("flags: active=%v cls=%v cfets=%v", c.IsActive(), c.IsCLSEligible(), c.IsCFETSEligible())
	}
	c.Deactivate()
	if c.IsActive() {
		t.Error("after Deactivate: still active")
	}
}

func TestCurrency_RejectsBadInputs(t *testing.T) {
	cases := []struct {
		name              string
		code              string
		curName           string
		minor             int
	}{
		{"short code", "US", "Dollar", 2},
		{"long code", "USDX", "Dollar", 2},
		{"numeric code", "US1", "Dollar", 2},
		{"empty name", "USD", "", 2},
		{"bad minor", "USD", "Dollar", 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewCurrency(tc.code, tc.curName, tc.minor, false, false)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
}

// ─── Calendar ─────────────────────────────────────────────────────────────

func TestCalendar_BusinessDay(t *testing.T) {
	cal, err := domain.NewCalendar("USD_NYC", []time.Time{
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC), // observed July 4
	})
	if err != nil {
		t.Fatalf("NewCalendar: %v", err)
	}

	cases := []struct {
		date       time.Time
		isBusiness bool
	}{
		{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), false}, // holiday
		{time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), true},  // Fri
		{time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), false}, // Sat
		{time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC), false}, // Sun
		{time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC), true},  // Mon
		{time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC), false}, // observed holiday
		{time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC), true},  // Mon after
	}
	for _, tc := range cases {
		got := cal.IsBusinessDay(tc.date)
		if got != tc.isBusiness {
			t.Errorf("IsBusinessDay(%s) = %v, want %v", tc.date.Format("2006-01-02"), got, tc.isBusiness)
		}
	}
}

func TestCalendar_AddBusinessDays(t *testing.T) {
	cal, _ := domain.NewCalendar("USD_NYC", []time.Time{
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	// Thursday + 2 BD with Fri ok, Sat/Sun skipped → next Monday
	start := time.Date(2026, 1, 8, 0, 0, 0, 0, time.UTC) // Thu
	got := cal.AddBusinessDays(start, 2)
	want := time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC) // Mon
	if !got.Equal(want) {
		t.Fatalf("AddBusinessDays(Thu, 2) = %s, want %s", got, want)
	}
}

// ─── BICRecord ────────────────────────────────────────────────────────────

func TestBIC_Valid(t *testing.T) {
	b, err := domain.NewBICRecord("DEUTDEFF", "Deutsche Bank AG", "DE", "7LTWFZYICNSX8D621K86")
	if err != nil {
		t.Fatalf("NewBICRecord: %v", err)
	}
	if b.BIC() != "DEUTDEFF" {
		t.Errorf("BIC: got %s want DEUTDEFF", b.BIC())
	}
	if !b.IsActive() {
		t.Error("expected active")
	}
}

func TestBIC_Invalid(t *testing.T) {
	cases := []struct {
		name, bic, country string
	}{
		{"short", "DEUTDEF", "DE"},
		{"long", "DEUTDEFFXXXY", "DE"},
		{"prefix non-alpha", "DEU1DEFF", "DE"},
		{"country segment non-alpha", "DEUT12FF", "DE"},
		{"country too short", "DEUTDEFF", "D"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewBICRecord(tc.bic, "Some Bank", tc.country, "")
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
}

// ─── SSI ──────────────────────────────────────────────────────────────────

func TestSSI_Valid(t *testing.T) {
	now := time.Now().UTC()
	s, err := domain.NewSSI(domain.NewSSIInput{
		TenantID:        uuid.New(),
		CounterpartyBIC: "CHASUS33",
		Currency:        "usd",
		BeneficiaryBIC:  "CHASUS33",
		AccountNumber:   "1234567890",
		ValidFrom:       now.AddDate(0, -1, 0),
		ValidTo:         now.AddDate(1, 0, 0),
	})
	if err != nil {
		t.Fatalf("NewSSI: %v", err)
	}
	if s.Currency() != "USD" {
		t.Errorf("currency normalised: got %s", s.Currency())
	}
	if !s.IsActiveAt(now) {
		t.Error("expected active now")
	}
	if s.IsActiveAt(now.AddDate(2, 0, 0)) {
		t.Error("expected expired in 2 years")
	}
	if s.IsActiveAt(now.AddDate(0, -6, 0)) {
		t.Error("expected not-yet-active 6 months ago")
	}
}

func TestSSI_RequiresAccountOrIBAN(t *testing.T) {
	now := time.Now().UTC()
	_, err := domain.NewSSI(domain.NewSSIInput{
		TenantID:        uuid.New(),
		CounterpartyBIC: "CHASUS33",
		Currency:        "USD",
		BeneficiaryBIC:  "CHASUS33",
		ValidFrom:       now,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestSSI_BadIBANLength(t *testing.T) {
	now := time.Now().UTC()
	_, err := domain.NewSSI(domain.NewSSIInput{
		TenantID:        uuid.New(),
		CounterpartyBIC: "CHASUS33",
		Currency:        "USD",
		BeneficiaryBIC:  "CHASUS33",
		IBAN:            "TOO-SHORT",
		ValidFrom:       now,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}
