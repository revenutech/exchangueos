package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/quote/application"
	"github.com/revenutech/exchangeos/modules/quote/domain"
	"github.com/revenutech/exchangeos/modules/quote/infrastructure/memory"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// stubEngine returns a fixed mid + half-spread.
type stubEngine struct {
	mid, half decimal.Decimal
	err       error
	calls     int
}

func (e *stubEngine) GetMidRate(_ context.Context, _, _ string) (decimal.Decimal, decimal.Decimal, error) {
	e.calls++
	if e.err != nil {
		return decimal.Zero, decimal.Zero, e.err
	}
	return e.mid, e.half, nil
}

func newSvc(t *testing.T, eng application.PricingEngine) (*application.Service, *memory.QuoteRepo, *memory.RFQRepo, *memory.NoopPublisher) {
	t.Helper()
	qr := memory.NewQuoteRepo()
	rr := memory.NewRFQRepo()
	pub := memory.NewNoopPublisher()
	svc := application.NewService(qr, rr, eng, pub, application.Options{DefaultQuoteTTL: 5 * time.Second})
	return svc, qr, rr, pub
}

func TestGetQuote_PricesAndPersists(t *testing.T) {
	eng := &stubEngine{mid: dec("1.0800"), half: dec("0.0002")}
	svc, qr, _, pub := newSvc(t, eng)
	ctx := context.Background()

	q, err := svc.GetQuote(ctx, application.GetQuoteRequest{
		TenantID:     uuid.New(),
		RequesterBIC: "CHASUS33",
		ProviderBIC:  "DEUTDEFF",
		Side:         domain.SideBuy,
		BaseCCY:      "eur",
		QuoteCCY:     "usd",
		Notional:     decimal.NewFromInt(1_000_000),
		NotionalCCY:  "EUR",
		Venue:        "INTERNAL",
	})
	if err != nil {
		t.Fatalf("GetQuote: %v", err)
	}
	if !q.Bid().Equal(dec("1.0798")) || !q.Ask().Equal(dec("1.0802")) {
		t.Fatalf("bid/ask: %s / %s want 1.0798 / 1.0802", q.Bid(), q.Ask())
	}
	if q.BaseCCY() != "EUR" {
		t.Errorf("ccy normalisation: got %s", q.BaseCCY())
	}
	if got, _ := qr.Get(ctx, q.ID()); got == nil {
		t.Fatal("not persisted")
	}
	if len(pub.Published) != 1 || pub.Published[0].EventName() != "quote.created.v1" {
		t.Fatalf("publication: got %d events first=%s", len(pub.Published),
			func() string {
				if len(pub.Published) == 0 {
					return "(none)"
				}
				return pub.Published[0].EventName()
			}())
	}
}

func TestGetQuote_PricingError_Propagated(t *testing.T) {
	eng := &stubEngine{err: errors.New("no liquidity")}
	svc, _, _, _ := newSvc(t, eng)
	_, err := svc.GetQuote(context.Background(), application.GetQuoteRequest{
		TenantID:     uuid.New(),
		RequesterBIC: "CHASUS33",
		ProviderBIC:  "DEUTDEFF",
		Side:         domain.SideBuy,
		BaseCCY:      "EUR",
		QuoteCCY:     "USD",
		Notional:     decimal.NewFromInt(1000),
		NotionalCCY:  "EUR",
	})
	if err == nil || err.Error() != "no liquidity" {
		t.Fatalf("expected propagated pricing error, got %v", err)
	}
}

