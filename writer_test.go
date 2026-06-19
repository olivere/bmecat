package bmecat_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/olivere/bmecat"
)

func wInt(v int) *int           { return &v }
func wFloat(v float64) *float64 { return &v }

// sliceCatalogWriter is a test CatalogWriter that streams a fixed header and
// product slice. Production code streams from its own source; this just adapts a
// slice for the tests.
type sliceCatalogWriter struct {
	header   *bmecat.Header
	products []*bmecat.Product
	err      error // if set, sent on the error channel after the products
}

func (c *sliceCatalogWriter) Header() *bmecat.Header { return c.header }

func (c *sliceCatalogWriter) Products(ctx context.Context) (<-chan *bmecat.Product, <-chan error) {
	out := make(chan *bmecat.Product)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		for _, p := range c.products {
			select {
			case out <- p:
			case <-ctx.Done():
				errc <- ctx.Err()
				return
			}
		}
		if c.err != nil {
			errc <- c.err
		}
	}()
	return out, errc
}

// fullProduct is a neutral product with every common field populated, so a
// write→read round-trip exercises the whole conversion and reflect.DeepEqual is
// meaningful (no nil-vs-empty-slice ambiguity). QuantityMax is left zero because
// BMEcat 1.2 has no QUANTITY_MAX; it is covered separately.
func fullProduct() *bmecat.Product {
	// price is shared between the flattened Prices view and the PriceDetails
	// wrapper so a write→read round trip reproduces both: the reader rebuilds
	// Prices by flattening the wrappers it reads back.
	price := &bmecat.Price{
		Type:       "net_customer",
		Amount:     9.99,
		Currency:   "EUR",
		Tax:        wFloat(0.19),
		Factor:     1,
		LowerBound: 1,
		Territory:  []string{"DE"},
	}
	return &bmecat.Product{
		ID:                      "1000",
		GTIN:                    "1234567890123",
		DescriptionShort:        "Widget",
		DescriptionLong:         "A useful widget.",
		SupplierAltID:           "ALT-1",
		ManufacturerID:          "MPN-1",
		ManufacturerName:        "Acme",
		ManufacturerTypeDescr:   "Type-X",
		ERPGroupBuyer:           "EB",
		ERPGroupSupplier:        "ES",
		DeliveryTime:            wInt(5),
		Keywords:                []string{"tool", "widget"},
		Remarks:                 "handle with care",
		Segments:                []string{"SEG1"},
		BuyerIDs:                []*bmecat.TypedValue{{Type: "buyer", Value: "B-1"}},
		SpecialTreatmentClasses: []*bmecat.TypedValue{{Type: "GGVS", Value: "12"}},
		Status:                  []*bmecat.TypedValue{{Type: "new", Value: "yes"}},
		Features: []*bmecat.Features{{
			SystemName: "ECLASS-5.1",
			GroupID:    "19010203",
			Features: []*bmecat.Feature{
				{Name: "Voltage", Values: []string{"230"}, Unit: "VLT"},
			},
		}},
		OrderUnit:        "PCE",
		ContentUnit:      "PCE",
		NoCuPerOu:        1,
		PriceQuantity:    1,
		QuantityMin:      1,
		QuantityInterval: 1,
		Prices:           []*bmecat.Price{price},
		PriceDetails:     []*bmecat.PriceDetails{{Prices: []*bmecat.Price{price}}},
		Mimes: []*bmecat.Mime{{
			Type: "image/jpeg", Source: "img.jpg", Descr: "Image", Purpose: "normal", Order: 1,
		}},
		UDX: []*bmecat.UDXField{{Name: "SYSTEM.CUSTOM_FIELD1", Value: "A"}},
	}
}

