package bmecat_test

import (
	"testing"
	"time"

	"github.com/olivere/bmecat"
)

func tp(s string) *time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return &t
}

func TestPriceDetailsIsValidAt(t *testing.T) {
	at := time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		pd   *bmecat.PriceDetails
		want bool
	}{
		{"open both ends", &bmecat.PriceDetails{}, true},
		{"within window", &bmecat.PriceDetails{ValidStart: tp("2020-01-01"), ValidEnd: tp("2020-06-30")}, true},
		{"before start", &bmecat.PriceDetails{ValidStart: tp("2020-04-01")}, false},
		{"after end", &bmecat.PriceDetails{ValidEnd: tp("2020-02-01")}, false},
		{"open start, before end", &bmecat.PriceDetails{ValidEnd: tp("2020-06-30")}, true},
		{"open end, after start", &bmecat.PriceDetails{ValidStart: tp("2020-01-01")}, true},
		{"inclusive start bound", &bmecat.PriceDetails{ValidStart: tp("2020-03-15")}, true},
		{"inclusive end bound", &bmecat.PriceDetails{ValidEnd: tp("2020-03-15")}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pd.IsValidAt(at); got != tt.want {
				t.Errorf("IsValidAt = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestProductCurrentAndValidPriceDetails(t *testing.T) {
	first := &bmecat.PriceDetails{ValidStart: tp("2020-01-01"), ValidEnd: tp("2020-06-30")}
	second := &bmecat.PriceDetails{ValidStart: tp("2020-07-01")}
	overlap := &bmecat.PriceDetails{ValidStart: tp("2020-06-01"), ValidEnd: tp("2020-12-31")}
	p := &bmecat.Product{PriceDetails: []*bmecat.PriceDetails{first, second, overlap}}

	// In the first window only.
	if got := p.CurrentPriceDetails(time.Date(2020, 3, 1, 0, 0, 0, 0, time.UTC)); got != first {
		t.Errorf("CurrentPriceDetails = %p, want first %p", got, first)
	}
	// Overlap of second and overlap: returns first in document order (second).
	at := time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC)
	if got := p.CurrentPriceDetails(at); got != second {
		t.Errorf("CurrentPriceDetails = %p, want second %p", got, second)
	}
	if got := p.ValidPriceDetails(at); len(got) != 2 || got[0] != second || got[1] != overlap {
		t.Errorf("ValidPriceDetails = %v, want [second overlap]", got)
	}
	// No window matches.
	before := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := p.CurrentPriceDetails(before); got != nil {
		t.Errorf("CurrentPriceDetails = %v, want nil", got)
	}
	if got := p.ValidPriceDetails(before); got != nil {
		t.Errorf("ValidPriceDetails = %v, want nil", got)
	}
}

func TestPriceDetailsPriceFor(t *testing.T) {
	pd := &bmecat.PriceDetails{
		Prices: []*bmecat.Price{
			{Type: "net_list", Amount: 10, LowerBound: 1},
			{Type: "net_list", Amount: 8, LowerBound: 100},
			{Type: "net_customer", Amount: 7, LowerBound: 1},
		},
	}

	tests := []struct {
		name       string
		quantity   float64
		typ        string
		wantAmount float64 // -1 means expect nil
	}{
		{"below tier, picks lowest", 50, "net_list", 10},
		{"at higher tier bound", 100, "net_list", 8},
		{"above higher tier", 250, "net_list", 8},
		{"other type filtered", 50, "net_customer", 7},
		{"empty type considers all, highest bound wins", 100, "", 8},
		{"below lowest bound returns nil", 0.5, "net_list", -1},
		{"unknown type returns nil", 50, "gross_list", -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pd.PriceFor(tt.quantity, tt.typ)
			if tt.wantAmount < 0 {
				if got != nil {
					t.Errorf("PriceFor = %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("PriceFor = nil, want amount %v", tt.wantAmount)
			}
			if got.Amount != tt.wantAmount {
				t.Errorf("PriceFor amount = %v, want %v", got.Amount, tt.wantAmount)
			}
		})
	}
}
