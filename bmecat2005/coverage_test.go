package bmecat2005_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"

	"github.com/olivere/bmecat/bmecat2005"
	"github.com/olivere/bmecat/internal"
)

// errBoom is a sentinel error used to assert that the reader and writer wrap
// and propagate handler errors via %w.
var errBoom = errors.New("boom")

// --- Reader error paths -----------------------------------------------------

func TestReadMalformedXML(t *testing.T) {
	// Mismatched end tag: the tokenizer must fail in the first pass.
	const malformed = `<BMECAT><HEADER></BMECAT>`
	r := bmecat2005.NewReader(bytes.NewReader([]byte(malformed)))
	if err := r.Do(context.Background(), &testHandler{}); err == nil {
		t.Fatal("expected an error for malformed XML, got nil")
	}
}

type failingProductHandler struct{}

func (failingProductHandler) HandleProduct(*bmecat2005.Product) error { return errBoom }

func TestReadProductHandlerError(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "new_catalog.golden.xml"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r := bmecat2005.NewReader(f)
	err = r.Do(context.Background(), failingProductHandler{})
	if !errors.Is(err, errBoom) {
		t.Fatalf("expected wrapped errBoom, got %v", err)
	}
}

type failingHeaderHandler struct{}

func (failingHeaderHandler) HandleHeader(*bmecat2005.Header) error { return errBoom }

func TestReadHeaderHandlerError(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "new_catalog.golden.xml"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r := bmecat2005.NewReader(f)
	err = r.Do(context.Background(), failingHeaderHandler{})
	if !errors.Is(err, errBoom) {
		t.Fatalf("expected wrapped errBoom, got %v", err)
	}
}

// --- Reader group dispatch and completion -----------------------------------

type recordingHandler struct {
	classGroups   int
	catalogGroups []*bmecat2005.CatalogGroup
	completed     int
}

func (h *recordingHandler) HandleClassificationGroup(*bmecat2005.ClassificationGroup) error {
	h.classGroups++
	return nil
}

func (h *recordingHandler) HandleCatalogGroup(cg *bmecat2005.CatalogGroup) error {
	h.catalogGroups = append(h.catalogGroups, cg)
	return nil
}

func (h *recordingHandler) HandleComplete() { h.completed++ }

func TestReadClassificationGroupsAndComplete(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "new_catalog.golden.xml"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	h := &recordingHandler{}
	r := bmecat2005.NewReader(f)
	if err := r.Do(context.Background(), h); err != nil {
		t.Fatal(err)
	}
	if want, have := 5, h.classGroups; want != have {
		t.Errorf("want %d classification groups, have %d", want, have)
	}
	if want, have := 1, h.completed; want != have {
		t.Errorf("want HandleComplete called %d time(s), have %d", want, have)
	}
}

func TestReadCatalogGroupDispatch(t *testing.T) {
	// No golden file contains CATALOG_STRUCTURE, so drive it with inline XML.
	const xml = `<?xml version="1.0" encoding="UTF-8"?>
<BMECAT version="2005">
  <T_NEW_CATALOG>
    <CATALOG_STRUCTURE type="node">
      <GROUP_ID>1</GROUP_ID>
      <GROUP_NAME>Hardware</GROUP_NAME>
    </CATALOG_STRUCTURE>
  </T_NEW_CATALOG>
</BMECAT>`

	h := &recordingHandler{}
	r := bmecat2005.NewReader(bytes.NewReader([]byte(xml)))
	if err := r.Do(context.Background(), h); err != nil {
		t.Fatal(err)
	}
	if want, have := 1, len(h.catalogGroups); want != have {
		t.Fatalf("want %d catalog group(s), have %d", want, have)
	}
	cg := h.catalogGroups[0]
	if want, have := "1", cg.ID; want != have {
		t.Errorf("want group ID %q, have %q", want, have)
	}
	if want, have := "Hardware", cg.Name; want != have {
		t.Errorf("want group name %q, have %q", want, have)
	}
	if !cg.IsNode() {
		t.Errorf("want IsNode() == true for type %q", cg.Type)
	}
}

