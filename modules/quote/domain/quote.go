package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Side — the direction the quote requester takes on the BASE currency.
//
// It decides both which price applies and which counterparty ends up on each
// leg of the resulting trade, so a quote without a side cannot be turned into a
// bookable trade.
type Side string

const (
	// SideBuy — the requester buys the base currency and pays the ask.
	SideBuy Side = "BUY"
	// SideSell — the requester sells the base currency and receives the bid.
	SideSell Side = "SELL"
)

// ParseSide normalises a wire value into a Side.
func ParseSide(s string) (Side, error) {
	switch Side(strings.ToUpper(strings.TrimSpace(s))) {
	case SideBuy:
		return SideBuy, nil
	case SideSell:
		return SideSell, nil
	case "":
		return "", fmt.Errorf("%w: side required (BUY or SELL)", ErrInvalidInput)
	default:
		return "", fmt.Errorf("%w: unknown side %q", ErrInvalidInput, s)
	}
}

// Quote — a streamable bid/ask price with a validity window.
type Quote struct {
	id           uuid.UUID
	tenantID     uuid.UUID
	requesterBIC string
	providerBIC  string
	side         Side
	baseCCY      string
	quoteCCY     string
	notional     decimal.Decimal
	notionalCCY  string
	bid          decimal.Decimal
	ask          decimal.Decimal
	validFrom    time.Time
	validTo      time.Time
	venue        string
	version      int
	events       []DomainEvent
}

// NewQuoteInput parameterises construction.
type NewQuoteInput struct {
	TenantID uuid.UUID
	// RequesterBIC — the party the price is quoted TO (the client).
	RequesterBIC string
	// ProviderBIC — the party quoting the price (the dealer).
	ProviderBIC string
	// Side — the requester's direction on the base currency.
	Side        Side
	BaseCCY     string
	QuoteCCY    string
	Notional    decimal.Decimal
	NotionalCCY string // base or quote
	Bid         decimal.Decimal
	Ask         decimal.Decimal
	ValidFrom   time.Time
	ValidTo     time.Time
	Venue       string
}

// NewQuote constructs and validates a Quote.
func NewQuote(in NewQuoteInput) (*Quote, error) {
	if in.TenantID == uuid.Nil {
		return nil, fmt.Errorf("%w: tenant_id required", ErrInvalidInput)
	}
	requester, err := normaliseBIC(in.RequesterBIC, "requester_bic")
	if err != nil {
		return nil, err
	}
	provider, err := normaliseBIC(in.ProviderBIC, "provider_bic")
	if err != nil {
		return nil, err
	}
	if requester == provider {
		return nil, fmt.Errorf("%w: requester_bic and provider_bic must differ", ErrInvalidInput)
	}
	side, err := ParseSide(string(in.Side))
	if err != nil {
		return nil, err
	}
	venue := strings.ToUpper(strings.TrimSpace(in.Venue))
	if venue == "" {
		return nil, fmt.Errorf("%w: venue required", ErrInvalidInput)
	}
	if err := validatePair(in.BaseCCY, in.QuoteCCY); err != nil {
		return nil, err
	}
	if !in.Notional.IsPositive() {
		return nil, fmt.Errorf("%w: notional must be > 0", ErrInvalidInput)
	}
	notCCY := strings.ToUpper(strings.TrimSpace(in.NotionalCCY))
	if notCCY != strings.ToUpper(in.BaseCCY) && notCCY != strings.ToUpper(in.QuoteCCY) {
		return nil, fmt.Errorf("%w: notional_ccy must equal base or quote ccy", ErrInvalidInput)
	}
	if !in.Bid.IsPositive() {
		return nil, fmt.Errorf("%w: bid must be > 0", ErrInvalidInput)
	}
	if !in.Ask.IsPositive() {
		return nil, fmt.Errorf("%w: ask must be > 0", ErrInvalidInput)
	}
	if in.Bid.GreaterThan(in.Ask) {
		return nil, fmt.Errorf("%w: bid (%s) must be <= ask (%s)", ErrInvalidInput, in.Bid, in.Ask)
	}
	if in.ValidFrom.IsZero() || in.ValidTo.IsZero() {
		return nil, fmt.Errorf("%w: valid_from / valid_to required", ErrInvalidInput)
	}
	if !in.ValidTo.After(in.ValidFrom) {
		return nil, fmt.Errorf("%w: valid_to must be > valid_from", ErrInvalidInput)
	}
	id := uuid.New()
	q := &Quote{
		id:           id,
		tenantID:     in.TenantID,
		requesterBIC: requester,
		providerBIC:  provider,
		side:         side,
		baseCCY:      strings.ToUpper(in.BaseCCY),
		quoteCCY:     strings.ToUpper(in.QuoteCCY),
		notional:     in.Notional,
		notionalCCY:  notCCY,
		bid:          in.Bid,
		ask:          in.Ask,
		validFrom:    in.ValidFrom.UTC(),
		validTo:      in.ValidTo.UTC(),
		venue:        venue,
		version:      1,
	}
	q.recordEvent(EventQuoteCreated{
		QuoteID:      id,
		TenantID:     in.TenantID,
		RequesterBIC: q.requesterBIC,
		ProviderBIC:  q.providerBIC,
		Side:         string(q.side),
		BaseCCY:      q.baseCCY,
		QuoteCCY:     q.quoteCCY,
		Bid:          q.bid,
		Ask:          q.ask,
		OccurredAt:   time.Now().UTC(),
	})
	return q, nil
}

