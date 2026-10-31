package bmecat_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/olivere/bmecat"
)

// writeXML writes header and products as one catalog in the given version and
// returns the document.
func writeXML(t *testing.T, version bmecat.Version, products ...*bmecat.Product) string {
	t.Helper()
	var buf bytes.Buffer
	cw := &sliceCatalogWriter{header: fullHeader(), products: products}
	if err := bmecat.NewWriter(&buf, bmecat.WithVersion(version)).Do(context.Background(), cw); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// withNilEntries returns p with a nil element placed before and after the
// entries of every pointer slice on the product and its nested blocks.
func withNilEntries(p *bmecat.Product) *bmecat.Product {
	p.PIDs = append([]*bmecat.TypedValue{nil}, append(p.PIDs, nil)...)
	p.BuyerIDs = append([]*bmecat.TypedValue{nil}, append(p.BuyerIDs, nil)...)
	p.SpecialTreatmentClasses = append([]*bmecat.TypedValue{nil}, append(p.SpecialTreatmentClasses, nil)...)
	p.Status = append([]*bmecat.TypedValue{nil}, append(p.Status, nil)...)
	for _, f := range p.Features {
		f.Features = append([]*bmecat.Feature{nil}, append(f.Features, nil)...)
	}
	p.Features = append([]*bmecat.Features{nil}, append(p.Features, nil)...)
	p.Prices = append([]*bmecat.Price{nil}, append(p.Prices, nil)...)
	for _, pd := range p.PriceDetails {
		pd.Prices = append([]*bmecat.Price{nil}, append(pd.Prices, nil)...)
	}
	p.PriceDetails = append([]*bmecat.PriceDetails{nil}, append(p.PriceDetails, nil)...)
	p.Mimes = append([]*bmecat.Mime{nil}, append(p.Mimes, nil)...)
	p.UDX = append([]*bmecat.UDXField{nil}, append(p.UDX, nil)...)
	return p
}

// TestWriteSkipsNilSliceEntries confirms the writer skips nil entries in a
// product's pointer slices instead of panicking: the output matches the same
// product written without them, and slices holding only nils write as if empty.
func TestWriteSkipsNilSliceEntries(t *testing.T) {
	onlyNils := func() *bmecat.Product {
		p := &bmecat.Product{ID: "1", OrderUnit: "PCE", GTIN: "4006381333931"}
		p.Features = []*bmecat.Features{{SystemName: "ECLASS-5.1", GroupID: "19010203"}}
		p.PriceDetails = []*bmecat.PriceDetails{{}}
		return p
	}
	tests := []struct {
		name     string
		want     func() *bmecat.Product
		withNils func() *bmecat.Product
	}{
		{
			name:     "mixed",
			want:     fullProduct,
			withNils: func() *bmecat.Product { return withNilEntries(fullProduct()) },
		},
		{
			name:     "only nils",
			want:     onlyNils,
			withNils: func() *bmecat.Product { return withNilEntries(onlyNils()) },
		},
		{
			// Without PriceDetails the writer falls back to the flattened Prices.
			name: "only nil prices",
			want: func() *bmecat.Product { return &bmecat.Product{ID: "1", OrderUnit: "PCE"} },
			withNils: func() *bmecat.Product {
				return &bmecat.Product{ID: "1", OrderUnit: "PCE", Prices: []*bmecat.Price{nil}}
			},
		},
		{
			// PriceDetails holding only nils must not hide the flattened Prices.
			name: "only nil price details",
			want: func() *bmecat.Product {
				return &bmecat.Product{ID: "1", OrderUnit: "PCE", Prices: []*bmecat.Price{{Type: "net_list", Amount: 1, Currency: "EUR"}}}
			},
			withNils: func() *bmecat.Product {
				return &bmecat.Product{
					ID: "1", OrderUnit: "PCE",
					Prices:       []*bmecat.Price{{Type: "net_list", Amount: 1, Currency: "EUR"}},
					PriceDetails: []*bmecat.PriceDetails{nil},
				}
			},
		},
	}
	for _, version := range []bmecat.Version{bmecat.Version12, bmecat.Version2005} {
		for _, tt := range tests {
			t.Run(version.String()+"/"+tt.name, func(t *testing.T) {
				want := writeXML(t, version, tt.want())
				have := writeXML(t, version, tt.withNils())
				if want != have {
					t.Errorf("nil entries changed the output:\nwant\n%s\nhave\n%s", want, have)
				}
			})
		}
	}
}
