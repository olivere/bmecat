package bmecat12_test

import (
	"context"
	"io"
	"os"
	"path"
	"testing"

	"github.com/olivere/bmecat/bmecat12"
)

type testHandler struct {
	firstPassOnly bool
	header        *bmecat12.Header
	articles      []*bmecat12.Article
}

func (h *testHandler) HandleHeader(header *bmecat12.Header) error {
	h.header = header
	if h.firstPassOnly {
		return io.EOF
	}
	return nil
}

func (h *testHandler) HandleArticle(article *bmecat12.Article) error {
	h.articles = append(h.articles, article)
	return nil
}

func TestReadCatalog(t *testing.T) {
	f, err := os.Open(path.Join("testdata", "new_catalog.golden.xml"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	h := &testHandler{}
	r := bmecat12.NewReader(f)
	err = r.Do(context.Background(), h)
	if err != nil {
		t.Fatal(err)
	}
	if h.header == nil {
		t.Fatal("want Header, have nil")
	}
	if want, have := 1, len(h.articles); want != have {
		t.Fatalf("want len(articles) = %d, have %d", want, have)
	}
}

func TestReadUpdateProducts(t *testing.T) {
	f, err := os.Open(path.Join("testdata", "update_products.golden.xml"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	h := &testHandler{}
	r := bmecat12.NewReader(f)
	err = r.Do(context.Background(), h)
	if err != nil {
		t.Fatal(err)
	}
	if h.header == nil {
		t.Fatal("want Header, have nil")
	}
	if want, have := 2, len(h.articles); want != have {
		t.Fatalf("want len(articles) = %d, have %d", want, have)
	}
}

func TestReadUpdatePrices(t *testing.T) {
	f, err := os.Open(path.Join("testdata", "update_prices.golden.xml"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	h := &testHandler{}
	r := bmecat12.NewReader(f)
	err = r.Do(context.Background(), h)
	if err != nil {
		t.Fatal(err)
	}
	if h.header == nil {
		t.Fatal("want Header, have nil")
	}
	if want, have := 1, len(h.articles); want != have {
		t.Fatalf("want len(articles) = %d, have %d", want, have)
	}
}