func (q *Quote) ID() uuid.UUID             { return q.id }
func (q *Quote) TenantID() uuid.UUID       { return q.tenantID }
func (q *Quote) Bid() decimal.Decimal      { return q.bid }
func (q *Quote) Ask() decimal.Decimal      { return q.ask }
func (q *Quote) Mid() decimal.Decimal      { return q.bid.Add(q.ask).Div(decimal.NewFromInt(2)) }
func (q *Quote) BaseCCY() string           { return q.baseCCY }
func (q *Quote) QuoteCCY() string          { return q.quoteCCY }
func (q *Quote) Notional() decimal.Decimal { return q.notional }
func (q *Quote) NotionalCCY() string       { return q.notionalCCY }
func (q *Quote) ValidFrom() time.Time      { return q.validFrom }
func (q *Quote) Venue() string             { return q.venue }
func (q *Quote) RequesterBIC() string      { return q.requesterBIC }
func (q *Quote) ProviderBIC() string       { return q.providerBIC }
func (q *Quote) Side() Side                { return q.side }

func (q *Quote) ValidTo() time.Time { return q.validTo }
func (q *Quote) Version() int       { return q.version }
func (q *Quote) PendingEvents() []DomainEvent {
	return append([]DomainEvent(nil), q.events...)
}
func (q *Quote) MarkEventsCommitted() { q.events = nil }

// DealRate is the price the requester actually deals at.
//
// Buying the base currency lifts the offer; selling it hits the bid. Using the
// mid instead — which is what callers did while the quote carried no side —
// hands the requester half the spread in either direction.
func (q *Quote) DealRate() decimal.Decimal {
	if q.side == SideSell {
		return q.bid
	}
	return q.ask
}

// BuyerBIC is the party that ends up buying the BASE currency.
//
// The resulting trade is always framed as bought=base / sold=quote, so the legs
// swap with the side: on a SELL the requester is the seller and the dealer takes
// the other side.
func (q *Quote) BuyerBIC() string {
	if q.side == SideSell {
		return q.providerBIC
	}
	return q.requesterBIC
}

// SellerBIC is the counterparty to BuyerBIC.
func (q *Quote) SellerBIC() string {
	if q.side == SideSell {
		return q.requesterBIC
	}
	return q.providerBIC
}

// IsActiveAt reports whether the quote is within its validity window at `t`.
func (q *Quote) IsActiveAt(t time.Time) bool {
	t = t.UTC()
	if t.Before(q.validFrom) {
		return false
	}
	if t.After(q.validTo) {
		return false
	}
	return true
}

// Accept records acceptance of this quote at `t`. Returns ErrQuoteExpired if `t` is outside the validity window.
// On success records an EventQuoteAccepted event that the application layer
// translates into a trade-creation command.
func (q *Quote) Accept(t time.Time, acceptorActor string) error {
	if !q.IsActiveAt(t) {
		return ErrQuoteExpired
	}
	q.version++
	q.recordEvent(EventQuoteAccepted{
		QuoteID:    q.id,
		Actor:      acceptorActor,
		AcceptedAt: t.UTC(),
	})
	return nil
}

func (q *Quote) recordEvent(e DomainEvent) { q.events = append(q.events, e) }

func validatePair(base, quote string) error {
	base = strings.ToUpper(strings.TrimSpace(base))
	quote = strings.ToUpper(strings.TrimSpace(quote))
	if err := validCCY(base, "base_ccy"); err != nil {
		return err
	}
	if err := validCCY(quote, "quote_ccy"); err != nil {
		return err
	}
	if base == quote {
		return fmt.Errorf("%w: base_ccy and quote_ccy must differ", ErrInvalidInput)
	}
	return nil
}

// normaliseBIC upper-cases and validates an ISO 9362 BIC (8 or 11 characters).
func normaliseBIC(v, field string) (string, error) {
	b := strings.ToUpper(strings.TrimSpace(v))
	if b == "" {
		return "", fmt.Errorf("%w: %s required", ErrInvalidInput, field)
	}
	if len(b) != 8 && len(b) != 11 {
		return "", fmt.Errorf("%w: %s must be 8 or 11 characters (ISO 9362)", ErrInvalidInput, field)
	}
	for _, r := range b {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return "", fmt.Errorf("%w: %s must be alphanumeric", ErrInvalidInput, field)
		}
	}
	return b, nil
}

func validCCY(c, field string) error {
	if len(c) != 3 {
		return fmt.Errorf("%w: %s must be ISO 4217 alpha-3", ErrInvalidInput, field)
	}
	for _, r := range c {
		if r < 'A' || r > 'Z' {
			return fmt.Errorf("%w: %s must be alpha-only", ErrInvalidInput, field)
		}
	}
	return nil
}