// --- Product to catalog group mapping ---------------------------------------

func TestReadProductToCatalogGroupMap(t *testing.T) {
	const doc = `<?xml version="1.0" encoding="UTF-8"?>
<BMECAT version="2005">
  <T_NEW_CATALOG>
    <PRODUCT>
      <SUPPLIER_PID>1000</SUPPLIER_PID>
      <PRODUCT_DETAILS><DESCRIPTION_SHORT>Test</DESCRIPTION_SHORT></PRODUCT_DETAILS>
      <PRODUCT_ORDER_DETAILS><ORDER_UNIT>PCE</ORDER_UNIT></PRODUCT_ORDER_DETAILS>
      <PRODUCT_PRICE_DETAILS><PRODUCT_PRICE price_type="net_customer"><PRICE_AMOUNT>1</PRICE_AMOUNT></PRODUCT_PRICE></PRODUCT_PRICE_DETAILS>
    </PRODUCT>
    <PRODUCT_TO_CATALOGGROUP_MAP>
      <PROD_ID>1000</PROD_ID>
      <CATALOG_GROUP_ID>5</CATALOG_GROUP_ID>
    </PRODUCT_TO_CATALOGGROUP_MAP>
  </T_NEW_CATALOG>
</BMECAT>`

	h := &testHandler{}
	r := bmecat2005.NewReader(bytes.NewReader([]byte(doc)))
	if err := r.Do(context.Background(), h); err != nil {
		t.Fatal(err)
	}
	if want, have := 1, len(h.products); want != have {
		t.Fatalf("want %d product(s), have %d", want, have)
	}
	if want, have := []string{"5"}, h.products[0].CatalogGroupIDs; len(want) != len(have) || (len(have) == 1 && have[0] != want[0]) {
		t.Errorf("want CatalogGroupIDs %v, have %v", want, have)
	}
}

// --- Charset handling -------------------------------------------------------

func TestReadISO8859_1(t *testing.T) {
	// GROUP_NAME contains 0xFC, which is "ü" in ISO-8859-1. The reader must
	// decode it to UTF-8 via the (explicitly supplied) charset reader.
	iso := "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>" +
		"<BMECAT version=\"2005\"><T_NEW_CATALOG>" +
		"<CATALOG_STRUCTURE type=\"leaf\"><GROUP_ID>1</GROUP_ID>" +
		"<GROUP_NAME>M\xfcller</GROUP_NAME></CATALOG_STRUCTURE>" +
		"</T_NEW_CATALOG></BMECAT>"

	h := &recordingHandler{}
	r := bmecat2005.NewReader(
		bytes.NewReader([]byte(iso)),
		bmecat2005.WithCharsetReader(internal.AutoCharsetReader),
	)
	if err := r.Do(context.Background(), h); err != nil {
		t.Fatal(err)
	}
	if want, have := 1, len(h.catalogGroups); want != have {
		t.Fatalf("want %d catalog group(s), have %d", want, have)
	}
	if want, have := "Müller", h.catalogGroups[0].Name; want != have {
		t.Errorf("want decoded group name %q, have %q", want, have)
	}
}

func TestReadGBKChinese(t *testing.T) {
	const utf8doc = `<?xml version="1.0" encoding="GBK"?>` +
		`<BMECAT version="2005"><T_NEW_CATALOG>` +
		`<CATALOG_STRUCTURE type="node"><GROUP_ID>1</GROUP_ID>` +
		`<GROUP_NAME>电子产品</GROUP_NAME></CATALOG_STRUCTURE>` +
		`</T_NEW_CATALOG></BMECAT>`

	// Encode the whole document to GBK; the reader must decode it back.
	gbk, _, err := transform.String(simplifiedchinese.GBK.NewEncoder(), utf8doc)
	if err != nil {
		t.Fatal(err)
	}

	h := &recordingHandler{}
	r := bmecat2005.NewReader(
		bytes.NewReader([]byte(gbk)),
		bmecat2005.WithCharsetReader(internal.AutoCharsetReader),
	)
	if err := r.Do(context.Background(), h); err != nil {
		t.Fatal(err)
	}
	if want, have := 1, len(h.catalogGroups); want != have {
		t.Fatalf("want %d catalog group(s), have %d", want, have)
	}
	if want, have := "电子产品", h.catalogGroups[0].Name; want != have {
		t.Errorf("want decoded group name %q, have %q", want, have)
	}
}

