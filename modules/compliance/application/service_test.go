package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/compliance/application"
	"github.com/revenutech/exchangeos/modules/compliance/domain"
	"github.com/revenutech/exchangeos/modules/compliance/infrastructure/memory"
	"github.com/revenutech/exchangeos/pkg/bacen"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// fakeScreener stands in for a watchlist provider.
type fakeScreener struct {
	hits []string
	err  error
	// calls records what the service actually asked the provider about.
	calls []struct{ bic, lei string }
}

func (f *fakeScreener) Screen(_ context.Context, bic, lei string) ([]string, error) {
	f.calls = append(f.calls, struct{ bic, lei string }{bic, lei})
	return f.hits, f.err
}

func newSvc(t *testing.T) (*application.Service, *memory.ScreeningRepo) {
	t.Helper()
	svc, repo, _ := newSvcWith(t, &fakeScreener{})
	return svc, repo
}

// newSvcWith builds the service around a specific screener.
func newSvcWith(t *testing.T, screener application.SanctionsScreener) (*application.Service, *memory.ScreeningRepo, application.SanctionsScreener) {
	t.Helper()
	screen := memory.NewScreeningRepo()
	svc := application.NewService(
		bacen.NewClassifier(),
		bacen.NewIOFCalculator(),
		memory.NewClassificationRepo(),
		memory.NewIOFRepo(),
		memory.NewReportRepo(),
		screen,
		screener,
	)
	return svc, screen, screener
}

func TestClassifyOperation_ByCode(t *testing.T) {
	svc, _ := newSvc(t)
	c, err := svc.ClassifyOperation(context.Background(), uuid.New(), uuid.New(), "10001")
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if c.Code() != "10001" || c.Nature() != domain.NatureIngresso {
		t.Errorf("got %s/%s", c.Code(), c.Nature())
	}
}

// The expectation here was "20011", from the 20-code seed in pkg/bacen that
// predates the generated 46-code Circ. 3.690 catalogue. 20011 is not in that
// catalogue — it is one of 13 seed entries with no official counterpart — and
// Classifier.ByCode resolves against the generated catalogue first. The royalty
// codes BACEN actually defines are 60100 (direitos autorais), 60200 (royalties
// tecnologia) and 60201 (royalties marca / patente); pkg/bacen's own golden set
// already pins "Pagamento de royalty de patente" to 60200.
func TestClassifyOperation_FreeText(t *testing.T) {
	svc, _ := newSvc(t)
	c, err := svc.ClassifyOperation(context.Background(), uuid.New(), uuid.New(), "Pagamento de royalties")
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if c.Code() != "60200" {
		t.Errorf("code: got %s want 60200", c.Code())
	}
	// A royalty payment leaves the country, so it must classify as a remessa.
	// Getting the direction wrong misreports the operation to BACEN.
	if c.Nature() != domain.NatureRemessa {
		t.Errorf("nature: got %s want %s", c.Nature(), domain.NatureRemessa)
	}
}

// The free-text classifier must distinguish the three royalty families rather
// than collapsing them onto the generic fallback.
func TestClassifyOperation_FreeText_RoyaltyFamilies(t *testing.T) {
	tests := []struct {
		hint string
		want string
	}{
		{"Royalties de obra literária", "60100"},
		{"Pagamento de direitos autorais", "60100"},
		{"Royalty de marca", "60201"},
		{"Royalty de patente", "60200"},
		{"Licença de uso de tecnologia", "60200"},
		{"Pagamento de royalties", "60200"}, // generic fallback
	}

	for _, tt := range tests {
		t.Run(tt.hint, func(t *testing.T) {
			svc, _ := newSvc(t)
			c, err := svc.ClassifyOperation(context.Background(), uuid.New(), uuid.New(), tt.hint)
			if err != nil {
				t.Fatalf("Classify(%q): %v", tt.hint, err)
			}
			if c.Code() != tt.want {
				t.Errorf("Classify(%q) = %s, want %s", tt.hint, c.Code(), tt.want)
			}
		})
	}
}

func TestClassifyOperation_UnknownHint(t *testing.T) {
	svc, _ := newSvc(t)
	_, err := svc.ClassifyOperation(context.Background(), uuid.New(), uuid.New(), "xyzzy")
	if !errors.Is(err, bacen.ErrUnknown) {
		t.Fatalf("want ErrUnknown, got %v", err)
	}
}

func TestComputeIOF_Golden(t *testing.T) {
	svc, _ := newSvc(t)
	iof, err := svc.ComputeIOF(context.Background(), uuid.New(), uuid.New(),
		"DEFAULT", decimal.NewFromInt(10000), "USD")
	if err != nil {
		t.Fatalf("ComputeIOF: %v", err)
	}
	if !iof.IOFAmount().Equal(dec("38.00")) {
		t.Errorf("amount: got %s want 38.00", iof.IOFAmount())
	}
}

func TestComputeIOF_BadOpType_Propagates(t *testing.T) {
	svc, _ := newSvc(t)
	_, err := svc.ComputeIOF(context.Background(), uuid.New(), uuid.New(),
		"BOGUS", decimal.NewFromInt(100), "USD")
	if !errors.Is(err, bacen.ErrUnknown) {
		t.Fatalf("want bacen.ErrUnknown, got %v", err)
	}
}

func TestSubmitBACENReport_PersistsPending(t *testing.T) {
	svc, _ := newSvc(t)
	r, err := svc.SubmitBACENReport(context.Background(), domain.NewBACENReportInput{
		TenantID:      uuid.New(),
		ReportType:    domain.ReportSISBACEN,
		ReferenceDate: time.Now().UTC(),
		PayloadHash:   "abc123",
	})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if r.Status() != domain.StatusPending {
		t.Errorf("status: %s", r.Status())
	}
}

