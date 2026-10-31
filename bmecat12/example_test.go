package bmecat12_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/olivere/bmecat/bmecat12"
)

// articlePrinter implements the bmecat12 handler interfaces it cares about.
type articlePrinter struct{}

func (articlePrinter) HandleHeader(h *bmecat12.Header) error {
	fmt.Printf("Catalog %q\n", h.Catalog.Name)
	return nil
}

func (articlePrinter) HandleArticle(a *bmecat12.Article) error {
	fmt.Printf("- %s: %s (EAN %s)\n", a.SupplierAID, a.Details.DescriptionShort, a.Details.EAN)
	return nil
}

// Example reads a BMEcat 1.2 catalog with the version-specific reader, which
// exposes the 1.2 elements as they appear in the document.
func Example() {
	const doc = `<?xml version="1.0" encoding="UTF-8"?>
<BMECAT version="1.2" xmlns="http://www.bmecat.org/bmecat/1.2/bmecat_new_catalog">
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
    <ARTICLE>
      <SUPPLIER_AID>1000</SUPPLIER_AID>
      <ARTICLE_DETAILS>
        <DESCRIPTION_SHORT>Widget</DESCRIPTION_SHORT>
        <EAN>1234567890123</EAN>
      </ARTICLE_DETAILS>
      <ARTICLE_ORDER_DETAILS><ORDER_UNIT>PCE</ORDER_UNIT></ARTICLE_ORDER_DETAILS>
    </ARTICLE>
  </T_NEW_CATALOG>
</BMECAT>`

	r := bmecat12.NewReader(strings.NewReader(doc))
	if err := r.Do(context.Background(), articlePrinter{}); err != nil {
		panic(err)
	}
	// Output:
	// Catalog "Spring Catalog"
	// - 1000: Widget (EAN 1234567890123)
}