func fullHeader() *bmecat.Header {
	return &bmecat.Header{
		GeneratorInfo: "Test Generator",
		Catalog: &bmecat.Catalog{
			Language:    "deu",
			ID:          "CAT1",
			Version:     "1.0",
			Name:        "Test Catalog",
			Currency:    "EUR",
			Territories: []string{"DE", "AT"},
		},
		Buyer:    &bmecat.Buyer{ID: "BUYCO", Name: "BuyCo Inc."},
		Supplier: &bmecat.Supplier{ID: "SUPPLYCO", Name: "SupplyCo Ltd."},
	}
}

// TestWriteReadRoundTrip writes a neutral catalog to each version and reads it
// back through the neutral reader, asserting the common-field model survives the
// round trip unchanged for both 1.2 and 2005.
func TestWriteReadRoundTrip(t *testing.T) {
	for _, version := range []bmecat.Version{bmecat.Version12, bmecat.Version2005} {
		t.Run(version.String(), func(t *testing.T) {
			header := fullHeader()
			product := fullProduct()

			var buf bytes.Buffer
			cw := &sliceCatalogWriter{header: header, products: []*bmecat.Product{product}}
			if err := bmecat.NewWriter(&buf, bmecat.WithVersion(version)).Do(context.Background(), cw); err != nil {
				t.Fatal(err)
			}

			c := read(t, buf.String())

			// The reader stamps Version, Transaction and the count fields; clear
			// them so the comparison is against the data we actually wrote.
			if c.header == nil {
				t.Fatal("want header, have nil")
			}
			if want, have := version, c.header.Version; want != have {
				t.Errorf("want read-back version %v, have %v", want, have)
			}
			if want, have := bmecat.NewCatalog, c.header.Transaction; want != have {
				t.Errorf("want read-back transaction %v, have %v", want, have)
			}
			c.header.Version = 0
			c.header.Transaction = 0
			c.header.NumberOfProducts = 0
			c.header.NumberOfCatalogGroups = 0
			c.header.NumberOfClassificationGroups = 0
			if !reflect.DeepEqual(header, c.header) {
				t.Errorf("header round trip mismatch:\nwant %+v\nhave %+v", header, c.header)
			}

			if len(c.products) != 1 {
				t.Fatalf("want one product, have %d", len(c.products))
			}
			if !reflect.DeepEqual(product, c.products[0]) {
				t.Errorf("product round trip mismatch:\nwant %+v\nhave %+v", product, c.products[0])
			}
		})
	}
}

// TestWriteDefaultVersion confirms the writer emits 1.2 when no version option
// is given.
func TestWriteDefaultVersion(t *testing.T) {
	var buf bytes.Buffer
	cw := &sliceCatalogWriter{header: fullHeader()}
	if err := bmecat.NewWriter(&buf).Do(context.Background(), cw); err != nil {
		t.Fatal(err)
	}
	v, err := bmecat.NewReader(bytes.NewReader(buf.Bytes())).DetectVersion()
	if err != nil {
		t.Fatal(err)
	}
	if v != bmecat.Version12 {
		t.Errorf("want default version 1.2, have %v", v)
	}
}

// TestWriteTransaction confirms the configured transaction is emitted and reads
// back via DetectTransaction, including the prev_version attribute on updates.
func TestWriteTransaction(t *testing.T) {
	for _, version := range []bmecat.Version{bmecat.Version12, bmecat.Version2005} {
		t.Run(version.String(), func(t *testing.T) {
			var buf bytes.Buffer
			cw := &sliceCatalogWriter{header: fullHeader(), products: []*bmecat.Product{fullProduct()}}
			w := bmecat.NewWriter(
				&buf,
				bmecat.WithVersion(version),
				bmecat.WithTransaction(bmecat.UpdateProducts),
				bmecat.WithPreviousVersion(7),
			)
			if err := w.Do(context.Background(), cw); err != nil {
				t.Fatal(err)
			}

			tx, err := bmecat.NewReader(bytes.NewReader(buf.Bytes())).DetectTransaction()
			if err != nil {
				t.Fatal(err)
			}
			if tx != bmecat.UpdateProducts {
				t.Errorf("want transaction UpdateProducts, have %v", tx)
			}
			if !strings.Contains(buf.String(), `prev_version="7"`) {
				t.Errorf("want prev_version=\"7\" in output, have:\n%s", buf.String())
			}
		})
	}
}

