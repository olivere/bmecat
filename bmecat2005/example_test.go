package bmecat2005_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/olivere/bmecat/bmecat2005"
)

// productPrinter implements the bmecat2005 handler interfaces it cares about.
type productPrinter struct{}

func (productPrinter) HandleHeader(h *bmecat2005.Header) error {
	fmt.Printf("Catalog %q\n", h.Catalog.Name.Value())
	return nil
}

func (productPrinter) HandleProduct(p *bmecat2005.Product) error {
	fmt.Printf("- %s: %s (%s)\n", p.SupplierPID, p.Details.DescriptionShort.Get("eng"), p.Details.DescriptionShort.Get("deu"))
	return nil
}

// Example reads a BMEcat 2005 catalog with the version-specific reader. Text
// elements that repeat once per language come back as LocalizedStrings.
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
        <DESCRIPTION_SHORT lang="eng">Widget</DESCRIPTION_SHORT>
        <DESCRIPTION_SHORT lang="deu">Werkstück</DESCRIPTION_SHORT>
      </PRODUCT_DETAILS>
      <PRODUCT_ORDER_DETAILS><ORDER_UNIT>PCE</ORDER_UNIT></PRODUCT_ORDER_DETAILS>
    </PRODUCT>
  </T_NEW_CATALOG>
</BMECAT>`

	r := bmecat2005.NewReader(strings.NewReader(doc))
	if err := r.Do(context.Background(), productPrinter{}); err != nil {
		panic(err)
	}
	// Output:
	// Catalog "Spring Catalog"
	// - 1000: Widget (Werkstück)
}
