package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	qdomain "github.com/revenutech/exchangeos/modules/quote/domain"
	"github.com/revenutech/exchangeos/modules/trade/domain"
)

// QuoteAcceptedHandler reacts to a quote.accepted.v1 event by booking the
// corresponding FXTrade.
//
// The handler needs the original Quote aggregate to reconstruct trade economics;
// it's looked up via the supplied QuoteLookup function (in production, the
// quote/application service; in tests, a stub).
type QuoteAcceptedHandler struct {
	Trades      *Service
	QuoteLookup func(ctx context.Context, quoteID uuid.UUID) (AcceptedQuoteView, error)
}

// AcceptedQuoteView is the read-model the handler needs from the quote BC.
// Decoupled from the full Quote aggregate to avoid coupling to its internals.
type AcceptedQuoteView struct {
	TenantID    uuid.UUID
	BuyerBIC    string
	SellerBIC   string
	BaseCCY     string
	QuoteCCY    string
	NotionalCCY string
	Notional    decimal.Decimal
	DealRate    decimal.Decimal // the side-adjusted deal rate (ask on a buy, bid on a sell)
	Venue       string          // "CLS" | "BILATERAL" | "CFETS" (raw)
	AcceptedAt  time.Time
}

// Handle is the eventbus.Handler entrypoint. Accepts any DomainEvent — only
// quote.accepted.v1 produces a trade; everything else is silently ignored.
func (h *QuoteAcceptedHandler) Handle(ctx context.Context, e qdomain.DomainEvent) error {
	acc, ok := e.(qdomain.EventQuoteAccepted)
	if !ok {
		return nil
	}
	if h.QuoteLookup == nil {
		return fmt.Errorf("quote-accepted handler: QuoteLookup nil")
	}
	view, err := h.QuoteLookup(ctx, acc.QuoteID)
	if err != nil {
		return fmt.Errorf("quote-accepted handler: lookup: %w", err)
	}

	soldAmount := view.Notional.Mul(view.DealRate)
	td := acc.AcceptedAt
	if td.IsZero() {
		td = time.Now().UTC()
	}
	_, err = h.Trades.BookTrade(ctx, BookTradeRequest{
		TenantID:       view.TenantID,
		ExternalRef:    "from-quote:" + acc.QuoteID.String(),
		Type:           domain.TradeTypeSpot,
		Venue:          venueFromString(view.Venue),
		BuyerBIC:       view.BuyerBIC,
		SellerBIC:      view.SellerBIC,
		BoughtCurrency: view.BaseCCY,
		BoughtAmount:   view.Notional,
		SoldCurrency:   view.QuoteCCY,
		SoldAmount:     soldAmount,
		DealRate:       view.DealRate,
		TradeDate:      td,
		ValueDate:      td.AddDate(0, 0, 2), // T+2 spot default
	})
	return err
}

func venueFromString(s string) domain.SettlementVenue {
	switch s {
	case "CLS", "CLSBUS33":
		return domain.VenueCLS
	case "CFETS":
		return domain.VenueCFETS
	default:
		return domain.VenueBilateral
	}
}
