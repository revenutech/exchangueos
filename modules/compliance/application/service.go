// Package application — Compliance use cases (BACEN classification + IOF + reporting + screening).
package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/compliance/domain"
	"github.com/revenutech/exchangeos/pkg/bacen"
)

// Repositories — split per aggregate to keep each focused.
type ClassificationRepo interface {
	Save(ctx context.Context, c *domain.Classification) error
	GetByTrade(ctx context.Context, tradeID uuid.UUID) (*domain.Classification, error)
}

type IOFRepo interface {
	Save(ctx context.Context, i *domain.IOFComputation) error
	GetByTrade(ctx context.Context, tradeID uuid.UUID) (*domain.IOFComputation, error)
}

type ReportRepo interface {
	Save(ctx context.Context, r *domain.BACENReport) error
	Get(ctx context.Context, id uuid.UUID) (*domain.BACENReport, error)
}

type ScreeningRepo interface {
	Save(ctx context.Context, s *domain.ScreeningResult) error
}

// SanctionsScreener queries the watchlists (OFAC SDN, UN, EU, COAF) for a
// counterparty and returns the list names that matched. An empty slice with a
// nil error means "screened and clear" — which is a materially different claim
// from "not screened", and only a real provider may make it.
type SanctionsScreener interface {
	Screen(ctx context.Context, bic, lei string) (hits []string, err error)
}

var (
	ErrInvalidInput = errors.New("compliance-app: invalid input")
	ErrNotFound     = errors.New("compliance-app: not found")

	// ErrScreeningUnavailable is returned when no sanctions provider is wired.
	ErrScreeningUnavailable = errors.New("compliance-app: sanctions screening unavailable")
)

// UnavailableScreener is the fail-closed default. It refuses to answer rather
// than reporting a counterparty as clear.
//
// ScreenCounterparty used to build its result straight from the caller's input,
// and the gRPC adapter never populated Hits — so every counterparty screened
// clear at RiskLow without a single watchlist being consulted. Under RN_FX_039
// (COS to SISCOAF) that is a false negative reported as a compliance pass.
//
// Swap this for a real provider at wiring time; until then screening errors out
// loudly instead of silently clearing sanctioned parties.
type UnavailableScreener struct{}

// Screen implements SanctionsScreener.
func (UnavailableScreener) Screen(context.Context, string, string) ([]string, error) {
	return nil, ErrScreeningUnavailable
}

// Service exposes compliance use cases.
type Service struct {
	classifier *bacen.Classifier
	iofCalc    *bacen.IOFCalculator
	classRepo  ClassificationRepo
	iofRepo    IOFRepo
	reportRepo ReportRepo
	screenRepo ScreeningRepo
	screener   SanctionsScreener
}

func NewService(
	classifier *bacen.Classifier,
	iofCalc *bacen.IOFCalculator,
	classRepo ClassificationRepo,
	iofRepo IOFRepo,
	reportRepo ReportRepo,
	screenRepo ScreeningRepo,
	screener SanctionsScreener,
) *Service {
	// Nil is not "no screening" — it is the fail-closed default.
	if screener == nil {
		screener = UnavailableScreener{}
	}
	return &Service{
		classifier: classifier,
		iofCalc:    iofCalc,
		classRepo:  classRepo,
		iofRepo:    iofRepo,
		reportRepo: reportRepo,
		screenRepo: screenRepo,
		screener:   screener,
	}
}

// ClassifyOperation runs the BACEN classifier against a free-text hint (or accepts
// an explicit code) and persists the Classification.
func (s *Service) ClassifyOperation(ctx context.Context, tenantID, tradeID uuid.UUID, codeOrHint string) (*domain.Classification, error) {
	if tenantID == uuid.Nil || tradeID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	// Try exact-code lookup first; fall back to free-text classifier.
	nc, ok := s.classifier.ByCode(codeOrHint)
	if !ok {
		var err error
		nc, err = s.classifier.Classify(codeOrHint)
		if err != nil {
			return nil, err
		}
	}
	c, err := domain.NewClassification(domain.NewClassificationInput{
		TenantID: tenantID, TradeID: tradeID,
		Code: nc.Code, Description: nc.Description,
		Nature: domain.Nature(nc.Nature),
	})
	if err != nil {
		return nil, err
	}
	if err := s.classRepo.Save(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// ComputeIOF runs the IOF calculator and persists.
func (s *Service) ComputeIOF(ctx context.Context, tenantID, tradeID uuid.UUID, opType string, notional decimal.Decimal, notionalCCY string) (*domain.IOFComputation, error) {
	if tenantID == uuid.Nil || tradeID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	rate, _, err := s.iofCalc.Compute(opType, notional)
	if err != nil {
		return nil, err
	}
	iof, err := domain.NewIOFComputation(domain.NewIOFInput{
		TenantID: tenantID, TradeID: tradeID,
		OperationType: opType, Notional: notional, NotionalCCY: notionalCCY, Rate: rate,
	})
	if err != nil {
		return nil, err
	}
	if err := s.iofRepo.Save(ctx, iof); err != nil {
		return nil, err
	}
	return iof, nil
}

// SubmitBACENReport persists + immediately marks SUBMITTED. Real submission happens
// downstream (cmd/worker reacts to status change).
func (s *Service) SubmitBACENReport(ctx context.Context, in domain.NewBACENReportInput) (*domain.BACENReport, error) {
	r, err := domain.NewBACENReport(in)
	if err != nil {
		return nil, err
	}
	if err := s.reportRepo.Save(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

// ScreenCounterparty consults the sanctions watchlists and persists the result.
//
// The hits come from the configured SanctionsScreener, never from the caller:
// this used to build the result straight from NewScreeningInput, so a caller
// that supplied no hits — which is exactly what the gRPC adapter did — got a
// clear, RiskLow verdict with no watchlist consulted.
//
// A screening failure is returned, not swallowed. Without a verdict the
// operation must not proceed, and a screening that did not happen must never be
// recorded as a pass.
func (s *Service) ScreenCounterparty(ctx context.Context, in domain.NewScreeningInput) (*domain.ScreeningResult, error) {
	hits, err := s.screener.Screen(ctx, in.CounterpartyBIC, in.LEI)
	if err != nil {
		return nil, err
	}
	in.Hits = hits

	res, err := domain.NewScreeningResult(in)
	if err != nil {
		return nil, err
	}
	if err := s.screenRepo.Save(ctx, res); err != nil {
		return nil, err
	}
	return res, nil
}
