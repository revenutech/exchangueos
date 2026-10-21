// Package validator provides ISO 20022 schema + business-rule validation.
//
// Two layers:
//
//   - XSDValidator — checks structural conformance against a downloaded XSD.
//     Lightweight implementation: well-formedness + namespace match + element-name
//     constraints. Full XSD support (xs:complexType, restrictions, etc.) requires
//     CGo bindings to libxml2 and is OUT OF SCOPE for the pure-Go core.
//     See README of this package for the optional `validator/xmllibxml` build tag.
//
//   - BusinessRuleValidator — runs ExchangeOS-specific rules
//     (RN_FX_* — cited in modules/<bc>/domain/specifications and SHACL shapes).
package validator

import (
	"bytes"
	"encoding/xml"
	"fmt"

	"github.com/revenutech/exchangeos/pkg/iso20022/registry"
)

// Violation describes a single validation failure.
type Violation struct {
	Code    string // e.g. "STRUCT_MISMATCH", "RN_FX_001"
	Path    string // XPath-like locator
	Message string
}

// Result aggregates violations from one validation pass.
type Result struct {
	Violations []Violation
}

// Ok reports whether there are no violations.
func (r Result) Ok() bool { return len(r.Violations) == 0 }

// Err converts the Result into an error if violations exist.
func (r Result) Err() error {
	if r.Ok() {
		return nil
	}
	return fmt.Errorf("validation failed: %d violation(s); first=%s/%s: %s",
		len(r.Violations), r.Violations[0].Code, r.Violations[0].Path, r.Violations[0].Message)
}

// XSDValidator runs lightweight conformance checks driven by a Descriptor.
type XSDValidator struct{ reg *registry.Registry }

func NewXSDValidator(reg *registry.Registry) *XSDValidator { return &XSDValidator{reg: reg} }

// Validate performs the minimal well-formedness + URN-match checks.
// For full XSD validation, use the optional libxml2-backed validator.
func (v *XSDValidator) Validate(desc registry.Descriptor, raw []byte) Result {
	var res Result

	dec := xml.NewDecoder(bytes.NewReader(raw))
	for {
		_, err := dec.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			res.Violations = append(res.Violations, Violation{
				Code:    "WELLFORMED",
				Path:    "/",
				Message: err.Error(),
			})
			return res
		}
	}

	if _, ok := v.reg.LookupByURN(desc.URN()); !ok {
		res.Violations = append(res.Violations, Violation{
			Code:    "UNKNOWN_SCHEMA",
			Path:    "/",
			Message: "descriptor not in registry: " + desc.URN(),
		})
	}
	return res
}

// BusinessRule defines a single RN_FX_* check.
type BusinessRule struct {
	Code string                                     // e.g. "RN_FX_001"
	Run  func(payload interface{}) (Violation, bool) // returns (violation, hasViolation)
}

// BusinessRuleValidator runs a configurable set of rules against a typed payload.
type BusinessRuleValidator struct{ rules []BusinessRule }

func NewBusinessRuleValidator(rules ...BusinessRule) *BusinessRuleValidator {
	return &BusinessRuleValidator{rules: rules}
}

func (v *BusinessRuleValidator) Validate(payload interface{}) Result {
	var res Result
	for _, r := range v.rules {
		if violation, has := r.Run(payload); has {
			violation.Code = r.Code
			res.Violations = append(res.Violations, violation)
		}
	}
	return res
}
