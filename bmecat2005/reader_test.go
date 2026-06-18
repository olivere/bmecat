package bmecat2005_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/olivere/bmecat/bmecat2005"
)

type testHandler struct {
	firstPassOnly bool
	header        *bmecat2005.Header
	products      []*bmecat2005.Product
}

func (h *testHandler) HandleHeader(header *bmecat2005.Header) error {
	h.header = header
	if h.firstPassOnly {
		return io.EOF
	}
	return nil
}

func (h *testHandler) HandleProduct(product *bmecat2005.Product) error {
	h.products = append(h.products, product)
	return nil
}

func TestReadCatalog(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "new_catalog.golden.xml"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	h := &testHandler{}
	r := bmecat2005.NewReader(f)
	err = r.Do(context.Background(), h)
	if err != nil {
		t.Fatal(err)
	}
	if h.header == nil {
		t.Fatal("want Header, have nil")
	}
	if want, have := 1, h.header.NumberOfProducts; want != have {
		t.Fatalf("want NumberOfProducts = %d, have %d", want, have)
	}
	if want, have := 1, len(h.products); want != have {
		t.Fatalf("want len(products) = %d, have %d", want, have)
	}
	p := h.products[0]
	if want, have := "1000", p.SupplierPID; want != have {
		t.Errorf("want SupplierPID %q, have %q", want, have)
	}
	if p.Details == nil || len(p.Details.InternationalPIDs) != 1 {
		t.Fatalf("want one INTERNATIONAL_PID, have %#v", p.Details)
	}
	if want, have := "gtin", p.Details.InternationalPIDs[0].Type; want != have {
		t.Errorf("want INTERNATIONAL_PID type %q, have %q", want, have)
	}
}

func BenchmarkReader(b *testing.B) {
	b.ReportAllocs()

	buf, err := os.ReadFile(filepath.Join("testdata", "new_catalog.golden.xml"))
	if err != nil {
		b.Fatal(err)
	}
	buffer := strings.NewReader(string(buf))

	for i := 0; i < b.N; i++ {
		if _, err := buffer.Seek(0, io.SeekStart); err != nil {
			b.Fatal(err)
		}

		h := &testHandler{}
		r := bmecat2005.NewReader(buffer)
		err = r.Do(context.Background(), h)
		if err != nil {
			b.Fatal(err)
		}
		if h.header == nil {
			b.Fatal("want Header, have nil")
		}
		if want, have := 1, len(h.products); want != have {
			b.Fatalf("want len(products) = %d, have %d", want, have)
		}
	}
}
