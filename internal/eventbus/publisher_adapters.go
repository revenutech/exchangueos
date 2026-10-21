package eventbus

import (
	"context"

	qdomain "github.com/revenutech/exchangeos/modules/quote/domain"
	tdomain "github.com/revenutech/exchangeos/modules/trade/domain"
)

// QuotePublisher adapts the in-process Bus to quoteapp.EventPublisher.
// Each quote-domain event is wrapped (it already satisfies eventbus.Event via EventName)
// and Publish is dispatched.
type QuotePublisher struct{ Bus *Bus }

func (p QuotePublisher) Publish(ctx context.Context, events []qdomain.DomainEvent) error {
	wrapped := make([]Event, 0, len(events))
	for _, e := range events {
		wrapped = append(wrapped, e)
	}
	_ = p.Bus.Publish(ctx, wrapped...)
	return nil
}

// TradePublisher adapts the in-process Bus to tradeapp.EventPublisher.
type TradePublisher struct{ Bus *Bus }

func (p TradePublisher) Publish(ctx context.Context, events []tdomain.DomainEvent) error {
	wrapped := make([]Event, 0, len(events))
	for _, e := range events {
		wrapped = append(wrapped, e)
	}
	_ = p.Bus.Publish(ctx, wrapped...)
	return nil
}
