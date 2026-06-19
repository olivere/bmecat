package bmecat

import "time"

// IsValidAt reports whether t falls within the wrapper's validity window. A nil
// ValidStart means the wrapper is valid from the beginning of time; a nil
// ValidEnd means it is valid indefinitely (open-ended). Both bounds are
// inclusive.
func (pd *PriceDetails) IsValidAt(t time.Time) bool {
	if pd.ValidStart != nil && t.Before(*pd.ValidStart) {
		return false
	}
	if pd.ValidEnd != nil && t.After(*pd.ValidEnd) {
		return false
	}
	return true
}

// CurrentPriceDetails returns the first PriceDetails wrapper, in document order,
// whose validity window contains at. It returns nil when no wrapper is valid at
// that time. See PriceDetails.IsValidAt for the window semantics.
//
// When a product carries a price calendar with overlapping windows, this
// returns only the first match; use ValidPriceDetails to detect or inspect all
// of them.
func (p *Product) CurrentPriceDetails(at time.Time) *PriceDetails {
	for _, pd := range p.PriceDetails {
		if pd.IsValidAt(at) {
			return pd
		}
	}
	return nil
}

// ValidPriceDetails returns every PriceDetails wrapper, in document order, whose
// validity window contains at. It returns nil when none is valid. A result with
// more than one element indicates overlapping validity windows (a price
// calendar), which callers may want to warn about.
func (p *Product) ValidPriceDetails(at time.Time) []*PriceDetails {
	var out []*PriceDetails
	for _, pd := range p.PriceDetails {
		if pd.IsValidAt(at) {
			out = append(out, pd)
		}
	}
	return out
}

// PriceFor selects the price tier applicable to the given quantity from the
// wrapper's Prices. It returns the price with the highest LowerBound that does
// not exceed quantity. Prices whose Type does not match typ are ignored; pass
// an empty typ to consider every price regardless of type.
//
// It returns nil when no price applies — for example when quantity is below the
// lowest LowerBound, or when no price carries the requested type.
//
// PriceFor does not apply Factor; callers that need the factor-adjusted amount
// should multiply the returned Price.Amount by Price.Factor themselves.
func (pd *PriceDetails) PriceFor(quantity float64, typ string) *Price {
	var best *Price
	for _, p := range pd.Prices {
		if typ != "" && p.Type != typ {
			continue
		}
		if p.LowerBound > quantity {
			continue
		}
		if best == nil || p.LowerBound > best.LowerBound {
			best = p
		}
	}
	return best
}
