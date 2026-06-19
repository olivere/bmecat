package bmecat_test

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/olivere/bmecat"
	"github.com/olivere/bmecat/internal"
)

// The same logical catalog, encoded in both BMEcat versions. Reading either
// through bmecat.NewReader must produce the same neutral model.

const catalog12 = `<?xml version="1.0" encoding="UTF-8"?>
<BMECAT version="1.2">
  <HEADER>
    <GENERATOR_INFO>Test</GENERATOR_INFO>
    <CATALOG>
      <LANGUAGE>deu</LANGUAGE>
      <CATALOG_ID>CAT1</CATALOG_ID>
      <CATALOG_VERSION>1.0</CATALOG_VERSION>
      <CATALOG_NAME>Test Catalog</CATALOG_NAME>
      <CURRENCY>EUR</CURRENCY>
    </CATALOG>
    <SUPPLIER>
      <SUPPLIER_ID type="supplier">SUP</SUPPLIER_ID>
      <SUPPLIER_NAME>SupplyCo</SUPPLIER_NAME>
    </SUPPLIER>
  </HEADER>
  <T_NEW_CATALOG>
    <ARTICLE>
      <SUPPLIER_AID>1000</SUPPLIER_AID>
      <ARTICLE_DETAILS>
        <DESCRIPTION_SHORT>Widget</DESCRIPTION_SHORT>
        <DESCRIPTION_LONG>A useful widget.</DESCRIPTION_LONG>
        <EAN>1234567890123</EAN>
        <MANUFACTURER_AID>MPN-1</MANUFACTURER_AID>
        <MANUFACTURER_NAME>Acme</MANUFACTURER_NAME>
        <KEYWORD>tool</KEYWORD>
      </ARTICLE_DETAILS>
      <ARTICLE_FEATURES>
        <REFERENCE_FEATURE_SYSTEM_NAME>ECLASS-5.1</REFERENCE_FEATURE_SYSTEM_NAME>
        <REFERENCE_FEATURE_GROUP_ID>19010203</REFERENCE_FEATURE_GROUP_ID>
        <FEATURE><FNAME>Voltage</FNAME><FVALUE>230</FVALUE><FUNIT>VLT</FUNIT></FEATURE>
      </ARTICLE_FEATURES>
      <ARTICLE_ORDER_DETAILS>
        <ORDER_UNIT>PCE</ORDER_UNIT>
      </ARTICLE_ORDER_DETAILS>
      <ARTICLE_PRICE_DETAILS>
        <ARTICLE_PRICE price_type="net_customer">
          <PRICE_AMOUNT>9.99</PRICE_AMOUNT>
          <PRICE_CURRENCY>EUR</PRICE_CURRENCY>
        </ARTICLE_PRICE>
      </ARTICLE_PRICE_DETAILS>
    </ARTICLE>
  </T_NEW_CATALOG>
</BMECAT>`

const catalog2005 = `<?xml version="1.0" encoding="UTF-8"?>
<BMECAT version="2005" xmlns="http://www.bmecat.org/bmecat/2005">
  <HEADER>
    <GENERATOR_INFO>Test</GENERATOR_INFO>
    <CATALOG>
      <LANGUAGE>deu</LANGUAGE>
      <CATALOG_ID>CAT1</CATALOG_ID>
      <CATALOG_VERSION>1.0</CATALOG_VERSION>
      <CATALOG_NAME>Test Catalog</CATALOG_NAME>
      <CURRENCY>EUR</CURRENCY>
    </CATALOG>
    <SUPPLIER>
      <SUPPLIER_ID type="supplier">SUP</SUPPLIER_ID>
      <SUPPLIER_NAME>SupplyCo</SUPPLIER_NAME>
    </SUPPLIER>
  </HEADER>
  <T_NEW_CATALOG>
    <PRODUCT>
      <SUPPLIER_PID>1000</SUPPLIER_PID>
      <PRODUCT_DETAILS>
        <DESCRIPTION_SHORT>Widget</DESCRIPTION_SHORT>
        <DESCRIPTION_LONG>A useful widget.</DESCRIPTION_LONG>
        <INTERNATIONAL_PID type="gtin">1234567890123</INTERNATIONAL_PID>
        <MANUFACTURER_PID>MPN-1</MANUFACTURER_PID>
        <MANUFACTURER_NAME>Acme</MANUFACTURER_NAME>
        <KEYWORD>tool</KEYWORD>
      </PRODUCT_DETAILS>
      <PRODUCT_FEATURES>
        <REFERENCE_FEATURE_SYSTEM_NAME>ECLASS-5.1</REFERENCE_FEATURE_SYSTEM_NAME>
        <REFERENCE_FEATURE_GROUP_ID>19010203</REFERENCE_FEATURE_GROUP_ID>
        <FEATURE><FNAME>Voltage</FNAME><FVALUE>230</FVALUE><FUNIT>VLT</FUNIT></FEATURE>
      </PRODUCT_FEATURES>
      <PRODUCT_ORDER_DETAILS>
        <ORDER_UNIT>PCE</ORDER_UNIT>
      </PRODUCT_ORDER_DETAILS>
      <PRODUCT_PRICE_DETAILS>
        <PRODUCT_PRICE price_type="net_customer">
          <PRICE_AMOUNT>9.99</PRICE_AMOUNT>
          <PRICE_CURRENCY>EUR</PRICE_CURRENCY>
        </PRODUCT_PRICE>
      </PRODUCT_PRICE_DETAILS>
    </PRODUCT>
  </T_NEW_CATALOG>
</BMECAT>`

