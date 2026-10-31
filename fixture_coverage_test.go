package bmecat_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/olivere/bmecat"
)

// roundTripLosses lists, per version, the Product fields a neutral
// write→read round trip does not preserve, with the reason. The round-trip
// tests clear these fields on the expected product before comparing, so every
// other field must survive unchanged. Remove an entry when the writer learns to
// carry the field.
var roundTripLosses = map[bmecat.Version]map[string]string{
	bmecat.Version12: {
		"QuantityMax":     "BMEcat 1.2 has no QUANTITY_MAX",
		"PIDs":            "BMEcat 1.2 has no typed INTERNATIONAL_PID; GTIN is written as EAN",
		"CatalogGroupIDs": "the writer emits no ARTICLE_TO_CATALOGGROUP_MAP yet (#42)",
	},
	bmecat.Version2005: {
		"CatalogGroupIDs": "the writer emits no PRODUCT_TO_CATALOGGROUP_MAP yet (#42)",
	},
}

// withoutLosses returns a copy of p with the fields listed in
// roundTripLosses for version set to their zero value.
func withoutLosses(t *testing.T, version bmecat.Version, p *bmecat.Product) *bmecat.Product {
	t.Helper()
	out := *p
	v := reflect.ValueOf(&out).Elem()
	for name := range roundTripLosses[version] {
		f := v.FieldByName(name)
		if !f.IsValid() {
			t.Fatalf("roundTripLosses names unknown Product field %q", name)
		}
		f.SetZero()
	}
	return &out
}

// readerOnlyHeaderFields are Header fields the reader fills from the document
// and the writer derives on its own, so a fixture has no reason to set them.
var readerOnlyHeaderFields = []string{
	"Header.Version",
	"Header.Transaction",
	"Header.NumberOfProducts",
	"Header.NumberOfCatalogGroups",
	"Header.NumberOfClassificationGroups",
}

// TestFixturesSetEveryField fails when fullProduct or fullHeader leaves an
// exported field unset. The round-trip tests compare these fixtures after a
// write and read, so a field added to the neutral model must be added to the
// fixture, and then fails the round trip until every converter carries it.
func TestFixturesSetEveryField(t *testing.T) {
	var unset []string
	collectUnset(reflect.ValueOf(fullProduct()).Elem(), "Product", &unset)
	collectUnset(reflect.ValueOf(fullHeader()).Elem(), "Header", &unset)
	unset = slices.DeleteFunc(unset, func(path string) bool {
		return slices.Contains(readerOnlyHeaderFields, path)
	})
	for _, path := range unset {
		t.Errorf("%s is unset in the round-trip fixture", path)
	}
}

var localizedStringsType = reflect.TypeFor[bmecat.LocalizedStrings]()

// collectUnset appends the path of every zero-valued exported field below v.
// It descends into structs, through non-nil pointers and into every element of
// non-empty slices. A non-nil LocalizedStrings counts as set without checking
// its variants; per-language handling has its own round-trip tests in
// localized_test.go.
// Structs without exported fields, such as time.Time, are checked as a whole.
func collectUnset(v reflect.Value, path string, unset *[]string) {
	if v.IsZero() {
		*unset = append(*unset, path)
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		collectUnset(v.Elem(), path, unset)
	case reflect.Slice:
		if v.Type() == localizedStringsType {
			return
		}
		for i := range v.Len() {
			collectUnset(v.Index(i), path+"[]", unset)
		}
	case reflect.Struct:
		for i := range v.NumField() {
			if f := v.Type().Field(i); f.IsExported() {
				collectUnset(v.Field(i), path+"."+f.Name, unset)
			}
		}
	}
}
