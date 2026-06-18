package bmecat12_test

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

	"github.com/olivere/bmecat/bmecat12"
	"github.com/olivere/bmecat/internal"
)

// errBoom is a sentinel error used to assert that the reader and writer wrap
// and propagate handler errors via %w.
var errBoom = errors.New("boom")

// --- Reader error paths -----------------------------------------------------

func TestReadMalformedXML(t *testing.T) {
	// Mismatched end tag: the tokenizer must fail in the first pass.
	const malformed = `<BMECAT><HEADER></BMECAT>`
	r := bmecat12.NewReader(bytes.NewReader([]byte(malformed)))
	if err := r.Do(context.Background(), &testHandler{}); err == nil {
		t.Fatal("expected an error for malformed XML, got nil")
	}
}

type failingArticleHandler struct{}

func (failingArticleHandler) HandleArticle(*bmecat12.Article) error { return errBoom }

func TestReadArticleHandlerError(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "new_catalog.golden.xml"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r := bmecat12.NewReader(f)
	err = r.Do(context.Background(), failingArticleHandler{})
	if !errors.Is(err, errBoom) {
		t.Fatalf("expected wrapped errBoom, got %v", err)
	}
}

type failingHeaderHandler struct{}

func (failingHeaderHandler) HandleHeader(*bmecat12.Header) error { return errBoom }

func TestReadHeaderHandlerError(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "new_catalog.golden.xml"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r := bmecat12.NewReader(f)
	err = r.Do(context.Background(), failingHeaderHandler{})
	if !errors.Is(err, errBoom) {
		t.Fatalf("expected wrapped errBoom, got %v", err)
	}
}

// --- Reader group dispatch and completion -----------------------------------

type recordingHandler struct {
	classGroups   int
	catalogGroups []*bmecat12.CatalogGroup
	completed     int
}

func (h *recordingHandler) HandleClassificationGroup(*bmecat12.ClassificationGroup) error {
	h.classGroups++
	return nil
}

