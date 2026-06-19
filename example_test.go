package bmecat_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/olivere/bmecat"
)

// catalogPrinter implements the neutral handler interfaces it cares about.
// The same handler works for BMEcat 1.2 and 2005 input.
type catalogPrinter struct{}

func (catalogPrinter) HandleHeader(h *bmecat.Header) error {
	fmt.Printf("Catalog %q (BMEcat %s)\n", h.Catalog.Name, h.Version)
	return nil
}

func (catalogPrinter) HandleProduct(p *bmecat.Product) error {
	fmt.Printf("- %s: %s (GTIN %s)\n", p.ID, p.DescriptionShort, p.GTIN)
	return nil
}

// Example reads a catalog through the version-neutral facade. bmecat.NewReader
// auto-detects the version, so the same code ingests 1.2 and 2005 documents.
func Example() {
	const doc = `<?xml version="1.0" encoding="UTF-8"?>
<BMECAT version="2005" xmlns="http://www.bmecat.org/bmecat/2005">
  <HEADER>
    <CATALOG>
      <LANGUAGE>deu</LANGUAGE>
      <CATALOG_ID>CAT1</CATALOG_ID>
      <CATALOG_VERSION>1.0</CATALOG_VERSION>
      <CATALOG_NAME>Spring Catalog</CATALOG_NAME>
    </CATALOG>
    <SUPPLIER><SUPPLIER_NAME>SupplyCo</SUPPLIER_NAME></SUPPLIER>
  </HEADER>
  <T_NEW_CATALOG>
    <PRODUCT>
      <SUPPLIER_PID>1000</SUPPLIER_PID>
      <PRODUCT_DETAILS>
        <DESCRIPTION_SHORT>Widget</DESCRIPTION_SHORT>
        <INTERNATIONAL_PID type="gtin">1234567890123</INTERNATIONAL_PID>
      </PRODUCT_DETAILS>
      <PRODUCT_ORDER_DETAILS><ORDER_UNIT>PCE</ORDER_UNIT></PRODUCT_ORDER_DETAILS>
      <PRODUCT_PRICE_DETAILS>
        <PRODUCT_PRICE price_type="net_customer"><PRICE_AMOUNT>9.99</PRICE_AMOUNT></PRODUCT_PRICE>
      </PRODUCT_PRICE_DETAILS>
    </PRODUCT>
  </T_NEW_CATALOG>
</BMECAT>`

	r := bmecat.NewReader(strings.NewReader(doc))
	if err := r.Do(context.Background(), catalogPrinter{}); err != nil {
		panic(err)
	}
	// Output:
	// Catalog "Spring Catalog" (BMEcat 2005)
	// - 1000: Widget (GTIN 1234567890123)
}

// ExampleReader_DetectVersion shows how to detect a document's BMEcat version
// without consuming it.
func ExampleReader_DetectVersion() {
	const doc = `<BMECAT version="1.2"><HEADER></HEADER><T_NEW_CATALOG></T_NEW_CATALOG></BMECAT>`

	r := bmecat.NewReader(strings.NewReader(doc))
	v, err := r.DetectVersion()
	if err != nil {
		panic(err)
	}
	fmt.Printf("version %s\n", v)
	// Output:
	// version 1.2
}

// springCatalog is a neutral CatalogWriter: it supplies the header and streams
// the products. A real implementation would stream from a database or file so
// the whole catalog is never held in memory; here it ranges over a slice.
type springCatalog struct{}

func (springCatalog) Header() *bmecat.Header {
	return &bmecat.Header{
		Catalog: &bmecat.Catalog{
			Language: "deu",
			ID:       "CAT1",
			Version:  "1.0",
			Name:     "Spring Catalog",
		},
		Supplier: &bmecat.Supplier{Name: "SupplyCo"},
	}
}

func (springCatalog) Products(ctx context.Context) (<-chan *bmecat.Product, <-chan error) {
	products := []*bmecat.Product{
		{
			ID:               "1000",
			GTIN:             "1234567890123",
			DescriptionShort: "Widget",
			OrderUnit:        "PCE",
			Prices: []*bmecat.Price{
				{Type: "net_customer", Amount: 9.99, Currency: "EUR"},
			},
		},
	}
	out := make(chan *bmecat.Product)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		for _, p := range products {
			select {
			case out <- p:
			case <-ctx.Done():
				errc <- ctx.Err()
				return
			}
		}
	}()
	return out, errc
}

// ExampleWriter writes a neutral catalog as BMEcat 2005. The same CatalogWriter
// would produce a 1.2 document with bmecat.WithVersion(bmecat.Version12).
func ExampleWriter() {
	var buf strings.Builder
	w := bmecat.NewWriter(&buf, bmecat.WithVersion(bmecat.Version2005))
	if err := w.Do(context.Background(), springCatalog{}); err != nil {
		panic(err)
	}

	// Read it straight back through the neutral reader to show it round-trips.
	if err := bmecat.NewReader(strings.NewReader(buf.String())).Do(context.Background(), catalogPrinter{}); err != nil {
		panic(err)
	}
	// Output:
	// Catalog "Spring Catalog" (BMEcat 2005)
	// - 1000: Widget (GTIN 1234567890123)
}

// ExampleWriter_writeFunc writes the same catalog with the pull-style WriteFunc:
// the producer streams products by calling yield, with no channels to manage.
func ExampleWriter_writeFunc() {
	header := &bmecat.Header{
		Catalog:  &bmecat.Catalog{Language: "deu", ID: "CAT1", Version: "1.0", Name: "Spring Catalog"},
		Supplier: &bmecat.Supplier{Name: "SupplyCo"},
	}
	products := []*bmecat.Product{
		{
			ID:               "1000",
			GTIN:             "1234567890123",
			DescriptionShort: "Widget",
			OrderUnit:        "PCE",
			Prices:           []*bmecat.Price{{Type: "net_customer", Amount: 9.99, Currency: "EUR"}},
		},
	}

	var buf strings.Builder
	w := bmecat.NewWriter(&buf, bmecat.WithVersion(bmecat.Version2005))
	err := w.WriteFunc(context.Background(), header, func(yield func(*bmecat.Product) error) error {
		for _, p := range products {
			if err := yield(p); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	if err := bmecat.NewReader(strings.NewReader(buf.String())).Do(context.Background(), catalogPrinter{}); err != nil {
		panic(err)
	}
	// Output:
	// Catalog "Spring Catalog" (BMEcat 2005)
	// - 1000: Widget (GTIN 1234567890123)
}
