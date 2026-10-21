package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/compliance/domain"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// ─── Classification ───────────────────────────────────────────────────────

func TestClassification_Happy(t *testing.T) {
	c, err := domain.NewClassification(domain.NewClassificationInput{
		TenantID: uuid.New(), TradeID: uuid.New(),
		Code: "32101", Description: "Importacao mercadoria", Nature: domain.NatureRemessa,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.Code() != "32101" || c.Nature() != domain.NatureRemessa {
		t.Errorf("classification: %s/%s", c.Code(), c.Nature())
	}
}

func TestClassification_BadInputs(t *testing.T) {
	base := domain.NewClassificationInput{
		TenantID: uuid.New(), TradeID: uuid.New(),
		Code: "32101", Description: "x", Nature: domain.NatureRemessa,
	}
	cases := []struct {
		name   string
		mutate func(*domain.NewClassificationInput)
	}{
		{"nil tenant", func(in *domain.NewClassificationInput) { in.TenantID = uuid.Nil }},
		{"short code", func(in *domain.NewClassificationInput) { in.Code = "12" }},
		{"non-numeric code", func(in *domain.NewClassificationInput) { in.Code = "AB123" }},
		{"bad nature", func(in *domain.NewClassificationInput) { in.Nature = "BOGUS" }},
		{"empty desc", func(in *domain.NewClassificationInput) { in.Description = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := base
			tc.mutate(&in)
			if _, err := domain.NewClassification(in); !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
}

// ─── IOFComputation ───────────────────────────────────────────────────────

func TestIOF_HappyGolden(t *testing.T) {
	// Default 0.38% rate × USD 10,000 notional = USD 38.00
	iof, err := domain.NewIOFComputation(domain.NewIOFInput{
		TenantID: uuid.New(), TradeID: uuid.New(),
		OperationType: "REMESSA",
		Notional:      decimal.NewFromInt(10000),
		NotionalCCY:   "USD",
		Rate:          dec("0.0038"),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if !iof.IOFAmount().Equal(dec("38.00")) {
		t.Fatalf("amount: got %s want 38.00", iof.IOFAmount())
	}
}

func TestIOF_TravelHighRate(t *testing.T) {
	// 1.10% rate × USD 5,000 = USD 55.00
	iof, _ := domain.NewIOFComputation(domain.NewIOFInput{
		TenantID: uuid.New(), TradeID: uuid.New(),
		OperationType: "TRAVEL_CASH",
		Notional:      decimal.NewFromInt(5000),
		NotionalCCY:   "USD",
		Rate:          dec("0.011"),
	})
	if !iof.IOFAmount().Equal(dec("55.00")) {
		t.Fatalf("amount: %s", iof.IOFAmount())
	}
}

func TestIOF_BadInputs(t *testing.T) {
	base := domain.NewIOFInput{
		TenantID: uuid.New(), TradeID: uuid.New(),
		OperationType: "REMESSA", Notional: dec("100"), NotionalCCY: "USD",
		Rate: dec("0.0038"),
	}
	cases := []struct {
		name   string
		mutate func(*domain.NewIOFInput)
	}{
		{"zero notional", func(in *domain.NewIOFInput) { in.Notional = dec("0") }},
		{"negative rate", func(in *domain.NewIOFInput) { in.Rate = dec("-0.01") }},
		{"rate above 1", func(in *domain.NewIOFInput) { in.Rate = dec("1.5") }},
		{"bad ccy", func(in *domain.NewIOFInput) { in.NotionalCCY = "DOLLAR" }},
		{"empty op", func(in *domain.NewIOFInput) { in.OperationType = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := base
			tc.mutate(&in)
			if _, err := domain.NewIOFComputation(in); !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
}

// ─── BACENReport ──────────────────────────────────────────────────────────

func TestBACENReport_Lifecycle_Accepted(t *testing.T) {
	r, err := domain.NewBACENReport(domain.NewBACENReportInput{
		TenantID:      uuid.New(),
		ReportType:    domain.ReportSISBACEN,
		ReferenceDate: time.Now().UTC(),
		PayloadHash:   "abc123def456",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := r.MarkSubmitted(time.Now().UTC()); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if err := r.MarkAccepted(time.Now().UTC()); err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if r.Status() != domain.StatusAccepted {
		t.Errorf("status: %s", r.Status())
	}
}

func TestBACENReport_Lifecycle_RejectedPath(t *testing.T) {
	r, _ := domain.NewBACENReport(domain.NewBACENReportInput{
		TenantID:      uuid.New(),
		ReportType:    domain.ReportCambio,
		ReferenceDate: time.Now().UTC(),
		PayloadHash:   "deadbeef",
	})
	_ = r.MarkSubmitted(time.Now().UTC())
	if err := r.MarkRejected(time.Now().UTC(), ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("missing reason: %v", err)
	}
	if err := r.MarkRejected(time.Now().UTC(), "duplicate"); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if r.Status() != domain.StatusRejected {
		t.Errorf("status: %s", r.Status())
	}
}

func TestBACENReport_BadInputs(t *testing.T) {
	base := domain.NewBACENReportInput{
		TenantID: uuid.New(), ReportType: domain.ReportCCS,
		ReferenceDate: time.Now().UTC(), PayloadHash: "x",
	}
	cases := []struct {
		name   string
		mutate func(*domain.NewBACENReportInput)
	}{
		{"nil tenant", func(in *domain.NewBACENReportInput) { in.TenantID = uuid.Nil }},
		{"bad type", func(in *domain.NewBACENReportInput) { in.ReportType = "BOGUS" }},
		{"zero date", func(in *domain.NewBACENReportInput) { in.ReferenceDate = time.Time{} }},
		{"empty hash", func(in *domain.NewBACENReportInput) { in.PayloadHash = "  " }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := base
			tc.mutate(&in)
			if _, err := domain.NewBACENReport(in); !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
}

// ─── ScreeningResult ──────────────────────────────────────────────────────

func TestScreening_LevelsAndCOS(t *testing.T) {
	cases := []struct {
		hits      []string
		wantLevel domain.RiskLevel
		wantCOS   bool
	}{
		{nil, domain.RiskLow, false},
		{[]string{"OFAC:SDN:demo"}, domain.RiskMedium, false},
		{[]string{"OFAC:SDN:a", "UN:1267:b", "EU:RestricMeasure:c"}, domain.RiskHigh, true},
	}
	for _, tc := range cases {
		t.Run(string(tc.wantLevel), func(t *testing.T) {
			s, err := domain.NewScreeningResult(domain.NewScreeningInput{
				TenantID:        uuid.New(),
				CounterpartyBIC: "CHASUS33",
				Hits:            tc.hits,
			})
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			if s.RiskLevel() != tc.wantLevel {
				t.Errorf("level: got %s want %s", s.RiskLevel(), tc.wantLevel)
			}
			if s.RequiresCOS() != tc.wantCOS {
				t.Errorf("RequiresCOS: got %v want %v", s.RequiresCOS(), tc.wantCOS)
			}
			if s.IsClear() != (tc.wantLevel == domain.RiskLow) {
				t.Errorf("IsClear flag mismatch")
			}
		})
	}
}

func TestScreening_BadInputs(t *testing.T) {
	_, err := domain.NewScreeningResult(domain.NewScreeningInput{TenantID: uuid.Nil, CounterpartyBIC: "CHASUS33"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("nil tenant: %v", err)
	}
	_, err = domain.NewScreeningResult(domain.NewScreeningInput{TenantID: uuid.New(), CounterpartyBIC: "SHORT"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("bad bic: %v", err)
	}
	_, err = domain.NewScreeningResult(domain.NewScreeningInput{TenantID: uuid.New(), CounterpartyBIC: "CHASUS33", LEI: "TOO-SHORT-LEI"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("bad lei: %v", err)
	}
}
