package bmecat_test

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/olivere/bmecat"
)

func TestLocalizedAccessors(t *testing.T) {
	multi := bmecat.LocalizedStrings{
		{Lang: "deu", Value: "Hallo"},
		{Lang: "eng", Value: "Hi"},
		{Lang: "deu", Value: "Servus"},
	}
	tests := []struct {
		name string
		in   bmecat.LocalizedStrings
		lang string
		want string
	}{
		{"exact match returns first such", multi, "eng", "Hi"},
		{"no match falls back to first", multi, "fra", "Hallo"},
		{"empty slice", nil, "deu", ""},
		{"empty lang matches attr-free entry", bmecat.Localized("Plain"), "", "Plain"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if have := tt.in.Get(tt.lang); have != tt.want {
				t.Errorf("Get(%q) = %q, want %q", tt.lang, have, tt.want)
			}
		})
	}

	if have := multi.Value(); have != "Hallo" {
		t.Errorf("Value() = %q, want %q", have, "Hallo")
	}
	if have := bmecat.LocalizedStrings(nil).Value(); have != "" {
		t.Errorf("Value() on empty = %q, want empty", have)
	}

	// All returns every value for a language, falling back to all values when
	// none match (so language-less data is still returned).
	if have, want := multi.All("deu"), []string{"Hallo", "Servus"}; !reflect.DeepEqual(have, want) {
		t.Errorf("All(deu) = %v, want %v", have, want)
	}
	if have, want := multi.All("fra"), []string{"Hallo", "Hi", "Servus"}; !reflect.DeepEqual(have, want) {
		t.Errorf("All(fra) fallback = %v, want %v", have, want)
	}
	if have := bmecat.LocalizedStrings(nil).All("deu"); have != nil {
		t.Errorf("All on empty = %v, want nil", have)
	}

	if have := bmecat.Localized("a", "b"); !reflect.DeepEqual(have, bmecat.LocalizedStrings{{Value: "a"}, {Value: "b"}}) {
		t.Errorf("Localized(a, b) = %+v, want two attr-free entries", have)
	}
}

func TestLocalizedSet(t *testing.T) {
	var s bmecat.LocalizedStrings
	s.Set("deu", "Hallo")
	s.Set("eng", "Hi")
	if want := (bmecat.LocalizedStrings{{Lang: "deu", Value: "Hallo"}, {Lang: "eng", Value: "Hi"}}); !reflect.DeepEqual(s, want) {
		t.Fatalf("after adds = %+v, want %+v", s, want)
	}
	s.Set("deu", "Servus") // replaces in place
	if want := (bmecat.LocalizedStrings{{Lang: "deu", Value: "Servus"}, {Lang: "eng", Value: "Hi"}}); !reflect.DeepEqual(s, want) {
		t.Fatalf("after replace = %+v, want %+v", s, want)
	}
}

// multiLang2005 is a 2005 catalog whose localized elements carry two languages
// each, across the spread of fields the neutral model exposes.
const multiLang2005 = `<?xml version="1.0"?>
<BMECAT version="2005" xmlns="http://www.bmecat.org/bmecat/2005">
  <HEADER>
    <CATALOG>
      <LANGUAGE>deu</LANGUAGE>
      <CATALOG_ID>C</CATALOG_ID>
      <CATALOG_VERSION>1</CATALOG_VERSION>
      <CATALOG_NAME lang="deu">Katalog</CATALOG_NAME>
      <CATALOG_NAME lang="eng">Catalog</CATALOG_NAME>
    </CATALOG>
  </HEADER>
  <T_NEW_CATALOG>
    <PRODUCT>
      <SUPPLIER_PID>1</SUPPLIER_PID>
      <PRODUCT_DETAILS>
        <DESCRIPTION_SHORT lang="deu">Hallo</DESCRIPTION_SHORT>
        <DESCRIPTION_SHORT lang="eng">Hi</DESCRIPTION_SHORT>
        <DESCRIPTION_LONG lang="deu">Langer Text</DESCRIPTION_LONG>
        <MANUFACTURER_TYPE_DESCR lang="deu">Typ</MANUFACTURER_TYPE_DESCR>
        <MANUFACTURER_TYPE_DESCR lang="eng">Type</MANUFACTURER_TYPE_DESCR>
        <KEYWORD lang="deu">Schraube</KEYWORD>
        <KEYWORD lang="eng">Screw</KEYWORD>
        <REMARKS lang="deu">Hinweis</REMARKS>
      </PRODUCT_DETAILS>
      <PRODUCT_FEATURES>
        <REFERENCE_FEATURE_SYSTEM_NAME>ECLASS-5.1</REFERENCE_FEATURE_SYSTEM_NAME>
        <REFERENCE_FEATURE_GROUP_NAME lang="deu">Gruppe</REFERENCE_FEATURE_GROUP_NAME>
        <REFERENCE_FEATURE_GROUP_NAME lang="eng">Group</REFERENCE_FEATURE_GROUP_NAME>
        <FEATURE>
          <FNAME lang="deu">Farbe</FNAME>
          <FNAME lang="eng">Color</FNAME>
          <FVALUE lang="deu">rot</FVALUE>
          <FVALUE lang="eng">red</FVALUE>
        </FEATURE>
      </PRODUCT_FEATURES>
    </PRODUCT>
  </T_NEW_CATALOG>
</BMECAT>`

