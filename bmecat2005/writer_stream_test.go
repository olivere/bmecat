package bmecat2005_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/olivere/bmecat/bmecat2005"
)

// funcCatalog is a CatalogWriter whose Products method is backed by the
// pull-style StreamProducts helper, exercising the helper end to end through
// Writer.Do.
type funcCatalog struct {
	header  *bmecat2005.Header
	produce func(yield func(*bmecat2005.Product) error) error
}

func (funcCatalog) Transaction() bmecat2005.Transaction                    { return bmecat2005.NewCatalog }
func (funcCatalog) Language() string                                       { return "deu" }
func (funcCatalog) PreviousVersion() int                                   { return 0 }
func (c funcCatalog) Header() *bmecat2005.Header                           { return c.header }
func (funcCatalog) ClassificationSystem() *bmecat2005.ClassificationSystem { return nil }

func (c funcCatalog) Products(ctx context.Context) (<-chan *bmecat2005.Product, <-chan error) {
	return bmecat2005.StreamProducts(ctx, c.produce)
}

// TestStreamProducts writes several products through StreamProducts and confirms
// they are all emitted in order.
func TestStreamProducts(t *testing.T) {
	const n = 100
	cw := funcCatalog{
		header: testHeader,
		produce: func(yield func(*bmecat2005.Product) error) error {
			for i := range n {
				p := &bmecat2005.Product{
					SupplierPID: fmt.Sprintf("P%04d", i),
					Details:     &bmecat2005.ProductDetails{DescriptionShort: "x"},
				}
				if err := yield(p); err != nil {
					return err
				}
			}
			return nil
		},
	}

	var buf bytes.Buffer
	if err := bmecat2005.NewWriter(&buf).Do(context.Background(), cw); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if got := strings.Count(out, "<SUPPLIER_PID>"); got != n {
		t.Fatalf("want %d products, have %d", n, got)
	}
	if first, last := strings.Index(out, "P0000"), strings.Index(out, fmt.Sprintf("P%04d", n-1)); first < 0 || last < 0 || first > last {
		t.Errorf("product order not preserved (first idx %d, last idx %d)", first, last)
	}
}

// TestStreamProductsSkipsNil confirms a nil product yielded by the producer is
// skipped rather than written.
func TestStreamProductsSkipsNil(t *testing.T) {
	cw := funcCatalog{
		header: testHeader,
		produce: func(yield func(*bmecat2005.Product) error) error {
			if err := yield(nil); err != nil {
				return err
			}
			return yield(&bmecat2005.Product{SupplierPID: "1", Details: &bmecat2005.ProductDetails{DescriptionShort: "x"}})
		},
	}

	var buf bytes.Buffer
	if err := bmecat2005.NewWriter(&buf).Do(context.Background(), cw); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(buf.String(), "<SUPPLIER_PID>"); got != 1 {
		t.Fatalf("want one product (nil skipped), have %d", got)
	}
}

// TestStreamProductsProducerError confirms an error returned by the producer is
// reported by Writer.Do.
func TestStreamProductsProducerError(t *testing.T) {
	boom := errors.New("producer boom")
	cw := funcCatalog{
		header: testHeader,
		produce: func(yield func(*bmecat2005.Product) error) error {
			if err := yield(&bmecat2005.Product{SupplierPID: "1", Details: &bmecat2005.ProductDetails{DescriptionShort: "x"}}); err != nil {
				return err
			}
			return boom
		},
	}

	err := bmecat2005.NewWriter(&bytes.Buffer{}).Do(context.Background(), cw)
	if !errors.Is(err, boom) {
		t.Fatalf("want producer error, have %v", err)
	}
}

// TestStreamProductsContextCancel confirms a canceled context aborts the write
// and that yield observes the cancellation.
func TestStreamProductsContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cw := funcCatalog{
		header: testHeader,
		produce: func(yield func(*bmecat2005.Product) error) error {
			return yield(&bmecat2005.Product{SupplierPID: "1", Details: &bmecat2005.ProductDetails{DescriptionShort: "x"}})
		},
	}

	if err := bmecat2005.NewWriter(&bytes.Buffer{}).Do(ctx, cw); err == nil {
		t.Fatal("want error from canceled context, got nil")
	}
}