// TestWriteQuantityMax confirms QUANTITY_MAX survives a 2005 round trip but is
// dropped for 1.2, which has no such element.
func TestWriteQuantityMax(t *testing.T) {
	tests := []struct {
		version bmecat.Version
		want    float64
	}{
		{bmecat.Version2005, 99},
		{bmecat.Version12, 0},
	}
	for _, tt := range tests {
		t.Run(tt.version.String(), func(t *testing.T) {
			p := &bmecat.Product{ID: "1", OrderUnit: "PCE", QuantityMax: 99}
			cw := &sliceCatalogWriter{products: []*bmecat.Product{p}}

			var buf bytes.Buffer
			if err := bmecat.NewWriter(&buf, bmecat.WithVersion(tt.version)).Do(context.Background(), cw); err != nil {
				t.Fatal(err)
			}

			c := read(t, buf.String())
			if len(c.products) != 1 {
				t.Fatalf("want one product, have %d", len(c.products))
			}
			if have := c.products[0].QuantityMax; have != tt.want {
				t.Errorf("want QuantityMax %v, have %v", tt.want, have)
			}
		})
	}
}

// TestWriteStreaming writes many products and reads them back, exercising the
// streaming bridge across more than one product (and the channel handoff).
func TestWriteStreaming(t *testing.T) {
	const n = 1000
	products := make([]*bmecat.Product, n)
	for i := range products {
		products[i] = &bmecat.Product{ID: fmt.Sprintf("P%04d", i), DescriptionShort: "x", OrderUnit: "PCE"}
	}
	cw := &sliceCatalogWriter{header: fullHeader(), products: products}

	var buf bytes.Buffer
	if err := bmecat.NewWriter(&buf, bmecat.WithVersion(bmecat.Version2005)).Do(context.Background(), cw); err != nil {
		t.Fatal(err)
	}

	c := read(t, buf.String())
	if len(c.products) != n {
		t.Fatalf("want %d products, have %d", n, len(c.products))
	}
	if c.products[0].ID != "P0000" || c.products[n-1].ID != fmt.Sprintf("P%04d", n-1) {
		t.Errorf("product order not preserved: first %q last %q", c.products[0].ID, c.products[n-1].ID)
	}
}

// TestWriteProducerError confirms an error from the producer's error channel is
// returned by Do.
func TestWriteProducerError(t *testing.T) {
	boom := errors.New("producer boom")
	cw := &sliceCatalogWriter{
		header:   fullHeader(),
		products: []*bmecat.Product{fullProduct()},
		err:      boom,
	}
	var buf bytes.Buffer
	err := bmecat.NewWriter(&buf).Do(context.Background(), cw)
	if !errors.Is(err, boom) {
		t.Fatalf("want producer error, have %v", err)
	}
}

// TestWriteContextCancel confirms a canceled context aborts the write.
func TestWriteContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cw := &sliceCatalogWriter{header: fullHeader(), products: []*bmecat.Product{fullProduct()}}
	var buf bytes.Buffer
	if err := bmecat.NewWriter(&buf).Do(ctx, cw); err == nil {
		t.Fatal("want error from canceled context, got nil")
	}
}