// TestRead2005LocalizedFields verifies that 2005 preserves every language
// variant across the localized fields, in document order.
func TestRead2005LocalizedFields(t *testing.T) {
	c := read(t, multiLang2005)
	if c.header == nil || c.header.Catalog == nil {
		t.Fatal("want catalog header")
	}
	if want, have := "Catalog", c.header.Catalog.Name.Get("eng"); want != have {
		t.Errorf("Catalog.Name Get(eng) = %q, want %q", have, want)
	}
	if len(c.products) != 1 {
		t.Fatalf("want one product, have %d", len(c.products))
	}
	p := c.products[0]

	checks := []struct {
		name string
		have bmecat.LocalizedStrings
		want bmecat.LocalizedStrings
	}{
		{"DescriptionShort", p.DescriptionShort, bmecat.LocalizedStrings{{Lang: "deu", Value: "Hallo"}, {Lang: "eng", Value: "Hi"}}},
		{"DescriptionLong", p.DescriptionLong, bmecat.LocalizedStrings{{Lang: "deu", Value: "Langer Text"}}},
		{"ManufacturerTypeDescr", p.ManufacturerTypeDescr, bmecat.LocalizedStrings{{Lang: "deu", Value: "Typ"}, {Lang: "eng", Value: "Type"}}},
		{"Keywords", p.Keywords, bmecat.LocalizedStrings{{Lang: "deu", Value: "Schraube"}, {Lang: "eng", Value: "Screw"}}},
		{"Remarks", p.Remarks, bmecat.LocalizedStrings{{Lang: "deu", Value: "Hinweis"}}},
	}
	for _, ck := range checks {
		if !reflect.DeepEqual(ck.have, ck.want) {
			t.Errorf("%s = %+v, want %+v", ck.name, ck.have, ck.want)
		}
	}

	if len(p.Features) != 1 || len(p.Features[0].Features) != 1 {
		t.Fatalf("want one feature, have %+v", p.Features)
	}
	if want, have := "Group", p.Features[0].GroupName.Get("eng"); want != have {
		t.Errorf("GroupName Get(eng) = %q, want %q", have, want)
	}
	f := p.Features[0].Features[0]
	if want, have := "Color", f.Name.Get("eng"); want != have {
		t.Errorf("Feature.Name Get(eng) = %q, want %q", have, want)
	}
	if want, have := []string{"red"}, f.Values.All("eng"); !reflect.DeepEqual(want, have) {
		t.Errorf("Feature.Values All(eng) = %v, want %v", have, want)
	}
}