// --- Progress callbacks -----------------------------------------------------

func TestReaderProgress(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "new_catalog.golden.xml"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var maxPass int
	r := bmecat2005.NewReader(f, bmecat2005.WithReaderProgress(func(pass int, _ int64) {
		if pass > maxPass {
			maxPass = pass
		}
	}))
	if err := r.Do(context.Background(), &testHandler{}); err != nil {
		t.Fatal(err)
	}
	if maxPass < 2 {
		t.Errorf("want progress to reach pass 2, reached %d", maxPass)
	}
}

func TestWriterProgress(t *testing.T) {
	products := []*bmecat2005.Product{
		{SupplierPID: "1000", Details: &bmecat2005.ProductDetails{DescriptionShort: "Test"}},
	}
	cw := catalogWriter{tx: bmecat2005.NewCatalog, header: testHeader, products: products}

	var written int
	w := bmecat2005.NewWriter(io.Discard, bmecat2005.WithProgress(func(n int) { written = n }))
	if err := w.Do(context.Background(), cw); err != nil {
		t.Fatal(err)
	}
	if written < 1 {
		t.Errorf("want at least 1 product reported, have %d", written)
	}
}

// --- Writer error path ------------------------------------------------------

// errProductsWriter is a CatalogWriter whose product stream fails.
type errProductsWriter struct {
	catalogWriter
}

func (errProductsWriter) Products(context.Context) (<-chan *bmecat2005.Product, <-chan error) {
	outCh := make(chan *bmecat2005.Product)
	errCh := make(chan error, 1)
	errCh <- errBoom
	return outCh, errCh
}

func TestWriteProductsError(t *testing.T) {
	cw := errProductsWriter{catalogWriter{tx: bmecat2005.NewCatalog, header: testHeader}}

	w := bmecat2005.NewWriter(io.Discard)
	err := w.Do(context.Background(), cw)
	if !errors.Is(err, errBoom) {
		t.Fatalf("expected wrapped errBoom, got %v", err)
	}
}

// --- Round trip -------------------------------------------------------------

func TestRoundTrip(t *testing.T) {
	products := []*bmecat2005.Product{sampleProduct()}
	cw := catalogWriter{
		tx:                   bmecat2005.NewCatalog,
		header:               testHeader,
		classificationSystem: sampleClassificationSystem(),
		products:             products,
	}

	var buf bytes.Buffer
	w := bmecat2005.NewWriter(&buf, bmecat2005.WithIndent("  "))
	if err := w.Do(context.Background(), cw); err != nil {
		t.Fatalf("write: %v", err)
	}

	h := &testHandler{}
	r := bmecat2005.NewReader(bytes.NewReader(buf.Bytes()))
	if err := r.Do(context.Background(), h); err != nil {
		t.Fatalf("read: %v", err)
	}
	if h.header == nil {
		t.Fatal("want a header after round trip, have nil")
	}
	if want, have := testHeader.Catalog.Name, h.header.Catalog.Name; want != have {
		t.Errorf("want catalog name %q, have %q", want, have)
	}
	if want, have := 1, len(h.products); want != have {
		t.Fatalf("want %d product(s), have %d", want, have)
	}
	p := h.products[0]
	if want, have := "1000", p.SupplierPID; want != have {
		t.Errorf("want supplier PID %q, have %q", want, have)
	}
	if p.LogisticDetails == nil || p.LogisticDetails.Dimensions == nil {
		t.Fatalf("want logistic details with dimensions after round trip, have %#v", p.LogisticDetails)
	}
	if want, have := 1.37, p.LogisticDetails.Dimensions.Weight; want != have {
		t.Errorf("want weight %v, have %v", want, have)
	}
}
