package bmecat_test

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/olivere/bmecat"
)

// classificationSystem is a neutral CLASSIFICATION_SYSTEM with system-level
// metadata, level names, and a node→leaf group tree, so an emit + read-back
// exercises the whole conversion.
func classificationSystem() *bmecat.ClassificationSystem {
	return &bmecat.ClassificationSystem{
		Name:        "udf_Supplier-1.0",
		FullName:    bmecat.Localized("Supplier classification 1.0"),
		Version:     "1.0",
		Description: bmecat.Localized("Supplier-defined classification"),
		Levels:      2,
		LevelNames: []*bmecat.ClassificationSystemLevelName{
			{Level: 1, Name: "Main group"},
			{Level: 2, Name: "Sub group"},
		},
		Groups: []*bmecat.ClassificationGroup{
			{Type: "node", ID: "1", Name: bmecat.Localized("Tools"), Description: bmecat.Localized("All tools")},
			{Type: "leaf", ID: "2", Name: bmecat.Localized("Widgets"), Description: bmecat.Localized("Widget tools"), ParentID: "1"},
		},
	}
}

// classifCollector captures the header and every classification group the reader
// surfaces, so a write→read round trip can be asserted.
type classifCollector struct {
	groups []*bmecat.ClassificationGroup
}

func (c *classifCollector) HandleHeader(*bmecat.Header) error { return nil }

func (c *classifCollector) HandleClassificationGroup(g *bmecat.ClassificationGroup) error {
	c.groups = append(c.groups, g)
	return nil
}

// TestWriteClassificationSystemRoundTrip writes a catalog carrying a
// classification system for each version and reads the groups back, asserting
// the group tree survives the round trip for both 1.2 and 2005.
func TestWriteClassificationSystemRoundTrip(t *testing.T) {
	for _, version := range []bmecat.Version{bmecat.Version12, bmecat.Version2005} {
		t.Run(version.String(), func(t *testing.T) {
			cw := &sliceCatalogWriter{
				header:   fullHeader(),
				products: []*bmecat.Product{fullProduct()},
			}
			var buf bytes.Buffer
			err := bmecat.NewWriter(
				&buf,
				bmecat.WithVersion(version),
				bmecat.WithClassificationSystem(classificationSystem()),
			).Do(context.Background(), cw)
			if err != nil {
				t.Fatalf("write: %v", err)
			}

			// System-level metadata is emitted but not surfaced on read, so
			// assert it on the wire.
			out := buf.String()
			for _, want := range []string{
				"<CLASSIFICATION_SYSTEM>",
				"<CLASSIFICATION_SYSTEM_NAME>udf_Supplier-1.0</CLASSIFICATION_SYSTEM_NAME>",
				"<CLASSIFICATION_SYSTEM_FULLNAME>Supplier classification 1.0</CLASSIFICATION_SYSTEM_FULLNAME>",
				"<CLASSIFICATION_SYSTEM_LEVELS>2</CLASSIFICATION_SYSTEM_LEVELS>",
			} {
				if !strings.Contains(out, want) {
					t.Errorf("emitted XML missing %q\n%s", want, out)
				}
			}

			c := &classifCollector{}
			r := bmecat.NewReader(bytes.NewReader(buf.Bytes()))
			if err := r.Do(context.Background(), c); err != nil {
				t.Fatalf("read: %v", err)
			}

			want := classificationSystem().Groups
			if !reflect.DeepEqual(c.groups, want) {
				t.Errorf("groups round trip mismatch\n got: %+v\nwant: %+v", c.groups, want)
			}
		})
	}
}

// TestWriteClassificationSystemOmitted confirms that a nil or blank (no groups)
// classification system, and the default of configuring none at all, emit no
// CLASSIFICATION_SYSTEM element.
func TestWriteClassificationSystemOmitted(t *testing.T) {
	tests := []struct {
		name string
		opts []bmecat.WriterOption
	}{
		{"none configured", nil},
		{"nil system", []bmecat.WriterOption{bmecat.WithClassificationSystem(nil)}},
		{"blank system", []bmecat.WriterOption{bmecat.WithClassificationSystem(&bmecat.ClassificationSystem{Name: "x"})}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cw := &sliceCatalogWriter{header: fullHeader(), products: []*bmecat.Product{fullProduct()}}
			var buf bytes.Buffer
			opts := append([]bmecat.WriterOption{bmecat.WithVersion(bmecat.Version12)}, tt.opts...)
			if err := bmecat.NewWriter(&buf, opts...).Do(context.Background(), cw); err != nil {
				t.Fatalf("write: %v", err)
			}
			if strings.Contains(buf.String(), "<CLASSIFICATION_SYSTEM>") {
				t.Errorf("expected no CLASSIFICATION_SYSTEM, got\n%s", buf.String())
			}
		})
	}
}

// TestWriteClassificationSystemUpdatePrices confirms the classification system
// is emitted only for a NewCatalog transaction, mirroring the version writers,
// which gate it behind T_NEW_CATALOG.
func TestWriteClassificationSystemUpdatePrices(t *testing.T) {
	cw := &sliceCatalogWriter{header: fullHeader(), products: []*bmecat.Product{fullProduct()}}
	var buf bytes.Buffer
	err := bmecat.NewWriter(
		&buf,
		bmecat.WithVersion(bmecat.Version12),
		bmecat.WithTransaction(bmecat.UpdatePrices),
		bmecat.WithClassificationSystem(classificationSystem()),
	).Do(context.Background(), cw)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if strings.Contains(buf.String(), "<CLASSIFICATION_SYSTEM>") {
		t.Errorf("expected no CLASSIFICATION_SYSTEM in T_UPDATE_PRICES, got\n%s", buf.String())
	}
}

// TestWriteFuncClassificationSystem confirms the classification system is
// emitted through the pull-style WriteFunc path too, since it is Writer
// configuration rather than part of the product stream.
func TestWriteFuncClassificationSystem(t *testing.T) {
	var buf bytes.Buffer
	err := bmecat.NewWriter(
		&buf,
		bmecat.WithVersion(bmecat.Version2005),
		bmecat.WithClassificationSystem(classificationSystem()),
	).WriteFunc(context.Background(), fullHeader(), func(yield func(*bmecat.Product) error) error {
		return yield(fullProduct())
	})
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	c := &classifCollector{}
	r := bmecat.NewReader(bytes.NewReader(buf.Bytes()))
	if err := r.Do(context.Background(), c); err != nil {
		t.Fatalf("read: %v", err)
	}
	if !reflect.DeepEqual(c.groups, classificationSystem().Groups) {
		t.Errorf("groups round trip mismatch\n got: %+v\nwant: %+v", c.groups, classificationSystem().Groups)
	}
}
