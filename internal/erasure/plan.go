// Package erasure — LGPD Art. 18 IV right-to-erasure executor.
//
// Plan parsing + validation. The executor (executor.go) consumes a Plan
// and applies it transactionally per-table with audit emission.
//
// Reference workflow: docs/security/data-lifecycle/erasure-workflow.md.
package erasure

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed schema/erasure-plan-v1.json
var planSchemaBytes []byte

var (
	planSchemaOnce sync.Once
	planSchema     *jsonschema.Schema
	planSchemaErr  error
)

func loadSchema() (*jsonschema.Schema, error) {
	planSchemaOnce.Do(func() {
		c := jsonschema.NewCompiler()
		if err := c.AddResource("erasure-plan-v1.json", bytes.NewReader(planSchemaBytes)); err != nil {
			planSchemaErr = fmt.Errorf("erasure: embed schema: %w", err)
			return
		}
		planSchema, planSchemaErr = c.Compile("erasure-plan-v1.json")
	})
	return planSchema, planSchemaErr
}

// Op identifies the operation kind. Only two are supported by design:
// `redact` (UPDATE fields to a marker string) and `hard_delete` (DELETE WHERE).
type Op string

const (
	OpRedact     Op = "redact"
	OpHardDelete Op = "hard_delete"
)

// Operation is one mutation against a single table.
type Operation struct {
	Table   string   `json:"table"`
	Where   string   `json:"where"`
	Op      Op       `json:"op"`
	Fields  []string `json:"fields,omitempty"`
	Comment string   `json:"comment,omitempty"`
}

// Plan is the signed instruction set produced by the DPO + Compliance Officer
// after running scripts/lgpd-eligibility.sh and translating the report into ops.
type Plan struct {
	Ticket     string      `json:"ticket"`
	SubjectRef string      `json:"subject_ref"`
	Operations []Operation `json:"operations"`
	// Approvals must be co-signed by both 'dpo' and 'compliance_officer' roles
	// for --execute to proceed.
	Approvals []string `json:"approvals"`
}

// ErrInvalidPlan signals any validation failure on the parsed plan.
var ErrInvalidPlan = errors.New("erasure: invalid plan")

// ParsePlan loads + validates a Plan from JSON bytes.
//
// JSON chosen over YAML to avoid an external dependency in a security-critical
// path. The DPO interface is expected to render YAML for human review and
// convert to JSON before signing.
//
// Validation runs in two layers: first the embedded JSON Schema
// (schemas/erasure-plan-v1.json) enforces shape + enums + per-op constraints,
// then the Go struct-level Validate() performs semantic checks the schema
// can't express (e.g. ticket-string ownership rules).
func ParsePlan(data []byte) (*Plan, error) {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: json: %v", ErrInvalidPlan, err)
	}
	schema, err := loadSchema()
	if err != nil {
		return nil, fmt.Errorf("%w: schema load: %v", ErrInvalidPlan, err)
	}
	if err := schema.Validate(raw); err != nil {
		return nil, fmt.Errorf("%w: schema: %v", ErrInvalidPlan, err)
	}
	var p Plan
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("%w: json: %v", ErrInvalidPlan, err)
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

// Validate runs all structural checks. Called automatically by ParsePlan
// but exposed so tests + callers can re-check after mutation.
func (p *Plan) Validate() error {
	if !strings.HasPrefix(p.Ticket, "LGPD-") {
		return fmt.Errorf("%w: ticket must start with LGPD- (got %q)", ErrInvalidPlan, p.Ticket)
	}
	if p.SubjectRef == "" {
		return fmt.Errorf("%w: subject_ref required", ErrInvalidPlan)
	}
	if len(p.Operations) == 0 {
		return fmt.Errorf("%w: at least one operation required", ErrInvalidPlan)
	}
	for i, op := range p.Operations {
		if err := op.validate(); err != nil {
			return fmt.Errorf("%w: op[%d]: %v", ErrInvalidPlan, i, err)
		}
	}
	return nil
}

func (op Operation) validate() error {
	if op.Table == "" {
		return errors.New("table required")
	}
	if op.Where == "" {
		return errors.New("where required (refusing to mutate full table)")
	}
	switch op.Op {
	case OpRedact:
		if len(op.Fields) == 0 {
			return errors.New("redact requires non-empty fields list")
		}
	case OpHardDelete:
		if len(op.Fields) > 0 {
			return errors.New("hard_delete must not list fields (use redact instead)")
		}
	default:
		return fmt.Errorf("unsupported op %q (must be redact or hard_delete)", op.Op)
	}
	return nil
}

// HasRequiredApprovals returns true iff both 'dpo' and 'compliance_officer'
// appear in p.Approvals (case-insensitive). Required for --execute mode.
func (p *Plan) HasRequiredApprovals() bool {
	have := make(map[string]bool, len(p.Approvals))
	for _, a := range p.Approvals {
		have[strings.ToLower(strings.TrimSpace(a))] = true
	}
	return have["dpo"] && have["compliance_officer"]
}

// RedactionMarker is the canonical replacement value for redacted PII fields.
// Cited by the LGPD article + ticket so audit reads are self-explanatory.
func (p *Plan) RedactionMarker() string {
	return fmt.Sprintf("[REDACTED PER LGPD ART 18 IV %s]", p.Ticket)
}