type collector struct {
	header   *bmecat.Header
	products []*bmecat.Product
}

func (c *collector) HandleHeader(h *bmecat.Header) error {
	c.header = h
	return nil
}

func (c *collector) HandleProduct(p *bmecat.Product) error {
	c.products = append(c.products, p)
	return nil
}

func read(t *testing.T, doc string) *collector {
	t.Helper()
	c := &collector{}
	r := bmecat.NewReader(bytes.NewReader([]byte(doc)))
	if err := r.Do(context.Background(), c); err != nil {
		t.Fatalf("read: %v", err)
	}
	return c
}

func TestDetectVersion(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		want bmecat.Version
	}{
		{"1.2 attribute", catalog12, bmecat.Version12},
		{"2005 attribute", catalog2005, bmecat.Version2005},
		{
			"2005 via namespace only",
			`<?xml version="1.0"?><BMECAT xmlns="http://www.bmecat.org/bmecat/2005"><HEADER></HEADER></BMECAT>`,
			bmecat.Version2005,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bmecat.NewReader(bytes.NewReader([]byte(tt.doc)))
			have, err := r.DetectVersion()
			if err != nil {
				t.Fatal(err)
			}
			if have != tt.want {
				t.Errorf("want version %v, have %v", tt.want, have)
			}
		})
	}
}