// TestRead2005RoundTrip writes a neutral catalog carrying multiple language
// variants to 2005 and reads it back, asserting the variants survive unchanged.
func TestWriteRead2005LocalizedRoundTrip(t *testing.T) {
	header := fullHeader()
	header.Catalog.Name = bmecat.LocalizedStrings{{Lang: "deu", Value: "Katalog"}, {Lang: "eng", Value: "Catalog"}}
	product := &bmecat.Product{
		ID:               "1",
		DescriptionShort: bmecat.LocalizedStrings{{Lang: "deu", Value: "Hallo"}, {Lang: "eng", Value: "Hi"}},
		Keywords:         bmecat.LocalizedStrings{{Lang: "deu", Value: "Schraube"}, {Lang: "eng", Value: "Screw"}},
		Remarks:          bmecat.LocalizedStrings{{Lang: "deu", Value: "Hinweis"}},
		Features: []*bmecat.Features{{
			SystemName: "ECLASS-5.1",
			GroupName:  bmecat.LocalizedStrings{{Lang: "deu", Value: "Gruppe"}, {Lang: "eng", Value: "Group"}},
			Features: []*bmecat.Feature{{
				Name:   bmecat.LocalizedStrings{{Lang: "deu", Value: "Farbe"}, {Lang: "eng", Value: "Color"}},
				Values: bmecat.LocalizedStrings{{Lang: "deu", Value: "rot"}, {Lang: "eng", Value: "red"}},
			}},
		}},
		Mimes: []*bmecat.Mime{{
			Type:   "application/pdf",
			Source: bmecat.LocalizedStrings{{Lang: "deu", Value: "blatt-de.pdf"}, {Lang: "eng", Value: "sheet-en.pdf"}},
			Descr:  bmecat.LocalizedStrings{{Lang: "deu", Value: "Datenblatt"}, {Lang: "eng", Value: "Datasheet"}},
		}},
	}

	var buf bytes.Buffer
	cw := &sliceCatalogWriter{header: header, products: []*bmecat.Product{product}}
	if err := bmecat.NewWriter(&buf, bmecat.WithVersion(bmecat.Version2005)).Do(context.Background(), cw); err != nil {
		t.Fatal(err)
	}
	c := read(t, buf.String())

	if !reflect.DeepEqual(header.Catalog.Name, c.header.Catalog.Name) {
		t.Errorf("Catalog.Name round trip: want %+v, have %+v", header.Catalog.Name, c.header.Catalog.Name)
	}
	if len(c.products) != 1 {
		t.Fatalf("want one product, have %d", len(c.products))
	}
	got := c.products[0]
	for _, ck := range []struct {
		name       string
		want, have bmecat.LocalizedStrings
	}{
		{"DescriptionShort", product.DescriptionShort, got.DescriptionShort},
		{"Keywords", product.Keywords, got.Keywords},
		{"Remarks", product.Remarks, got.Remarks},
		{"GroupName", product.Features[0].GroupName, got.Features[0].GroupName},
		{"Feature.Name", product.Features[0].Features[0].Name, got.Features[0].Features[0].Name},
		{"Feature.Values", product.Features[0].Features[0].Values, got.Features[0].Features[0].Values},
		{"Mime.Source", product.Mimes[0].Source, got.Mimes[0].Source},
		{"Mime.Descr", product.Mimes[0].Descr, got.Mimes[0].Descr},
	} {
		if !reflect.DeepEqual(ck.want, ck.have) {
			t.Errorf("%s round trip: want %+v, have %+v", ck.name, ck.want, ck.have)
		}
	}
}

// TestRead12Collapses confirms BMEcat 1.2, which has no per-element lang,
// collapses repeated localized elements onto a single language-less value
// (encoding/xml keeps the last) rather than inventing variants.
func TestRead12Collapses(t *testing.T) {
	const doc = `<?xml version="1.0"?>
<BMECAT version="1.2">
  <HEADER><CATALOG><LANGUAGE>deu</LANGUAGE><CATALOG_ID>C</CATALOG_ID><CATALOG_VERSION>1</CATALOG_VERSION></CATALOG></HEADER>
  <T_NEW_CATALOG>
    <ARTICLE>
      <SUPPLIER_AID>1</SUPPLIER_AID>
      <ARTICLE_DETAILS>
        <DESCRIPTION_SHORT>Widget</DESCRIPTION_SHORT>
        <KEYWORD>tool</KEYWORD>
        <KEYWORD>widget</KEYWORD>
      </ARTICLE_DETAILS>
    </ARTICLE>
  </T_NEW_CATALOG>
</BMECAT>`
	c := read(t, doc)
	p := c.products[0]
	if want, have := (bmecat.LocalizedStrings{{Value: "Widget"}}), p.DescriptionShort; !reflect.DeepEqual(want, have) {
		t.Errorf("DescriptionShort = %+v, want %+v (language-less)", have, want)
	}
	// Both keywords survive (KEYWORD repeats in 1.2 too), each language-less.
	if want, have := (bmecat.LocalizedStrings{{Value: "tool"}, {Value: "widget"}}), p.Keywords; !reflect.DeepEqual(want, have) {
		t.Errorf("Keywords = %+v, want %+v", have, want)
	}
}

