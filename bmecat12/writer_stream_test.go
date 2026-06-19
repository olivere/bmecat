package bmecat12_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/olivere/bmecat/bmecat12"
)

// funcCatalog is a CatalogWriter whose Articles method is backed by the
// pull-style StreamArticles helper, exercising the helper end to end through
// Writer.Do.
type funcCatalog struct {
	header  *bmecat12.Header
	produce func(yield func(*bmecat12.Article) error) error
}

func (funcCatalog) Transaction() bmecat12.Transaction                    { return bmecat12.NewCatalog }
func (funcCatalog) Language() string                                     { return "deu" }
func (funcCatalog) PreviousVersion() int                                 { return 0 }
func (c funcCatalog) Header() *bmecat12.Header                           { return c.header }
func (funcCatalog) ClassificationSystem() *bmecat12.ClassificationSystem { return nil }

func (c funcCatalog) Articles(ctx context.Context) (<-chan *bmecat12.Article, <-chan error) {
	return bmecat12.StreamArticles(ctx, c.produce)
}

// TestStreamArticles writes several articles through StreamArticles and confirms
// they are all emitted in order.
func TestStreamArticles(t *testing.T) {
	const n = 100
	cw := funcCatalog{
		header: testHeader,
		produce: func(yield func(*bmecat12.Article) error) error {
			for i := range n {
				a := &bmecat12.Article{
					SupplierAID: fmt.Sprintf("A%04d", i),
					Details:     &bmecat12.ArticleDetails{DescriptionShort: "x"},
				}
				if err := yield(a); err != nil {
					return err
				}
			}
			return nil
		},
	}

	var buf bytes.Buffer
	if err := bmecat12.NewWriter(&buf).Do(context.Background(), cw); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if got := strings.Count(out, "<SUPPLIER_AID>"); got != n {
		t.Fatalf("want %d articles, have %d", n, got)
	}
	if first, last := strings.Index(out, "A0000"), strings.Index(out, fmt.Sprintf("A%04d", n-1)); first < 0 || last < 0 || first > last {
		t.Errorf("article order not preserved (first idx %d, last idx %d)", first, last)
	}
}

// TestStreamArticlesSkipsNil confirms a nil article yielded by the producer is
// skipped rather than written.
func TestStreamArticlesSkipsNil(t *testing.T) {
	cw := funcCatalog{
		header: testHeader,
		produce: func(yield func(*bmecat12.Article) error) error {
			if err := yield(nil); err != nil {
				return err
			}
			return yield(&bmecat12.Article{SupplierAID: "1", Details: &bmecat12.ArticleDetails{DescriptionShort: "x"}})
		},
	}

	var buf bytes.Buffer
	if err := bmecat12.NewWriter(&buf).Do(context.Background(), cw); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(buf.String(), "<SUPPLIER_AID>"); got != 1 {
		t.Fatalf("want one article (nil skipped), have %d", got)
	}
}

// TestStreamArticlesProducerError confirms an error returned by the producer is
// reported by Writer.Do.
func TestStreamArticlesProducerError(t *testing.T) {
	boom := errors.New("producer boom")
	cw := funcCatalog{
		header: testHeader,
		produce: func(yield func(*bmecat12.Article) error) error {
			if err := yield(&bmecat12.Article{SupplierAID: "1", Details: &bmecat12.ArticleDetails{DescriptionShort: "x"}}); err != nil {
				return err
			}
			return boom
		},
	}

	err := bmecat12.NewWriter(&bytes.Buffer{}).Do(context.Background(), cw)
	if !errors.Is(err, boom) {
		t.Fatalf("want producer error, have %v", err)
	}
}

// TestStreamArticlesContextCancel confirms a canceled context aborts the write
// and that yield observes the cancellation.
func TestStreamArticlesContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cw := funcCatalog{
		header: testHeader,
		produce: func(yield func(*bmecat12.Article) error) error {
			return yield(&bmecat12.Article{SupplierAID: "1", Details: &bmecat12.ArticleDetails{DescriptionShort: "x"}})
		},
	}

	if err := bmecat12.NewWriter(&bytes.Buffer{}).Do(ctx, cw); err == nil {
		t.Fatal("want error from canceled context, got nil")
	}
}