func TestGetQuote_RejectsBadInput(t *testing.T) {
	eng := &stubEngine{mid: dec("1.0800"), half: dec("0.0002")}
	svc, _, _, _ := newSvc(t, eng)
	cases := []struct {
		name string
		req  application.GetQuoteRequest
	}{
		{"nil tenant", application.GetQuoteRequest{BaseCCY: "EUR", QuoteCCY: "USD", Notional: decimal.NewFromInt(1), NotionalCCY: "EUR"}},
		{"zero notional", application.GetQuoteRequest{TenantID: uuid.New(), BaseCCY: "EUR", QuoteCCY: "USD", Notional: decimal.Zero, NotionalCCY: "EUR"}},
		{"bad ccy length", application.GetQuoteRequest{TenantID: uuid.New(), BaseCCY: "EURO", QuoteCCY: "USD", Notional: decimal.NewFromInt(1), NotionalCCY: "EUR"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.GetQuote(context.Background(), tc.req)
			if !errors.Is(err, application.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestAcceptQuote_Lifecycle(t *testing.T) {
	eng := &stubEngine{mid: dec("1.0800"), half: dec("0.0002")}
	svc, _, _, pub := newSvc(t, eng)
	ctx := context.Background()
	tenant := uuid.New()

	q, err := svc.GetQuote(ctx, application.GetQuoteRequest{
		TenantID:     tenant,
		RequesterBIC: "CHASUS33",
		ProviderBIC:  "DEUTDEFF",
		Side:         domain.SideBuy,
		BaseCCY:      "EUR",
		QuoteCCY:     "USD",
		Notional:     decimal.NewFromInt(1_000_000),
		NotionalCCY:  "EUR",
		Venue:        "INTERNAL",
		TTL:          5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("setup GetQuote: %v", err)
	}
	pub.Published = nil // clear

	out, err := svc.AcceptQuote(ctx, application.AcceptQuoteRequest{
		QuoteID: q.ID(), Actor: "trader-a",
	})
	if err != nil {
		t.Fatalf("AcceptQuote: %v", err)
	}
	if out.Version() != 2 {
		t.Errorf("version: got %d want 2", out.Version())
	}
	if len(pub.Published) != 1 || pub.Published[0].EventName() != "quote.accepted.v1" {
		t.Fatalf("publication: got %d first=%s", len(pub.Published),
			func() string {
				if len(pub.Published) == 0 {
					return "(none)"
				}
				return pub.Published[0].EventName()
			}())
	}
}

func TestAcceptQuote_Expired_Propagates(t *testing.T) {
	eng := &stubEngine{mid: dec("1.0800"), half: dec("0.0002")}
	svc, _, _, _ := newSvc(t, eng)
	ctx := context.Background()
	// The error is checked rather than discarded: swallowing it turned a
	// construction failure into a nil-pointer panic on q.ID() below.
	q, err := svc.GetQuote(ctx, application.GetQuoteRequest{
		TenantID: uuid.New(), RequesterBIC: "CHASUS33", ProviderBIC: "DEUTDEFF",
		Side: domain.SideBuy, BaseCCY: "EUR", QuoteCCY: "USD",
		Notional: decimal.NewFromInt(1), NotionalCCY: "EUR",
		Venue: "INTERNAL", TTL: 1 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("setup GetQuote: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	_, err = svc.AcceptQuote(ctx, application.AcceptQuoteRequest{QuoteID: q.ID(), Actor: "trader-a"})
	if !errors.Is(err, domain.ErrQuoteExpired) {
		t.Fatalf("want ErrQuoteExpired, got %v", err)
	}
}

func TestAcceptQuote_NotFound(t *testing.T) {
	eng := &stubEngine{mid: dec("1.0800"), half: dec("0.0002")}
	svc, _, _, _ := newSvc(t, eng)
	_, err := svc.AcceptQuote(context.Background(), application.AcceptQuoteRequest{
		QuoteID: uuid.New(), Actor: "trader-a",
	})
	if !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestRFQ_FullFlow(t *testing.T) {
	eng := &stubEngine{mid: dec("1.0800"), half: dec("0.0002")}
	svc, _, _, pub := newSvc(t, eng)
	ctx := context.Background()
	tenant := uuid.New()

	// Create RFQ
	rfq, err := svc.CreateRFQ(ctx, application.CreateRFQRequest{
		TenantID:  tenant,
		Requester: "trader-a",
		BaseCCY:   "EUR",
		QuoteCCY:  "USD",
	})
	if err != nil {
		t.Fatalf("CreateRFQ: %v", err)
	}

	// Get a quote and attach it
	q, err := svc.GetQuote(ctx, application.GetQuoteRequest{
		TenantID: tenant, RequesterBIC: "CHASUS33", ProviderBIC: "DEUTDEFF",
		Side: domain.SideBuy, BaseCCY: "EUR", QuoteCCY: "USD",
		Notional: decimal.NewFromInt(1_000_000), NotionalCCY: "EUR",
		Venue: "INTERNAL", TTL: 5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("setup GetQuote: %v", err)
	}
	rfq2, err := svc.AttachQuoteToRFQ(ctx, application.AttachQuoteToRFQRequest{
		RFQID: rfq.ID(), QuoteID: q.ID(),
	})
	if err != nil {
		t.Fatalf("AttachQuoteToRFQ: %v", err)
	}
	if rfq2.Status() != domain.RFQQuoted {
		t.Errorf("after attach: %s", rfq2.Status())
	}

	// Accept
	rfq3, err := svc.AcceptRFQ(ctx, application.AcceptRFQRequest{
		RFQID: rfq.ID(), QuoteID: q.ID(), Actor: "trader-a",
	})
	if err != nil {
		t.Fatalf("AcceptRFQ: %v", err)
	}
	if rfq3.Status() != domain.RFQAccepted {
		t.Errorf("after accept: %s", rfq3.Status())
	}

	// Publisher should have seen: rfq.requested, quote.created, rfq.quoted, rfq.accepted
	wantNames := []string{
		"rfq.requested.v1",
		"quote.created.v1",
		"rfq.quoted.v1",
		"rfq.accepted.v1",
	}
	if len(pub.Published) != len(wantNames) {
		t.Fatalf("publications: got %d want %d", len(pub.Published), len(wantNames))
	}
	for i, n := range wantNames {
		if pub.Published[i].EventName() != n {
			t.Errorf("event[%d]: got %s want %s", i, pub.Published[i].EventName(), n)
		}
	}
}