// TestWrite12NoCatalogLanguage confirms that when no catalog language is
// configured, writing multi-language neutral data to 1.2 collapses to the first
// variant's language for both single and repeating elements, rather than leaking
// every language's values into the single-language 1.2 document.
func TestWrite12NoCatalogLanguage(t *testing.T) {
	header := fullHeader()
	header.Catalog.Language = "" // no declared language
	product := &bmecat.Product{
		ID:               "1",
		DescriptionShort: bmecat.LocalizedStrings{{Lang: "deu", Value: "Hallo"}, {Lang: "eng", Value: "Hi"}},
		Keywords:         bmecat.LocalizedStrings{{Lang: "deu", Value: "Schraube"}, {Lang: "eng", Value: "Screw"}},
	}
	var buf bytes.Buffer
	cw := &sliceCatalogWriter{header: header, products: []*bmecat.Product{product}}
	if err := bmecat.NewWriter(&buf, bmecat.WithVersion(bmecat.Version12)).Do(context.Background(), cw); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "<DESCRIPTION_SHORT>Hallo</DESCRIPTION_SHORT>") {
		t.Errorf("want first-language description, got:\n%s", out)
	}
	// Only the first language's keyword, not both — no cross-language leak.
	if !strings.Contains(out, "<KEYWORD>Schraube</KEYWORD>") || strings.Contains(out, "Screw") {
		t.Errorf("want only the first-language keyword, got:\n%s", out)
	}
}

// TestWriteAttrFreeDescription confirms a single-language description is written
// without a spurious lang attribute, for both versions.
func TestWriteAttrFreeDescription(t *testing.T) {
	for _, version := range []bmecat.Version{bmecat.Version12, bmecat.Version2005} {
		t.Run(version.String(), func(t *testing.T) {
			product := &bmecat.Product{ID: "1", DescriptionShort: bmecat.Localized("Widget")}
			var buf bytes.Buffer
			cw := &sliceCatalogWriter{header: fullHeader(), products: []*bmecat.Product{product}}
			if err := bmecat.NewWriter(&buf, bmecat.WithVersion(version)).Do(context.Background(), cw); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(buf.String(), "<DESCRIPTION_SHORT>Widget</DESCRIPTION_SHORT>") {
				t.Errorf("want attr-free <DESCRIPTION_SHORT>Widget</DESCRIPTION_SHORT>, got:\n%s", buf.String())
			}
			if strings.Contains(buf.String(), "DESCRIPTION_SHORT lang=") {
				t.Errorf("unexpected lang attribute on single-language description:\n%s", buf.String())
			}
		})
	}
}

// TestWrite12PicksCatalogLanguage confirms that writing a multi-language neutral
// catalog to 1.2 (which is single-language) emits the variant matching the
// catalog's declared language, falling back to the first otherwise.
func TestWrite12PicksCatalogLanguage(t *testing.T) {
	header := fullHeader()
	header.Catalog.Language = "eng"
	product := &bmecat.Product{
		ID:               "1",
		DescriptionShort: bmecat.LocalizedStrings{{Lang: "deu", Value: "Hallo"}, {Lang: "eng", Value: "Hi"}},
		Keywords:         bmecat.LocalizedStrings{{Lang: "deu", Value: "Schraube"}, {Lang: "eng", Value: "Screw"}},
	}
	var buf bytes.Buffer
	cw := &sliceCatalogWriter{header: header, products: []*bmecat.Product{product}}
	if err := bmecat.NewWriter(&buf, bmecat.WithVersion(bmecat.Version12)).Do(context.Background(), cw); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "<DESCRIPTION_SHORT>Hi</DESCRIPTION_SHORT>") {
		t.Errorf("want English description, got:\n%s", out)
	}
	if strings.Contains(out, "Hallo") {
		t.Errorf("did not want the German variant in 1.2 output:\n%s", out)
	}
	if !strings.Contains(out, "<KEYWORD>Screw</KEYWORD>") || strings.Contains(out, "Schraube") {
		t.Errorf("want only the English keyword, got:\n%s", out)
	}
	if strings.Contains(out, "lang=") {
		t.Errorf("1.2 output must not carry lang attributes:\n%s", out)
	}
}
