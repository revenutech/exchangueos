// Package application — Compliance use cases (BACEN classification + IOF + reporting + screening).
package application

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/compliance/domain"
	"github.com/revenutech/exchangeos/pkg/bacen"
	"github.com/revenutech/exchangeos/pkg/sanctions"
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

	// ErrScreeningFailed is returned when a wired screening could not reach a verdict: a list failed, the
	// counterparty name could not be resolved, or the context ended. It never means "clear".
	ErrScreeningFailed = errors.New("compliance-app: sanctions screening failed")
)

// UnavailableScreener is the fail-closed default. It refuses to answer rather
// than reporting a counterparty as clear.
//
// ScreenCounterparty used to build its result straight from the caller's input,
// and the gRPC adapter never populated Hits — so every counterparty screened
// clear at RiskLow without a single watchlist being consulted. Under RN_FX_039
// (COS to SISCOAF) that is a false negative reported as a compliance pass.
//
// Swap this for a real provider at wiring time (NewProviderScreener); until then
// screening errors out loudly instead of silently clearing sanctioned parties.
type UnavailableScreener struct{}

// Screen implements SanctionsScreener.
func (UnavailableScreener) Screen(context.Context, string, string) ([]string, error) {
	return nil, ErrScreeningUnavailable
}

// DefaultSanctionsMinScore is the Jaro-Winkler threshold of MS-024d.
const DefaultSanctionsMinScore = 0.85

// requiredSanctionsSources are the lists a ProviderScreener must cover. A screen that skipped one would report a
// counterparty as clear without having consulted that list.
var requiredSanctionsSources = []sanctions.Source{
	sanctions.SourceOFAC, sanctions.SourceUN, sanctions.SourceEU, sanctions.SourceCOAF,
}

// CounterpartyNameResolver returns the name a counterparty is screened under (the institution name of its BIC,
// in production from refdata). The name is PII: implementations must not log it nor put it in their errors.
type CounterpartyNameResolver interface {
	CounterpartyName(ctx context.Context, bic, lei string) (string, error)
}

// ProviderScreener is the SanctionsScreener over the live lists (MS-024d 1.12, RTSK.024.1212.01; RN_FX_039).
//
// Screen resolves the counterparty name and queries every provider (OFAC, UN, EU, COAF) with the same MatchQuery.
// The matches of all lists are merged, one per (source, entry id) at its best score, and returned as
// "<SOURCE>:<entry id>" ordered best score first, ties by source then entry id: the first hit is the
// highest-scoring one across the four lists.
//
// The query carries no country: narrowing a sanctions screen by the BIC's country would miss a listed party
// designated under another country.
//
// Error policy, fail-closed: if the name cannot be resolved (error or blank), the context ends, or any one provider
// fails, Screen returns an error wrapping ErrScreeningFailed and no hits, so ScreenCounterparty persists nothing.
// Partial results are never returned: a counterparty is not cleared on three lists out of four. Errors name the
// failing list and never the counterparty.
type ProviderScreener struct {
	names     CounterpartyNameResolver
	providers []sanctions.SanctionsProvider
	minScore  float64
}

var _ SanctionsScreener = (*ProviderScreener)(nil)

// NewProviderScreener wires the live lists. It requires a resolver, exactly one provider per required list
// (OFAC, UN, EU, COAF) and a threshold in (0, 1]; anything else is ErrInvalidInput.
func NewProviderScreener(names CounterpartyNameResolver, providers []sanctions.SanctionsProvider, minScore float64) (*ProviderScreener, error) {
	if names == nil {
		return nil, fmt.Errorf("%w: counterparty name resolver required", ErrInvalidInput)
	}
	if minScore <= 0 || minScore > 1 {
		return nil, fmt.Errorf("%w: sanctions min score %v outside (0, 1]", ErrInvalidInput, minScore)
	}
	seen := make(map[sanctions.Source]bool, len(providers))
	for _, p := range providers {
		if p == nil {
			return nil, fmt.Errorf("%w: nil sanctions provider", ErrInvalidInput)
		}
		if seen[p.Source()] {
			return nil, fmt.Errorf("%w: duplicate sanctions provider for %s", ErrInvalidInput, p.Source())
		}
		seen[p.Source()] = true
	}
	for _, src := range requiredSanctionsSources {
		if !seen[src] {
			return nil, fmt.Errorf("%w: no sanctions provider for %s", ErrInvalidInput, src)
		}
	}
	return &ProviderScreener{names: names, providers: slices.Clone(providers), minScore: minScore}, nil
}

// Screen implements SanctionsScreener.
func (s *ProviderScreener) Screen(ctx context.Context, bic, lei string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrScreeningFailed, err)
	}
	name, err := s.names.CounterpartyName(ctx, bic, lei)
	if err != nil {
		return nil, fmt.Errorf("%w: resolve counterparty name: %w", ErrScreeningFailed, err)
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("%w: counterparty has no name to screen", ErrScreeningFailed)
	}
	q := sanctions.MatchQuery{Name: name, MinScore: s.minScore}

	type key struct {
		src sanctions.Source
		id  string
	}
	best := make(map[key]float64)
	for _, p := range s.providers {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrScreeningFailed, p.Source(), err)
		}
		matches, err := p.Match(ctx, q)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrScreeningFailed, p.Source(), err)
		}
		for _, m := range matches {
			k := key{m.Source, m.EntryID}
			if prev, ok := best[k]; !ok || m.Score > prev {
				best[k] = m.Score
			}
		}
	}

	type scored struct {
		key
		score float64
	}
	all := make([]scored, 0, len(best))
	for k, v := range best {
		all = append(all, scored{k, v})
	}
	slices.SortFunc(all, func(a, b scored) int {
		if c := cmp.Compare(b.score, a.score); c != 0 {
			return c
		}
		if c := cmp.Compare(a.src, b.src); c != 0 {
			return c
		}
		return cmp.Compare(a.id, b.id)
	})
	hits := make([]string, len(all))
	for i, h := range all {
		hits[i] = string(h.src) + ":" + h.id
	}
	return hits, nil
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
