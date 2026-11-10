# Fixture — complete pattern

## FX-GP-901 — Aggregate constructor returns `(*T, error)`

**Context:** Every aggregate needs validated construction.

**Problem:** Zero-value structs bypass invariants.

**Solution:** Private fields and a constructor that validates.

**Example:**

```go
func NewFXTrade(in NewTradeInput) (*FXTrade, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	return &FXTrade{}, nil
}
```

Origin: `modules/trade/domain/fxtrade.go:NewFXTrade`.

**Anti-pattern:** Exposing the struct literal lets callers skip validation.

**Related:** FX-GP-007 (Reconstitute), RN_FX_026.
