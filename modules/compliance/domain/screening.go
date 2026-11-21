package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RiskLevel is the screening risk outcome.
type RiskLevel string

const (
	RiskLow    RiskLevel = "LOW"
	RiskMedium RiskLevel = "MEDIUM"
	RiskHigh   RiskLevel = "HIGH"
)

// ScreeningResult — outcome of a sanctions screen against OFAC/UN/EU/COAF.
// Cite RN_FX_039 (COS for SISCOAF when risk HIGH).
type ScreeningResult struct {
	id              uuid.UUID
	tenantID        uuid.UUID
	counterpartyBIC string
	lei             string
	hits            []string // list-prefixed hits, e.g. "OFAC:SDN:...", "UN:1267:..."
	riskLevel       RiskLevel
	screenedAt      time.Time
}

// NewScreeningInput parameterises construction.
type NewScreeningInput struct {
	TenantID        uuid.UUID
	CounterpartyBIC string
	LEI             string
	Hits            []string
}

// NewScreeningResult derives risk_level from hits:
//
//	0 hits  → LOW
//	1-2     → MEDIUM
//	3+      → HIGH  (caller MUST emit COS for SISCOAF when HIGH)
func NewScreeningResult(in NewScreeningInput) (*ScreeningResult, error) {
	if in.TenantID == uuid.Nil {
		return nil, fmt.Errorf("%w: tenant_id required", ErrInvalidInput)
	}
	bic := strings.ToUpper(strings.TrimSpace(in.CounterpartyBIC))
	if len(bic) != 8 && len(bic) != 11 {
		return nil, fmt.Errorf("%w: counterparty_bic must be 8 or 11 chars", ErrInvalidInput)
	}
	if in.LEI != "" && len(in.LEI) != 20 {
		return nil, fmt.Errorf("%w: LEI when present must be 20 chars", ErrInvalidInput)
	}
	risk := RiskLow
	switch {
	case len(in.Hits) >= 3:
		risk = RiskHigh
	case len(in.Hits) >= 1:
		risk = RiskMedium
	}
	return &ScreeningResult{
		id:              uuid.New(),
		tenantID:        in.TenantID,
		counterpartyBIC: bic,
		lei:             in.LEI,
		hits:            append([]string(nil), in.Hits...),
		riskLevel:       risk,
		screenedAt:      time.Now().UTC(),
	}, nil
}

// Accessors
func (s *ScreeningResult) ID() uuid.UUID           { return s.id }
func (s *ScreeningResult) TenantID() uuid.UUID     { return s.tenantID }
func (s *ScreeningResult) CounterpartyBIC() string { return s.counterpartyBIC }
func (s *ScreeningResult) LEI() string             { return s.lei }
func (s *ScreeningResult) Hits() []string          { return append([]string(nil), s.hits...) }
func (s *ScreeningResult) RiskLevel() RiskLevel    { return s.riskLevel }
func (s *ScreeningResult) ScreenedAt() time.Time   { return s.screenedAt }
func (s *ScreeningResult) IsClear() bool           { return s.riskLevel == RiskLow }
func (s *ScreeningResult) RequiresCOS() bool       { return s.riskLevel == RiskHigh }