// TestWriteFuncRoundTrip writes a neutral catalog via the pull-style WriteFunc
// and reads it back, asserting the model survives unchanged for both versions —
// the same guarantee TestWriteReadRoundTrip gives the channel-based Do.
func TestWriteFuncRoundTrip(t *testing.T) {
	for _, version := range []bmecat.Version{bmecat.Version12, bmecat.Version2005} {
		t.Run(version.String(), func(t *testing.T) {
			header := fullHeader()
			product := fullProduct()

			var buf bytes.Buffer
			err := bmecat.NewWriter(&buf, bmecat.WithVersion(version)).
				WriteFunc(context.Background(), header, func(yield func(*bmecat.Product) error) error {
					return yield(product)
				})
			if err != nil {
				t.Fatal(err)
			}

			c := read(t, buf.String())
			if c.header == nil {
				t.Fatal("want header, have nil")
			}
			c.header.Version = 0
			c.header.Transaction = 0
			c.header.NumberOfProducts = 0
			c.header.NumberOfCatalogGroups = 0
			c.header.NumberOfClassificationGroups = 0
			if !reflect.DeepEqual(header, c.header) {
				t.Errorf("header round trip mismatch:\nwant %+v\nhave %+v", header, c.header)
			}
			if len(c.products) != 1 {
				t.Fatalf("want one product, have %d", len(c.products))
			}
			if !reflect.DeepEqual(product, c.products[0]) {
				t.Errorf("product round trip mismatch:\nwant %+v\nhave %+v", product, c.products[0])
			}
		})
	}
}

// TestWriteFuncStreaming streams many products through yield and reads them
// back, confirming order is preserved across the pull-style bridge.
func TestWriteFuncStreaming(t *testing.T) {
	const n = 1000
	var buf bytes.Buffer
	err := bmecat.NewWriter(&buf, bmecat.WithVersion(bmecat.Version2005)).
		WriteFunc(context.Background(), fullHeader(), func(yield func(*bmecat.Product) error) error {
			for i := range n {
				p := &bmecat.Product{ID: fmt.Sprintf("P%04d", i), DescriptionShort: "x", OrderUnit: "PCE"}
				if err := yield(p); err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}

	c := read(t, buf.String())
	if len(c.products) != n {
		t.Fatalf("want %d products, have %d", n, len(c.products))
	}
	if c.products[0].ID != "P0000" || c.products[n-1].ID != fmt.Sprintf("P%04d", n-1) {
		t.Errorf("product order not preserved: first %q last %q", c.products[0].ID, c.products[n-1].ID)
	}
}

// TestWriteFuncSkipsNil confirms a nil product yielded by the producer is
// skipped rather than written or panicked on.
func TestWriteFuncSkipsNil(t *testing.T) {
	var buf bytes.Buffer
	err := bmecat.NewWriter(&buf).
		WriteFunc(context.Background(), fullHeader(), func(yield func(*bmecat.Product) error) error {
			if err := yield(nil); err != nil {
				return err
			}
			return yield(&bmecat.Product{ID: "1", DescriptionShort: "x", OrderUnit: "PCE"})
		})
	if err != nil {
		t.Fatal(err)
	}
	c := read(t, buf.String())
	if len(c.products) != 1 {
		t.Fatalf("want one product (nil skipped), have %d", len(c.products))
	}
}

// TestWriteFuncProducerError confirms an error returned by the producer is
// returned by WriteFunc.
func TestWriteFuncProducerError(t *testing.T) {
	boom := errors.New("producer boom")
	var buf bytes.Buffer
	err := bmecat.NewWriter(&buf).
		WriteFunc(context.Background(), fullHeader(), func(yield func(*bmecat.Product) error) error {
			if err := yield(fullProduct()); err != nil {
				return err
			}
			return boom
		})
	if !errors.Is(err, boom) {
		t.Fatalf("want producer error, have %v", err)
	}
}

// TestWriteFuncContextCancel confirms a canceled context aborts the write and
// that yield observes the cancellation so the producer can stop.
func TestWriteFuncContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	err := bmecat.NewWriter(&buf).
		WriteFunc(ctx, fullHeader(), func(yield func(*bmecat.Product) error) error {
			return yield(fullProduct())
		})
	if err == nil {
		t.Fatal("want error from canceled context, got nil")
	}
}
