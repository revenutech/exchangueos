package registry

import (
	"strings"
)

// OrganisationRouter resolves which ISO 20022 variant to use given a
// counterparty BIC or country code, choosing between OrgCLS, OrgCFETS, or OrgISO (fallback).
//
// Rules (cite RN_FX_010 + RN_FX_002):
//
//   - BIC matches CLSBUS33 OR counterparty appears in CLSMembers → OrgCLS
//   - Counterparty country = CN OR BIC bank code = CFETSCN…    → OrgCFETS
//   - Otherwise                                                 → OrgISO (generic ISO 20022)
type OrganisationRouter struct {
	clsMembers  map[string]struct{} // set of BICs (uppercase)
	cfetsPrefix string              // BIC prefix that signals CFETS (e.g. "CFETS")
	defaultOrg  Organisation
}

// NewOrganisationRouter constructs a Router with the given CLS member set + CFETS BIC prefix.
// clsMembers may be nil — in that case only CLSBUS33 itself routes to CLS.
func NewOrganisationRouter(clsMembers []string, cfetsPrefix string) *OrganisationRouter {
	set := make(map[string]struct{}, len(clsMembers)+1)
	set["CLSBUS33"] = struct{}{}
	for _, m := range clsMembers {
		set[strings.ToUpper(strings.TrimSpace(m))] = struct{}{}
	}
	if cfetsPrefix == "" {
		cfetsPrefix = "CFETS"
	}
	return &OrganisationRouter{
		clsMembers:  set,
		cfetsPrefix: strings.ToUpper(cfetsPrefix),
		defaultOrg:  OrgISO,
	}
}

// RouteParty returns the organisation to use for a given counterparty.
// `bic` is required (ISO 9362, 8 or 11 chars). `country` is optional (ISO 3166 alpha-2).
func (r *OrganisationRouter) RouteParty(bic, country string) (Organisation, error) {
	bic = strings.ToUpper(strings.TrimSpace(bic))
	country = strings.ToUpper(strings.TrimSpace(country))

	if bic == "" {
		return "", ErrInvalidParty
	}
	if _, ok := r.clsMembers[bic]; ok {
		return OrgCLS, nil
	}
	if strings.HasPrefix(bic, r.cfetsPrefix) || country == "CN" {
		return OrgCFETS, nil
	}
	return r.defaultOrg, nil
}

// AddCLSMember dynamically extends the CLS member list (e.g. on refdata refresh).
func (r *OrganisationRouter) AddCLSMember(bic string) {
	r.clsMembers[strings.ToUpper(strings.TrimSpace(bic))] = struct{}{}
}

// IsCLSMember reports whether the given BIC is a known CLS member.
func (r *OrganisationRouter) IsCLSMember(bic string) bool {
	_, ok := r.clsMembers[strings.ToUpper(strings.TrimSpace(bic))]
	return ok
}
