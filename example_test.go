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
