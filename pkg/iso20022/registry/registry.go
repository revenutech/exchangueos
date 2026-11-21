// Package registry — ISO 20022 schema Version Registry + Organisation Router.
//
// Every inbound/outbound message is identified by (Organisation, BusinessDomain,
// MessageDefinition, Variant, Version). The Registry resolves that tuple to:
//
//   - the pinned XSD source URL (for build-time download / validation)
//   - the Go struct factory (for marshal/unmarshal)
//   - submitting organisation (CLSBUS33 vs CFETS)
//
// Adding a new schema = single Register() call in init() block.
package registry

import (
	"fmt"
	"strings"
	"sync"
)

// Organisation identifies the ISO 20022 submitting body whose variant we use.
type Organisation string

const (
	OrgISO    Organisation = "ISO20022" // base/published spec
	OrgCLS    Organisation = "CLSBUS33" // CLS Bank variant
	OrgCFETS  Organisation = "CFETS"    // China FX Trade System variant
	OrgRevenu Organisation = "REVENU"   // Revenu internal (extension fields)
)

// BusinessDomain — ISO 20022 4-letter business area.
type BusinessDomain string

const (
	BusinessFXTR BusinessDomain = "fxtr"
	BusinessADMI BusinessDomain = "admi"
	BusinessCAMT BusinessDomain = "camt"
	BusinessREDA BusinessDomain = "reda"
	BusinessHEAD BusinessDomain = "head"
)

// Descriptor uniquely identifies one schema variant.
type Descriptor struct {
	Organisation Organisation
	Domain       BusinessDomain
	MessageDef   string // e.g. "014" — the 3-digit ID under the domain
	Variant      string // e.g. "001" — submitter variant suffix
	Version      string // semver/pinned, e.g. "2021-09-17"
	XSDSourceURL string // pinned URL for download / verification
	Description  string // human-readable summary
}

// URN returns the canonical ISO 20022 URN, e.g.
//
//	urn:iso:std:iso:20022:tech:xsd:fxtr.014.001.05
func (d Descriptor) URN() string {
	return fmt.Sprintf("urn:iso:std:iso:20022:tech:xsd:%s.%s.%s.%s",
		d.Domain, d.MessageDef, d.Variant, d.Version)
}

// Key returns a registry lookup key (case-insensitive).
func (d Descriptor) Key() string {
	return strings.ToLower(fmt.Sprintf("%s/%s.%s.%s/%s",
		d.Organisation, d.Domain, d.MessageDef, d.Variant, d.Version))
}

// Registry is the central, thread-safe schema catalog.
type Registry struct {
	mu    sync.RWMutex
	byKey map[string]Descriptor
	byURN map[string]Descriptor
}

// New constructs an empty Registry.
func New() *Registry {
	return &Registry{
		byKey: make(map[string]Descriptor),
		byURN: make(map[string]Descriptor),
	}
}

// Register adds a descriptor. Returns ErrDuplicate if the (org,domain,msg,variant,version) is already known.
func (r *Registry) Register(d Descriptor) error {
	if err := d.Validate(); err != nil {
		return fmt.Errorf("invalid descriptor %s: %w", d.Key(), err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byKey[d.Key()]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicate, d.Key())
	}
	r.byKey[d.Key()] = d
	r.byURN[d.URN()] = d
	return nil
}

// MustRegister panics on error. Use only in init() with verified literals.
func (r *Registry) MustRegister(d Descriptor) {
	if err := r.Register(d); err != nil {
		panic(err)
	}
}

// LookupByKey resolves by Key().
func (r *Registry) LookupByKey(key string) (Descriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.byKey[strings.ToLower(key)]
	return d, ok
}

// LookupByURN resolves by canonical URN.
func (r *Registry) LookupByURN(urn string) (Descriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.byURN[urn]
	return d, ok
}

// List returns all registered descriptors (copy).
func (r *Registry) List() []Descriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Descriptor, 0, len(r.byKey))
	for _, d := range r.byKey {
		out = append(out, d)
	}
	return out
}

// FilterByOrganisation returns descriptors matching org.
func (r *Registry) FilterByOrganisation(org Organisation) []Descriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Descriptor
	for _, d := range r.byKey {
		if d.Organisation == org {
			out = append(out, d)
		}
	}
	return out
}

// FilterByDomain returns descriptors matching domain.
func (r *Registry) FilterByDomain(domain BusinessDomain) []Descriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Descriptor
	for _, d := range r.byKey {
		if d.Domain == domain {
			out = append(out, d)
		}
	}
	return out
}

// Validate enforces invariants on a Descriptor.
func (d Descriptor) Validate() error {
	switch d.Organisation {
	case OrgISO, OrgCLS, OrgCFETS, OrgRevenu:
	default:
		return fmt.Errorf("unknown organisation %q", d.Organisation)
	}
	switch d.Domain {
	case BusinessFXTR, BusinessADMI, BusinessCAMT, BusinessREDA, BusinessHEAD:
	default:
		return fmt.Errorf("unknown business domain %q", d.Domain)
	}
	if d.MessageDef == "" || d.Variant == "" || d.Version == "" {
		return ErrEmptyField
	}
	if d.XSDSourceURL == "" {
		return ErrMissingXSD
	}
	return nil
}
