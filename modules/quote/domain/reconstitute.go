package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Reconstitute helpers for the persistence boundary. See
// modules/refdata/domain/reconstitute.go for rationale — these MUST NOT be
// invoked by application code, only by infrastructure repositories.

// ReconstituteQuote rebuilds a Quote with all persisted state including version + id.
func ReconstituteQuote(
	id, tenantID uuid.UUID,
	requesterBIC, providerBIC string,
	side Side,
	baseCCY, quoteCCY string,
	notional decimal.Decimal,
	notionalCCY string,
	bid, ask decimal.Decimal,
	validFrom, validTo time.Time,
	venue string,
	version int,
) *Quote {
	return &Quote{
		id:           id,
		tenantID:     tenantID,
		requesterBIC: requesterBIC,
		providerBIC:  providerBIC,
		side:         side,
		baseCCY:      baseCCY,
		quoteCCY:     quoteCCY,
		notional:     notional,
		notionalCCY:  notionalCCY,
		bid:          bid,
		ask:          ask,
		validFrom:    validFrom.UTC(),
		validTo:      validTo.UTC(),
		venue:        venue,
		version:      version,
	}
}

// ReconstituteRFQ rebuilds an RFQ with all persisted state.
func ReconstituteRFQ(
	id, tenantID uuid.UUID,
	requester, baseCCY, quoteCCY string,
	status RFQStatus,
	quoteIDs []uuid.UUID,
	createdAt time.Time,
	version int,
) *RFQ {
	return &RFQ{
		id:        id,
		tenantID:  tenantID,
		requester: requester,
		baseCCY:   baseCCY,
		quoteCCY:  quoteCCY,
		status:    status,
		quoteIDs:  append([]uuid.UUID(nil), quoteIDs...),
		createdAt: createdAt.UTC(),
		version:   version,
	}
}
