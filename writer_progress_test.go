package bmecat_test

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/olivere/bmecat"
)

// TestWriteProgress confirms WithProgress reports the running product count
// after each product, for both versions and both entry points.
func TestWriteProgress(t *testing.T) {
	products := make([]*bmecat.Product, 3)
	for i := range products {
		products[i] = &bmecat.Product{ID: fmt.Sprint(i + 1), OrderUnit: "PCE"}
	}
	write := map[string]func(*bmecat.Writer) error{
		"Do": func(w *bmecat.Writer) error {
			return w.Do(context.Background(), &sliceCatalogWriter{header: fullHeader(), products: products})
		},
		"WriteFunc": func(w *bmecat.Writer) error {
			return w.WriteFunc(context.Background(), fullHeader(), func(yield func(*bmecat.Product) error) error {
				for _, p := range products {
					if err := yield(p); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}
	for _, version := range []bmecat.Version{bmecat.Version12, bmecat.Version2005} {
		for name, fn := range write {
			t.Run(version.String()+"/"+name, func(t *testing.T) {
				var reported []int
				var buf bytes.Buffer
				w := bmecat.NewWriter(&buf,
					bmecat.WithVersion(version),
					bmecat.WithProgress(func(written int) { reported = append(reported, written) }),
				)
				if err := fn(w); err != nil {
					t.Fatal(err)
				}
				if want := []int{1, 2, 3}; !slices.Equal(want, reported) {
					t.Errorf("progress reported %v, want %v", reported, want)
				}
			})
		}
	}
}
