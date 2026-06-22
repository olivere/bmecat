package bmecat_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/olivere/bmecat"
)

// The benchmarks below stream a synthetic multi-product catalog through the
// neutral facade in both directions and both versions, so they measure the
// per-product conversion hot path (adapter_v12/adapter_v2005 on read,
// writer_v12/writer_v2005 on write) rather than the fixed per-document cost the
// tiny golden files in testdata would dominate.

const benchProductCount = 1000

// benchProduct builds one content-rich neutral product: several buyer IDs,
// statuses, price tiers, features and MIMEs, like a real industrial-catalog
// entry rather than the single-valued fixtures used by the round-trip tests.
// The multi-valued slices are what make the converters' allocation behavior
// (and any preallocation) observable in the benchmarks.
func benchProduct() *bmecat.Product {
	p := &bmecat.Product{
		ID:                      "1000",
		GTIN:                    "1234567890123",
		DescriptionShort:        bmecat.Localized("Widget"),
		DescriptionLong:         bmecat.Localized("A useful widget with a longer description."),
		ManufacturerID:          "MPN-1",
		ManufacturerName:        "Acme",
		Keywords:                bmecat.Localized("tool", "widget", "hardware", "industrial"),
		BuyerIDs:                []*bmecat.TypedValue{{Type: "buyer", Value: "B-1"}, {Type: "buyer", Value: "B-2"}},
		SpecialTreatmentClasses: []*bmecat.TypedValue{{Type: "GGVS", Value: "12"}, {Type: "WEEE", Value: "34"}},
		Status:                  []*bmecat.TypedValue{{Type: "new", Value: "yes"}, {Type: "bestseller", Value: "yes"}},
		OrderUnit:               "PCE",
		ContentUnit:             "PCE",
		NoCuPerOu:               1,
	}
	for range 4 {
		feat := &bmecat.Features{SystemName: "ECLASS-5.1", GroupID: "19010203"}
		for range 4 {
			feat.Features = append(feat.Features, &bmecat.Feature{
				Name:   bmecat.Localized("Voltage"),
				Values: bmecat.Localized("230", "240"),
				Unit:   "VLT",
			})
		}
		p.Features = append(p.Features, feat)
	}
	for range 3 {
		pd := &bmecat.PriceDetails{}
		for t := range 3 {
			pd.Prices = append(pd.Prices, &bmecat.Price{
				Type:       "net_customer",
				Amount:     9.99,
				Currency:   "EUR",
				Factor:     1,
				LowerBound: float64(t),
				Territory:  []string{"DE"},
			})
		}
		p.PriceDetails = append(p.PriceDetails, pd)
		p.Prices = append(p.Prices, pd.Prices...)
	}
	for m := range 4 {
		p.Mimes = append(p.Mimes, &bmecat.Mime{
			Type:    "image/jpeg",
			Source:  bmecat.Localized("image.jpg"),
			Purpose: "normal",
			Order:   m,
		})
	}
	for range 5 {
		p.UDX = append(p.UDX, &bmecat.UDXField{Name: "SYSTEM.FIELD", Value: "v"})
	}
	return p
}

// benchProducts builds n content-rich neutral products for the streaming
// benchmarks, so the per-product conversion hot path is what is measured.
func benchProducts(n int) []*bmecat.Product {
	products := make([]*bmecat.Product, n)
	for i := range products {
		products[i] = benchProduct()
	}
	return products
}

func benchHeader() *bmecat.Header {
	return &bmecat.Header{
		GeneratorInfo: "bench",
		Catalog: &bmecat.Catalog{
			Language: "deu",
			ID:       "CAT1",
			Version:  "1.0",
			Name:     bmecat.Localized("Bench"),
			Currency: "EUR",
		},
		Supplier: &bmecat.Supplier{ID: "SUP", Name: "SupplyCo"},
	}
}

// countingHandler discards every callback but counts products, so the read
// benchmark measures parsing and conversion without any handler-side work.
type countingHandler struct{ products int }

func (h *countingHandler) HandleHeader(*bmecat.Header) error   { return nil }
func (h *countingHandler) HandleProduct(*bmecat.Product) error { h.products++; return nil }

func benchmarkWrite(b *testing.B, version bmecat.Version) {
	cw := &sliceCatalogWriter{header: benchHeader(), products: benchProducts(benchProductCount)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := bmecat.NewWriter(io.Discard, bmecat.WithVersion(version))
		if err := w.Do(context.Background(), cw); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWrite12(b *testing.B)   { benchmarkWrite(b, bmecat.Version12) }
func BenchmarkWrite2005(b *testing.B) { benchmarkWrite(b, bmecat.Version2005) }

func benchmarkRead(b *testing.B, version bmecat.Version) {
	var buf bytes.Buffer
	cw := &sliceCatalogWriter{header: benchHeader(), products: benchProducts(benchProductCount)}
	if err := bmecat.NewWriter(&buf, bmecat.WithVersion(version)).Do(context.Background(), cw); err != nil {
		b.Fatal(err)
	}
	data := buf.Bytes()
	src := bytes.NewReader(data)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := src.Seek(0, io.SeekStart); err != nil {
			b.Fatal(err)
		}
		h := &countingHandler{}
		if err := bmecat.NewReader(src).Do(context.Background(), h); err != nil {
			b.Fatal(err)
		}
		if h.products != benchProductCount {
			b.Fatalf("read %d products, want %d", h.products, benchProductCount)
		}
	}
}

func BenchmarkRead12(b *testing.B)   { benchmarkRead(b, bmecat.Version12) }
func BenchmarkRead2005(b *testing.B) { benchmarkRead(b, bmecat.Version2005) }