// The hits now come from the provider, so each case drives them through the
// screener rather than through the caller's input.
func TestScreenCounterparty_RiskLevelDerivation(t *testing.T) {
	tests := []struct {
		hits      []string
		wantLevel domain.RiskLevel
	}{
		{nil, domain.RiskLow},
		{[]string{"OFAC:SDN:demo"}, domain.RiskMedium},
		{[]string{"OFAC:a", "UN:b", "EU:c"}, domain.RiskHigh},
	}
	for _, tc := range tests {
		t.Run(string(tc.wantLevel), func(t *testing.T) {
			svc, repo, _ := newSvcWith(t, &fakeScreener{hits: tc.hits})
			res, err := svc.ScreenCounterparty(context.Background(), domain.NewScreeningInput{
				TenantID:        uuid.New(),
				CounterpartyBIC: "CHASUS33",
			})
			if err != nil {
				t.Fatalf("Screen: %v", err)
			}
			if res.RiskLevel() != tc.wantLevel {
				t.Errorf("level: got %s want %s", res.RiskLevel(), tc.wantLevel)
			}
			if len(repo.Saved) != 1 {
				t.Errorf("repo Saved: got %d want 1", len(repo.Saved))
			}
		})
	}
}

// Without a provider wired, screening must fail rather than clear the
// counterparty. A screening that did not happen is not a pass.
func TestScreenCounterparty_FailsClosedWithoutProvider(t *testing.T) {
	svc, repo, _ := newSvcWith(t, application.UnavailableScreener{})

	_, err := svc.ScreenCounterparty(context.Background(), domain.NewScreeningInput{
		TenantID:        uuid.New(),
		CounterpartyBIC: "CHASUS33",
	})

	if !errors.Is(err, application.ErrScreeningUnavailable) {
		t.Fatalf("error = %v, want ErrScreeningUnavailable", err)
	}
	if len(repo.Saved) != 0 {
		t.Errorf("persisted %d results — an unscreened counterparty must not be recorded", len(repo.Saved))
	}
}

// A nil screener is the fail-closed default, not "screening disabled".
func TestNewService_NilScreenerFailsClosed(t *testing.T) {
	svc, _, _ := newSvcWith(t, nil)

	_, err := svc.ScreenCounterparty(context.Background(), domain.NewScreeningInput{
		TenantID:        uuid.New(),
		CounterpartyBIC: "CHASUS33",
	})
	if !errors.Is(err, application.ErrScreeningUnavailable) {
		t.Fatalf("error = %v, want ErrScreeningUnavailable", err)
	}
}

// Hits supplied by the caller must be ignored: only the provider decides.
// Otherwise a client could declare itself clear.
func TestScreenCounterparty_IgnoresCallerSuppliedHits(t *testing.T) {
	svc, _, _ := newSvcWith(t, &fakeScreener{hits: []string{"OFAC:a", "UN:b", "EU:c"}})

	res, err := svc.ScreenCounterparty(context.Background(), domain.NewScreeningInput{
		TenantID:        uuid.New(),
		CounterpartyBIC: "CHASUS33",
		Hits:            nil, // caller claims clear
	})
	if err != nil {
		t.Fatalf("Screen: %v", err)
	}
	if res.RiskLevel() != domain.RiskHigh {
		t.Errorf("level = %s, want %s — the provider verdict must win", res.RiskLevel(), domain.RiskHigh)
	}
	if len(res.Hits()) != 3 {
		t.Errorf("hits = %d, want 3", len(res.Hits()))
	}
}

// A provider failure must surface, not be swallowed into a clear verdict.
func TestScreenCounterparty_ProviderErrorSurfaces(t *testing.T) {
	boom := errors.New("watchlist feed timeout")
	svc, repo, _ := newSvcWith(t, &fakeScreener{err: boom})

	_, err := svc.ScreenCounterparty(context.Background(), domain.NewScreeningInput{
		TenantID:        uuid.New(),
		CounterpartyBIC: "CHASUS33",
	})
	if !errors.Is(err, boom) {
		t.Fatalf("error = %v, want it to wrap %v", err, boom)
	}
	if len(repo.Saved) != 0 {
		t.Errorf("persisted %d results after a provider failure, want 0", len(repo.Saved))
	}
}

// The provider must be asked about the counterparty actually requested.
func TestScreenCounterparty_PassesCounterpartyToProvider(t *testing.T) {
	f := &fakeScreener{}
	svc, _, _ := newSvcWith(t, f)

	const (
		bic = "DEUTDEFF"
		lei = "7LTWFZYICNSX8D621K86"
	)
	if _, err := svc.ScreenCounterparty(context.Background(), domain.NewScreeningInput{
		TenantID:        uuid.New(),
		CounterpartyBIC: bic,
		LEI:             lei,
	}); err != nil {
		t.Fatalf("Screen: %v", err)
	}

	if len(f.calls) != 1 {
		t.Fatalf("provider called %d times, want 1", len(f.calls))
	}
	if f.calls[0].bic != bic || f.calls[0].lei != lei {
		t.Errorf("provider asked about %q/%q, want %q/%q",
			f.calls[0].bic, f.calls[0].lei, bic, lei)
	}
}

func TestClassify_BadIDs(t *testing.T) {
	svc, _ := newSvc(t)
	if _, err := svc.ClassifyOperation(context.Background(), uuid.Nil, uuid.New(), "10001"); !errors.Is(err, application.ErrInvalidInput) {
		t.Errorf("nil tenant: %v", err)
	}
}
