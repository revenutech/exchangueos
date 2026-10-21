package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/quote/domain"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func validQuoteInput(t *testing.T) domain.NewQuoteInput {
	t.Helper()
	now := time.Now().UTC()
	return domain.NewQuoteInput{
		TenantID:     uuid.New(),
		RequesterBIC: "chasus33",
		ProviderBIC:  "DEUTDEFF",
		Side:         domain.SideBuy,
		BaseCCY:      "eur",
		QuoteCCY:     "usd",
		Notional:     dec("1000000"),
		NotionalCCY:  "EUR",
		Bid:          dec("1.0798"),
		Ask:          dec("1.0802"),
		ValidFrom:    now,
		ValidTo:      now.Add(10 * time.Second),
		Venue:        "INTERNAL",
	}
}

func TestQuote_Valid(t *testing.T) {
	q, err := domain.NewQuote(validQuoteInput(t))
	if err != nil {
		t.Fatalf("NewQuote: %v", err)
	}
	if q.BaseCCY() != "EUR" || q.QuoteCCY() != "USD" {
		t.Errorf("ccy normalised: base=%s quote=%s", q.BaseCCY(), q.QuoteCCY())
	}
	if !q.Mid().Equal(dec("1.0800")) {
		t.Errorf("mid: got %s want 1.0800", q.Mid())
	}
	if got := q.Version(); got != 1 {
		t.Errorf("version: got %d want 1", got)
	}
	if got := len(q.PendingEvents()); got != 1 {
		t.Errorf("pending events: got %d want 1", got)
	}
}

