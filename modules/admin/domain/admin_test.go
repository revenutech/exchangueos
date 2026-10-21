package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/admin/domain"
)

func TestSystemEvent_Happy(t *testing.T) {
	e, err := domain.NewSystemEvent(domain.NewSystemEventInput{
		Code: domain.EventCycleOpen, Component: "cls-cycle", Description: "07:00 CET",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if e.Code() != domain.EventCycleOpen {
		t.Errorf("code: %s", e.Code())
	}
	if e.At().IsZero() {
		t.Error("At defaulted to zero")
	}
}

func TestSystemEvent_BadInputs(t *testing.T) {
	if _, err := domain.NewSystemEvent(domain.NewSystemEventInput{Code: "BOGUS", Component: "x"}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("bad code: %v", err)
	}
	if _, err := domain.NewSystemEvent(domain.NewSystemEventInput{Code: domain.EventStartup}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("empty comp: %v", err)
	}
}

func TestEODJob_Lifecycle_Happy(t *testing.T) {
	j, err := domain.NewEODJob(domain.NewEODJobInput{
		TenantID: uuid.New(), BusinessDate: time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := j.Start(time.Now().UTC()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	for _, step := range []string{"PTAX", "MTM", "POSITION_SNAPSHOT", "BACEN_REPORT"} {
		if err := j.MarkStep(step); err != nil {
			t.Fatalf("MarkStep(%s): %v", step, err)
		}
	}
	// Idempotence: re-marking an existing step does NOT duplicate.
	_ = j.MarkStep("PTAX")
	if len(j.StepsDone()) != 4 {
		t.Fatalf("steps: %v", j.StepsDone())
	}
	if err := j.Complete(time.Now().UTC()); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if j.Status() != domain.EODCompleted {
		t.Errorf("status: %s", j.Status())
	}
}

func TestEODJob_FailPath(t *testing.T) {
	j, _ := domain.NewEODJob(domain.NewEODJobInput{
		TenantID: uuid.New(), BusinessDate: time.Now().UTC(),
	})
	_ = j.Start(time.Now().UTC())
	if err := j.Fail(time.Now().UTC(), ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("missing reason: %v", err)
	}
	if err := j.Fail(time.Now().UTC(), "PTAX feeder timeout"); err != nil {
		t.Fatalf("Fail: %v", err)
	}
	if j.Status() != domain.EODFailed {
		t.Errorf("status: %s", j.Status())
	}
	if err := j.Fail(time.Now().UTC(), "again"); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("re-fail: %v", err)
	}
}

func TestEODJob_BadInputs(t *testing.T) {
	if _, err := domain.NewEODJob(domain.NewEODJobInput{}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("missing tenant+date: %v", err)
	}
}

func TestEODJob_StartRequiresPending(t *testing.T) {
	j, _ := domain.NewEODJob(domain.NewEODJobInput{TenantID: uuid.New(), BusinessDate: time.Now().UTC()})
	_ = j.Start(time.Now().UTC())
	if err := j.Start(time.Now().UTC()); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("re-start: %v", err)
	}
}