func (h *recordingHandler) HandleCatalogGroup(cg *bmecat12.CatalogGroup) error {
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
	r := bmecat12.NewReader(f)
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
<BMECAT version="1.2">
  <T_NEW_CATALOG>
    <CATALOG_STRUCTURE type="node">
      <GROUP_ID>1</GROUP_ID>
      <GROUP_NAME>Hardware</GROUP_NAME>
    </CATALOG_STRUCTURE>
  </T_NEW_CATALOG>
</BMECAT>`

	h := &recordingHandler{}
	r := bmecat12.NewReader(bytes.NewReader([]byte(xml)))
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

// --- Charset handling -------------------------------------------------------

func TestReadISO8859_1(t *testing.T) {
	// GROUP_NAME contains 0xFC, which is "ü" in ISO-8859-1. The reader must
	// decode it to UTF-8 via the (explicitly supplied) charset reader.
	iso := "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>" +
		"<BMECAT version=\"1.2\"><T_NEW_CATALOG>" +
		"<CATALOG_STRUCTURE type=\"leaf\"><GROUP_ID>1</GROUP_ID>" +
		"<GROUP_NAME>M\xfcller</GROUP_NAME></CATALOG_STRUCTURE>" +
		"</T_NEW_CATALOG></BMECAT>"

	h := &recordingHandler{}
	r := bmecat12.NewReader(
		bytes.NewReader([]byte(iso)),
		bmecat12.WithCharsetReader(internal.AutoCharsetReader),
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

func TestReadUTF8Chinese(t *testing.T) {
	const doc = `<?xml version="1.0" encoding="UTF-8"?>` +
		`<BMECAT version="1.2"><T_NEW_CATALOG>` +
		`<CATALOG_STRUCTURE type="node"><GROUP_ID>1</GROUP_ID>` +
		`<GROUP_NAME>电子产品</GROUP_NAME></CATALOG_STRUCTURE>` +
		`</T_NEW_CATALOG></BMECAT>`

	h := &recordingHandler{}
	r := bmecat12.NewReader(bytes.NewReader([]byte(doc)))
	if err := r.Do(context.Background(), h); err != nil {
		t.Fatal(err)
	}
	if want, have := 1, len(h.catalogGroups); want != have {
		t.Fatalf("want %d catalog group(s), have %d", want, have)
	}
	if want, have := "电子产品", h.catalogGroups[0].Name; want != have {
		t.Errorf("want group name %q, have %q", want, have)
	}
}

func TestReadGBKChinese(t *testing.T) {
	const utf8doc = `<?xml version="1.0" encoding="GBK"?>` +
		`<BMECAT version="1.2"><T_NEW_CATALOG>` +
		`<CATALOG_STRUCTURE type="node"><GROUP_ID>1</GROUP_ID>` +
		`<GROUP_NAME>电子产品</GROUP_NAME></CATALOG_STRUCTURE>` +
		`</T_NEW_CATALOG></BMECAT>`

	// Encode the whole document to GBK; the reader must decode it back.
	gbk, _, err := transform.String(simplifiedchinese.GBK.NewEncoder(), utf8doc)
	if err != nil {
		t.Fatal(err)
	}

	h := &recordingHandler{}
	r := bmecat12.NewReader(
		bytes.NewReader([]byte(gbk)),
		bmecat12.WithCharsetReader(internal.AutoCharsetReader),
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
	r := bmecat12.NewReader(f, bmecat12.WithReaderProgress(func(pass int, _ int64) {
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
	articles := []*bmecat12.Article{
		{SupplierAID: "1000", Details: &bmecat12.ArticleDetails{DescriptionShort: "Test"}},
	}
	cw := catalogWriter{tx: bmecat12.NewCatalog, header: testHeader, articles: articles}

	var written int
	w := bmecat12.NewWriter(io.Discard, bmecat12.WithProgress(func(n int) { written = n }))
	if err := w.Do(context.Background(), cw); err != nil {
		t.Fatal(err)
	}
	if written < 1 {
		t.Errorf("want at least 1 article reported, have %d", written)
	}
}

// --- Writer error path ------------------------------------------------------

// errArticlesWriter is a CatalogWriter whose article stream fails.
type errArticlesWriter struct {
	catalogWriter
}

func (errArticlesWriter) Articles(context.Context) (<-chan *bmecat12.Article, <-chan error) {
	outCh := make(chan *bmecat12.Article)
	errCh := make(chan error, 1)
	errCh <- errBoom
	return outCh, errCh
}

func TestWriteArticlesError(t *testing.T) {
	cw := errArticlesWriter{catalogWriter{tx: bmecat12.NewCatalog, header: testHeader}}

	w := bmecat12.NewWriter(io.Discard)
	err := w.Do(context.Background(), cw)
	if !errors.Is(err, errBoom) {
		t.Fatalf("expected wrapped errBoom, got %v", err)
	}
}

// --- Round trip -------------------------------------------------------------

func TestRoundTrip(t *testing.T) {
	articles := []*bmecat12.Article{
		{SupplierAID: "1000", Details: &bmecat12.ArticleDetails{DescriptionShort: "Test product"}},
	}
	cw := catalogWriter{tx: bmecat12.NewCatalog, header: testHeader, articles: articles}

	var buf bytes.Buffer
	w := bmecat12.NewWriter(&buf, bmecat12.WithIndent("  "))
	if err := w.Do(context.Background(), cw); err != nil {
		t.Fatalf("write: %v", err)
	}

	h := &testHandler{}
	r := bmecat12.NewReader(bytes.NewReader(buf.Bytes()))
	if err := r.Do(context.Background(), h); err != nil {
		t.Fatalf("read: %v", err)
	}
	if h.header == nil {
		t.Fatal("want a header after round trip, have nil")
	}
	if want, have := testHeader.Catalog.Name, h.header.Catalog.Name; want != have {
		t.Errorf("want catalog name %q, have %q", want, have)
	}
	if want, have := 1, len(h.articles); want != have {
		t.Fatalf("want %d article(s), have %d", want, have)
	}
	if want, have := "1000", h.articles[0].SupplierAID; want != have {
		t.Errorf("want supplier AID %q, have %q", want, have)
	}
}