func TestQuote_BidExceedsAsk_Rejected(t *testing.T) {
	in := validQuoteInput(t)
	in.Bid = dec("1.10")
	in.Ask = dec("1.09")
	_, err := domain.NewQuote(in)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestQuote_NotionalCCYMustMatchPair(t *testing.T) {
	in := validQuoteInput(t)
	in.NotionalCCY = "GBP"
	_, err := domain.NewQuote(in)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestQuote_AcceptWithinWindow(t *testing.T) {
	q, _ := domain.NewQuote(validQuoteInput(t))
	if err := q.Accept(time.Now().UTC(), "trader-a"); err != nil {
		t.Fatalf("Accept: %v", err)
	}
	events := q.PendingEvents()
	if len(events) != 2 {
		t.Fatalf("events: got %d want 2 (created + accepted)", len(events))
	}
	if events[1].EventName() != "quote.accepted.v1" {
		t.Fatalf("event[1]: got %s want quote.accepted.v1", events[1].EventName())
	}
}

func TestQuote_AcceptExpired(t *testing.T) {
	in := validQuoteInput(t)
	in.ValidFrom = time.Now().UTC().Add(-1 * time.Hour)
	in.ValidTo = time.Now().UTC().Add(-1 * time.Minute)
	q, _ := domain.NewQuote(in)
	if err := q.Accept(time.Now().UTC(), "trader-a"); !errors.Is(err, domain.ErrQuoteExpired) {
		t.Fatalf("want ErrQuoteExpired, got %v", err)
	}
}

func TestRFQ_Lifecycle_HappyPath(t *testing.T) {
	r, err := domain.NewRFQ(domain.NewRFQInput{
		TenantID:  uuid.New(),
		Requester: "trader-a",
		BaseCCY:   "EUR",
		QuoteCCY:  "USD",
	})
	if err != nil {
		t.Fatalf("NewRFQ: %v", err)
	}
	if r.Status() != domain.RFQRequested {
		t.Fatalf("initial: got %s", r.Status())
	}
	qid := uuid.New()
	if err := r.AttachQuote(qid); err != nil {
		t.Fatalf("AttachQuote: %v", err)
	}
	if r.Status() != domain.RFQQuoted {
		t.Fatalf("after attach: got %s", r.Status())
	}
	if err := r.Accept(qid, "trader-a"); err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if r.Status() != domain.RFQAccepted {
		t.Fatalf("after accept: got %s", r.Status())
	}
	events := r.PendingEvents()
	wantNames := []string{"rfq.requested.v1", "rfq.quoted.v1", "rfq.accepted.v1"}
	if len(events) != len(wantNames) {
		t.Fatalf("events: got %d want %d", len(events), len(wantNames))
	}
	for i, n := range wantNames {
		if events[i].EventName() != n {
			t.Errorf("event[%d]: got %s want %s", i, events[i].EventName(), n)
		}
	}
}

func TestRFQ_AcceptUnknownQuote_Rejected(t *testing.T) {
	r, _ := domain.NewRFQ(domain.NewRFQInput{
		TenantID:  uuid.New(),
		Requester: "trader-a",
		BaseCCY:   "EUR",
		QuoteCCY:  "USD",
	})
	_ = r.AttachQuote(uuid.New())
	err := r.Accept(uuid.New(), "trader-a") // different quote
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestRFQ_RejectRequiresReason(t *testing.T) {
	r, _ := domain.NewRFQ(domain.NewRFQInput{
		TenantID:  uuid.New(),
		Requester: "trader-a",
		BaseCCY:   "EUR",
		QuoteCCY:  "USD",
	})
	if err := r.Reject(""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestRFQ_ExpireFromQuoted(t *testing.T) {
	r, _ := domain.NewRFQ(domain.NewRFQInput{
		TenantID:  uuid.New(),
		Requester: "trader-a",
		BaseCCY:   "EUR",
		QuoteCCY:  "USD",
	})
	_ = r.AttachQuote(uuid.New())
	if err := r.Expire(); err != nil {
		t.Fatalf("Expire: %v", err)
	}
	if r.Status() != domain.RFQExpired {
		t.Fatalf("after expire: %s", r.Status())
	}
}

// --- Counterparties, side and venue ---
//
// Before the Quote carried these, the quote→trade adapter substituted the
// literals "DEUTDEFF"/"CHASUS33"/"CLS" and priced off the mid, so every
// auto-booked trade named the wrong counterparties and gave away half the
// spread. These tests pin the rules that replaced that.

func TestQuote_DealRate_BuyLiftsTheOffer(t *testing.T) {
	in := validQuoteInput(t)
	in.Side = domain.SideBuy
	q, err := domain.NewQuote(in)
	if err != nil {
		t.Fatalf("NewQuote: %v", err)
	}

	if !q.DealRate().Equal(q.Ask()) {
		t.Errorf("DealRate = %s, want the ask %s", q.DealRate(), q.Ask())
	}
	if q.DealRate().Equal(q.Mid()) {
		t.Error("DealRate equals the mid — the spread is being given away")
	}
}

func TestQuote_DealRate_SellHitsTheBid(t *testing.T) {
	in := validQuoteInput(t)
	in.Side = domain.SideSell
	q, err := domain.NewQuote(in)
	if err != nil {
		t.Fatalf("NewQuote: %v", err)
	}

	if !q.DealRate().Equal(q.Bid()) {
		t.Errorf("DealRate = %s, want the bid %s", q.DealRate(), q.Bid())
	}
}

// The trade is framed bought=base / sold=quote, so the legs swap with the side.
func TestQuote_LegsFollowTheSide(t *testing.T) {
	tests := []struct {
		side              domain.Side
		wantBuy, wantSell string
	}{
		{domain.SideBuy, "CHASUS33", "DEUTDEFF"},
		{domain.SideSell, "DEUTDEFF", "CHASUS33"},
	}

	for _, tt := range tests {
		t.Run(string(tt.side), func(t *testing.T) {
			in := validQuoteInput(t)
			in.RequesterBIC = "CHASUS33"
			in.ProviderBIC = "DEUTDEFF"
			in.Side = tt.side
			q, err := domain.NewQuote(in)
			if err != nil {
				t.Fatalf("NewQuote: %v", err)
			}

			if q.BuyerBIC() != tt.wantBuy {
				t.Errorf("BuyerBIC = %s, want %s", q.BuyerBIC(), tt.wantBuy)
			}
			if q.SellerBIC() != tt.wantSell {
				t.Errorf("SellerBIC = %s, want %s", q.SellerBIC(), tt.wantSell)
			}
			if q.BuyerBIC() == q.SellerBIC() {
				t.Error("buyer and seller are the same party")
			}
		})
	}
}

func TestQuote_NormalisesBICsAndVenue(t *testing.T) {
	in := validQuoteInput(t)
	in.RequesterBIC = "  chasus33  "
	in.ProviderBIC = "deutdeff"
	in.Venue = " internal "
	q, err := domain.NewQuote(in)
	if err != nil {
		t.Fatalf("NewQuote: %v", err)
	}

	if q.RequesterBIC() != "CHASUS33" {
		t.Errorf("RequesterBIC = %q, want CHASUS33", q.RequesterBIC())
	}
	if q.ProviderBIC() != "DEUTDEFF" {
		t.Errorf("ProviderBIC = %q, want DEUTDEFF", q.ProviderBIC())
	}
	if q.Venue() != "INTERNAL" {
		t.Errorf("Venue = %q, want INTERNAL", q.Venue())
	}
}

func TestQuote_RejectsMissingOrInvalidParties(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*domain.NewQuoteInput)
	}{
		{"no requester", func(in *domain.NewQuoteInput) { in.RequesterBIC = "" }},
		{"no provider", func(in *domain.NewQuoteInput) { in.ProviderBIC = "" }},
		{"requester too short", func(in *domain.NewQuoteInput) { in.RequesterBIC = "CHAS" }},
		{"requester 9 chars", func(in *domain.NewQuoteInput) { in.RequesterBIC = "CHASUS333" }},
		{"requester non-alphanumeric", func(in *domain.NewQuoteInput) { in.RequesterBIC = "CHAS-S33" }},
		{"same party both sides", func(in *domain.NewQuoteInput) {
			in.RequesterBIC = "CHASUS33"
			in.ProviderBIC = "chasus33"
		}},
		{"no side", func(in *domain.NewQuoteInput) { in.Side = "" }},
		{"unknown side", func(in *domain.NewQuoteInput) { in.Side = domain.Side("LONG") }},
		{"no venue", func(in *domain.NewQuoteInput) { in.Venue = "" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validQuoteInput(t)
			tt.mutate(&in)
			if _, err := domain.NewQuote(in); !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("NewQuote() error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestParseSide(t *testing.T) {
	for _, in := range []string{"BUY", "buy", " Buy "} {
		if got, err := domain.ParseSide(in); err != nil || got != domain.SideBuy {
			t.Errorf("ParseSide(%q) = %q, %v; want BUY", in, got, err)
		}
	}
	for _, in := range []string{"SELL", "sell"} {
		if got, err := domain.ParseSide(in); err != nil || got != domain.SideSell {
			t.Errorf("ParseSide(%q) = %q, %v; want SELL", in, got, err)
		}
	}
	for _, in := range []string{"", "LONG", "B"} {
		if _, err := domain.ParseSide(in); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("ParseSide(%q) error = %v, want ErrInvalidInput", in, err)
		}
	}
}

// The created event must carry the parties and side, otherwise a consumer
// rebuilding from the stream cannot tell who dealt in which direction.
func TestQuote_CreatedEventCarriesPartiesAndSide(t *testing.T) {
	in := validQuoteInput(t)
	in.Side = domain.SideSell
	q, err := domain.NewQuote(in)
	if err != nil {
		t.Fatalf("NewQuote: %v", err)
	}

	events := q.PendingEvents()
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	ev, ok := events[0].(domain.EventQuoteCreated)
	if !ok {
		t.Fatalf("event type = %T, want EventQuoteCreated", events[0])
	}
	if ev.RequesterBIC != q.RequesterBIC() || ev.ProviderBIC != q.ProviderBIC() {
		t.Errorf("event parties = %s/%s, want %s/%s",
			ev.RequesterBIC, ev.ProviderBIC, q.RequesterBIC(), q.ProviderBIC())
	}
	if ev.Side != string(domain.SideSell) {
		t.Errorf("event side = %q, want SELL", ev.Side)
	}
}
