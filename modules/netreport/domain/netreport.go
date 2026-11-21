package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// NetReport summarises CLS netting for a (cycle, currency) pair. Aligned with
// the line-level model of camt.088.001.02.
type NetReport struct {
	id            uuid.UUID
	tenantID      uuid.UUID
	cycleID       uuid.UUID
	currency      string
	grossPayIn    decimal.Decimal
	grossPayOut   decimal.Decimal
	netSettlement decimal.Decimal
	tradeCount    int
	generatedAt   time.Time
}

// NewNetReportInput parameterises construction.
type NewNetReportInput struct {
	TenantID    uuid.UUID
	CycleID     uuid.UUID
	Currency    string
	GrossPayIn  decimal.Decimal
	GrossPayOut decimal.Decimal
	TradeCount  int
	GeneratedAt time.Time
}

// NewNetReport constructs a NetReport. Net = PayIn − PayOut (positive = receivable; negative = payable).
func NewNetReport(in NewNetReportInput) (*NetReport, error) {
	if in.TenantID == uuid.Nil || in.CycleID == uuid.Nil {
		return nil, fmt.Errorf("%w: tenant_id + cycle_id required", ErrInvalidInput)
	}
	ccy := strings.ToUpper(strings.TrimSpace(in.Currency))
	if len(ccy) != 3 {
		return nil, fmt.Errorf("%w: currency must be ISO 4217 alpha-3", ErrInvalidInput)
	}
	if in.GrossPayIn.IsNegative() || in.GrossPayOut.IsNegative() {
		return nil, fmt.Errorf("%w: gross amounts cannot be negative", ErrInvalidInput)
	}
	if in.TradeCount < 0 {
		return nil, fmt.Errorf("%w: trade_count cannot be negative", ErrInvalidInput)
	}
	gen := in.GeneratedAt
	if gen.IsZero() {
		gen = time.Now().UTC()
	}
	return &NetReport{
		id:            uuid.New(),
		tenantID:      in.TenantID,
		cycleID:       in.CycleID,
		currency:      ccy,
		grossPayIn:    in.GrossPayIn,
		grossPayOut:   in.GrossPayOut,
		netSettlement: in.GrossPayIn.Sub(in.GrossPayOut),
		tradeCount:    in.TradeCount,
		generatedAt:   gen.UTC(),
	}, nil
}

func (n *NetReport) ID() uuid.UUID                  { return n.id }
func (n *NetReport) TenantID() uuid.UUID            { return n.tenantID }
func (n *NetReport) CycleID() uuid.UUID             { return n.cycleID }
func (n *NetReport) Currency() string               { return n.currency }
func (n *NetReport) GrossPayIn() decimal.Decimal    { return n.grossPayIn }
func (n *NetReport) GrossPayOut() decimal.Decimal   { return n.grossPayOut }
func (n *NetReport) NetSettlement() decimal.Decimal { return n.netSettlement }
func (n *NetReport) TradeCount() int                { return n.tradeCount }
func (n *NetReport) GeneratedAt() time.Time         { return n.generatedAt }

// IsReceivable reports whether net settlement is positive (counterparty owes the tenant).
func (n *NetReport) IsReceivable() bool { return n.netSettlement.IsPositive() }

// IsPayable reports whether net settlement is negative (tenant owes the counterparty).
func (n *NetReport) IsPayable() bool { return n.netSettlement.IsNegative() }