func TestDetectVersionErrors(t *testing.T) {
	tests := []struct {
		name string
		doc  string
	}{
		{"no BMECAT element", `<?xml version="1.0"?><ROOT></ROOT>`},
		{"unsupported version", `<BMECAT version="3.0"></BMECAT>`},
		{"no version, unknown namespace", `<BMECAT xmlns="http://example.com/x"></BMECAT>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bmecat.NewReader(bytes.NewReader([]byte(tt.doc)))
			if _, err := r.DetectVersion(); err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}

func TestReadEquivalence(t *testing.T) {
	c12 := read(t, catalog12)
	c2005 := read(t, catalog2005)

	// Versions differ, but every other neutral field should match.
	if want, have := bmecat.Version12, c12.header.Version; want != have {
		t.Errorf("want 1.2 header version %v, have %v", want, have)
	}
	if want, have := bmecat.Version2005, c2005.header.Version; want != have {
		t.Errorf("want 2005 header version %v, have %v", want, have)
	}
	c12.header.Version = 0
	c2005.header.Version = 0
	if !reflect.DeepEqual(c12.header, c2005.header) {
		t.Errorf("neutral headers differ:\n 1.2: %+v\n2005: %+v", c12.header, c2005.header)
	}

	if !reflect.DeepEqual(c12.products, c2005.products) {
		t.Errorf("neutral products differ:\n 1.2: %+v\n2005: %+v", c12.products[0], c2005.products[0])
	}

	// Spot-check the unified GTIN accessor and a carried-over helper.
	p := c2005.products[0]
	if want, have := "1234567890123", p.GTIN; want != have {
		t.Errorf("want GTIN %q, have %q", want, have)
	}
	if len(p.Features) != 1 || !p.Features[0].IsEclass() {
		t.Errorf("want one eCl@ss feature group, have %+v", p.Features)
	}
	if want, have := "5.1", p.Features[0].Version(); want != have {
		t.Errorf("want feature system version %q, have %q", want, have)
	}
}

func TestReadHeaderCounts(t *testing.T) {
	c := read(t, catalog2005)
	if c.header == nil {
		t.Fatal("want header, have nil")
	}
	if want, have := 1, c.header.NumberOfProducts; want != have {
		t.Errorf("want NumberOfProducts %d, have %d", want, have)
	}
}

var errBoom = errors.New("boom")

type failingHeaderHandler struct{}

func (failingHeaderHandler) HandleHeader(*bmecat.Header) error { return errBoom }

// TestReadHeaderHandlerError ensures a non-EOF error from HandleHeader is
// surfaced for both versions, including the 1.2 path where the underlying
// reader would otherwise swallow it (#16).
func TestReadHeaderHandlerError(t *testing.T) {
	for _, tt := range []struct {
		name string
		doc  string
	}{
		{"1.2", catalog12},
		{"2005", catalog2005},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := bmecat.NewReader(bytes.NewReader([]byte(tt.doc)))
			err := r.Do(context.Background(), failingHeaderHandler{})
			if !errors.Is(err, errBoom) {
				t.Fatalf("want wrapped errBoom, have %v", err)
			}
		})
	}
}

// TestReadTaxDetails confirms a 2005 price expressed via TAX_DETAILS surfaces
// as the neutral Price.Tax.
func TestReadTaxDetails(t *testing.T) {
	const doc = `<?xml version="1.0"?>
<BMECAT version="2005" xmlns="http://www.bmecat.org/bmecat/2005">
  <HEADER><CATALOG><LANGUAGE>deu</LANGUAGE><CATALOG_ID>C</CATALOG_ID><CATALOG_VERSION>1</CATALOG_VERSION></CATALOG></HEADER>
  <T_NEW_CATALOG>
    <PRODUCT>
      <SUPPLIER_PID>1</SUPPLIER_PID>
      <PRODUCT_DETAILS><DESCRIPTION_SHORT>X</DESCRIPTION_SHORT></PRODUCT_DETAILS>
      <PRODUCT_ORDER_DETAILS><ORDER_UNIT>PCE</ORDER_UNIT></PRODUCT_ORDER_DETAILS>
      <PRODUCT_PRICE_DETAILS>
        <PRODUCT_PRICE price_type="net_customer">
          <PRICE_AMOUNT>10</PRICE_AMOUNT>
          <TAX_DETAILS><TAX_TYPE>vat</TAX_TYPE><TAX>0.19</TAX></TAX_DETAILS>
        </PRODUCT_PRICE>
      </PRODUCT_PRICE_DETAILS>
    </PRODUCT>
  </T_NEW_CATALOG>
</BMECAT>`

	c := read(t, doc)
	if len(c.products) != 1 || len(c.products[0].Prices) != 1 {
		t.Fatalf("want one product with one price, have %+v", c.products)
	}
	tax := c.products[0].Prices[0].Tax
	if tax == nil {
		t.Fatal("want Tax from TAX_DETAILS, have nil")
	}
	if want, have := 0.19, *tax; want != have {
		t.Errorf("want tax %v, have %v", want, have)
	}
}

// TestReadProductMode confirms the product mode attribute carries into the
// neutral model for both versions.
func TestReadProductMode(t *testing.T) {
	docs := map[string]string{
		"1.2":  `<BMECAT version="1.2"><HEADER><CATALOG><LANGUAGE>deu</LANGUAGE><CATALOG_ID>C</CATALOG_ID><CATALOG_VERSION>1</CATALOG_VERSION></CATALOG></HEADER><T_UPDATE_PRODUCTS prev_version="1"><ARTICLE mode="delete"><SUPPLIER_AID>1</SUPPLIER_AID></ARTICLE></T_UPDATE_PRODUCTS></BMECAT>`,
		"2005": `<BMECAT version="2005" xmlns="http://www.bmecat.org/bmecat/2005"><HEADER><CATALOG><LANGUAGE>deu</LANGUAGE><CATALOG_ID>C</CATALOG_ID><CATALOG_VERSION>1</CATALOG_VERSION></CATALOG></HEADER><T_UPDATE_PRODUCTS prev_version="1"><PRODUCT mode="delete"><SUPPLIER_PID>1</SUPPLIER_PID></PRODUCT></T_UPDATE_PRODUCTS></BMECAT>`,
	}
	for name, doc := range docs {
		t.Run(name, func(t *testing.T) {
			c := read(t, doc)
			if len(c.products) != 1 {
				t.Fatalf("want one product, have %d", len(c.products))
			}
			if want, have := "delete", c.products[0].Mode; want != have {
				t.Errorf("want mode %q, have %q", want, have)
			}
		})
	}
}

// TestReadCharset confirms the charset option is forwarded to the underlying
// version reader.
func TestReadCharset(t *testing.T) {
	iso := "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>" +
		"<BMECAT version=\"1.2\"><HEADER><CATALOG>" +
		"<LANGUAGE>deu</LANGUAGE><CATALOG_ID>C</CATALOG_ID>" +
		"<CATALOG_VERSION>1</CATALOG_VERSION>" +
		"<CATALOG_NAME>M\xfcller</CATALOG_NAME></CATALOG></HEADER>" +
		"<T_NEW_CATALOG></T_NEW_CATALOG></BMECAT>"

	c := &collector{}
	r := bmecat.NewReader(
		bytes.NewReader([]byte(iso)),
		bmecat.WithCharsetReader(internal.AutoCharsetReader),
	)
	if err := r.Do(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if c.header == nil || c.header.Catalog == nil {
		t.Fatal("want catalog header, have nil")
	}
	if want, have := "Müller", c.header.Catalog.Name; want != have {
		t.Errorf("want decoded catalog name %q, have %q", want, have)
	}
}

// updateDoc builds a minimal catalog wrapped in the given transaction element
// for the given version, so the transaction tests can exercise every
// (version, transaction) combination.
func updateDoc(version, txElem string) string {
	switch version {
	case "2005":
		return `<BMECAT version="2005" xmlns="http://www.bmecat.org/bmecat/2005"><HEADER><CATALOG><LANGUAGE>deu</LANGUAGE><CATALOG_ID>C</CATALOG_ID><CATALOG_VERSION>1</CATALOG_VERSION></CATALOG></HEADER><` + txElem + ` prev_version="1"><PRODUCT><SUPPLIER_PID>1</SUPPLIER_PID></PRODUCT></` + txElem + `></BMECAT>`
	default:
		return `<BMECAT version="1.2"><HEADER><CATALOG><LANGUAGE>deu</LANGUAGE><CATALOG_ID>C</CATALOG_ID><CATALOG_VERSION>1</CATALOG_VERSION></CATALOG></HEADER><` + txElem + ` prev_version="1"><ARTICLE><SUPPLIER_AID>1</SUPPLIER_AID></ARTICLE></` + txElem + `></BMECAT>`
	}
}

// TestDetectTransaction covers option 1 from #29: the phase-1 detector reports
// the wrapping transaction for both versions and seeks back so a later Do still
// reads the whole document.
func TestDetectTransaction(t *testing.T) {
	tests := []struct {
		txElem string
		want   bmecat.Transaction
	}{
		{"T_NEW_CATALOG", bmecat.NewCatalog},
		{"T_UPDATE_PRODUCTS", bmecat.UpdateProducts},
		{"T_UPDATE_PRICES", bmecat.UpdatePrices},
	}
	for _, version := range []string{"1.2", "2005"} {
		for _, tt := range tests {
			t.Run(version+"/"+tt.txElem, func(t *testing.T) {
				r := bmecat.NewReader(bytes.NewReader([]byte(updateDoc(version, tt.txElem))))
				have, err := r.DetectTransaction()
				if err != nil {
					t.Fatal(err)
				}
				if have != tt.want {
					t.Fatalf("want transaction %v, have %v", tt.want, have)
				}
				// DetectTransaction must rewind: a subsequent Do still sees the product.
				c := &collector{}
				if err := r.Do(context.Background(), c); err != nil {
					t.Fatal(err)
				}
				if len(c.products) != 1 {
					t.Errorf("want one product after detect+read, have %d", len(c.products))
				}
			})
		}
	}
}

// TestDetectTransactionErrors confirms a document without a transaction element
// reports an error rather than a bogus zero value.
func TestDetectTransactionErrors(t *testing.T) {
	tests := []struct {
		name string
		doc  string
	}{
		{"no BMECAT element", `<?xml version="1.0"?><ROOT></ROOT>`},
		{"header only, no transaction", `<BMECAT version="1.2"><HEADER></HEADER></BMECAT>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bmecat.NewReader(bytes.NewReader([]byte(tt.doc)))
			if _, err := r.DetectTransaction(); err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}

// TestReadHeaderTransaction covers option 2 from #29: the transaction is
// surfaced on the neutral Header during Do, for both versions.
func TestReadHeaderTransaction(t *testing.T) {
	for _, version := range []string{"1.2", "2005"} {
		t.Run(version, func(t *testing.T) {
			c := read(t, updateDoc(version, "T_UPDATE_PRICES"))
			if c.header == nil {
				t.Fatal("want header, have nil")
			}
			if want, have := bmecat.UpdatePrices, c.header.Transaction; want != have {
				t.Errorf("want Header.Transaction %v, have %v", want, have)
			}
			if !c.header.Transaction.IsUpdate() {
				t.Error("want IsUpdate true for T_UPDATE_PRICES")
			}
		})
	}
}

// TestTransactionString pins the element names a Transaction stringifies to.
func TestTransactionString(t *testing.T) {
	for tx, want := range map[bmecat.Transaction]string{
		bmecat.NewCatalog:     "T_NEW_CATALOG",
		bmecat.UpdateProducts: "T_UPDATE_PRODUCTS",
		bmecat.UpdatePrices:   "T_UPDATE_PRICES",
	} {
		if have := tx.String(); have != want {
			t.Errorf("want %q, have %q", want, have)
		}
	}
}
