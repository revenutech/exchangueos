package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SSI — Standing Settlement Instruction for a (tenant, counterparty, currency) triple.
// RN_FX_017: SSI is mandatory before first settlement with a counterparty.
type SSI struct {
	id              uuid.UUID
	tenantID        uuid.UUID
	counterpartyBIC string
	currency        string
	beneficiaryBIC  string
	intermediaryBIC string
	accountNumber   string
	iban            string
	validFrom       time.Time
	validTo         time.Time // zero = open-ended
}

// NewSSIInput parameterises construction.
type NewSSIInput struct {
	TenantID        uuid.UUID
	CounterpartyBIC string
	Currency        string
	BeneficiaryBIC  string
	IntermediaryBIC string
	AccountNumber   string
	IBAN            string
	ValidFrom       time.Time
	ValidTo         time.Time
}

// NewSSI constructs and validates an SSI.
func NewSSI(in NewSSIInput) (*SSI, error) {
	if in.TenantID == uuid.Nil {
		return nil, fmt.Errorf("%w: tenant_id required", ErrInvalidInput)
	}
	if err := requireBICLen(in.CounterpartyBIC, "counterparty_bic"); err != nil {
		return nil, err
	}
	if err := requireBICLen(in.BeneficiaryBIC, "beneficiary_bic"); err != nil {
		return nil, err
	}
	if in.IntermediaryBIC != "" {
		if err := requireBICLen(in.IntermediaryBIC, "intermediary_bic"); err != nil {
			return nil, err
		}
	}
	cur := strings.ToUpper(strings.TrimSpace(in.Currency))
	if len(cur) != 3 {
		return nil, fmt.Errorf("%w: currency must be ISO 4217 alpha-3", ErrInvalidInput)
	}
	if in.AccountNumber == "" && in.IBAN == "" {
		return nil, fmt.Errorf("%w: account_number or IBAN required", ErrInvalidInput)
	}
	if in.IBAN != "" && (len(in.IBAN) < 15 || len(in.IBAN) > 34) {
		return nil, fmt.Errorf("%w: IBAN length out of range (15-34)", ErrInvalidInput)
	}
	if in.ValidFrom.IsZero() {
		return nil, fmt.Errorf("%w: valid_from required", ErrInvalidInput)
	}
	if !in.ValidTo.IsZero() && in.ValidTo.Before(in.ValidFrom) {
		return nil, fmt.Errorf("%w: valid_to must be >= valid_from", ErrInvalidInput)
	}
	return &SSI{
		id:              uuid.New(),
		tenantID:        in.TenantID,
		counterpartyBIC: strings.ToUpper(strings.TrimSpace(in.CounterpartyBIC)),
		currency:        cur,
		beneficiaryBIC:  strings.ToUpper(strings.TrimSpace(in.BeneficiaryBIC)),
		intermediaryBIC: strings.ToUpper(strings.TrimSpace(in.IntermediaryBIC)),
		accountNumber:   in.AccountNumber,
		iban:            in.IBAN,
		validFrom:       in.ValidFrom.UTC(),
		validTo:         in.ValidTo.UTC(),
	}, nil
}

func (s *SSI) ID() uuid.UUID           { return s.id }
func (s *SSI) TenantID() uuid.UUID     { return s.tenantID }
func (s *SSI) Currency() string        { return s.currency }
func (s *SSI) CounterpartyBIC() string { return s.counterpartyBIC }
func (s *SSI) BeneficiaryBIC() string  { return s.beneficiaryBIC }
func (s *SSI) IntermediaryBIC() string { return s.intermediaryBIC }
func (s *SSI) AccountNumber() string   { return s.accountNumber }
func (s *SSI) IBAN() string            { return s.iban }

// IsActiveAt reports whether the SSI is valid at the given instant.
func (s *SSI) IsActiveAt(t time.Time) bool {
	t = t.UTC()
	if t.Before(s.validFrom) {
		return false
	}
	if !s.validTo.IsZero() && t.After(s.validTo) {
		return false
	}
	return true
}

func requireBICLen(bic, field string) error {
	bic = strings.ToUpper(strings.TrimSpace(bic))
	switch len(bic) {
	case 8, 11:
		return nil
	default:
		return fmt.Errorf("%w: %s must be 8 or 11 chars", ErrInvalidInput, field)
	}
}
