package bmecat2005_test

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/olivere/bmecat/bmecat2005"
)

func TestLocalizedStringsMarshal(t *testing.T) {
	tests := []struct {
		name    string
		details bmecat2005.ProductDetails
		want    []string // substrings that must appear
		absent  []string // substrings that must not appear
	}{
		{
			name:    "single variant has no lang attribute",
			details: bmecat2005.ProductDetails{DescriptionShort: bmecat2005.Localized("Widget")},
			want:    []string{"<DESCRIPTION_SHORT>Widget</DESCRIPTION_SHORT>"},
			absent:  []string{"DESCRIPTION_SHORT lang="},
		},
		{
			name: "multiple variants emit one element each with lang",
			details: bmecat2005.ProductDetails{DescriptionShort: bmecat2005.LocalizedStrings{
				{Lang: "deu", Value: "Hallo"},
				{Lang: "eng", Value: "Hi"},
			}},
			want: []string{
				`<DESCRIPTION_SHORT lang="deu">Hallo</DESCRIPTION_SHORT>`,
				`<DESCRIPTION_SHORT lang="eng">Hi</DESCRIPTION_SHORT>`,
			},
		},
		{
			name:    "empty required short still emits an element",
			details: bmecat2005.ProductDetails{},
			want:    []string{"<DESCRIPTION_SHORT></DESCRIPTION_SHORT>"},
		},
		{
			name:    "empty optional long is omitted",
			details: bmecat2005.ProductDetails{DescriptionShort: bmecat2005.Localized("Widget")},
			absent:  []string{"DESCRIPTION_LONG"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := xml.Marshal(&tt.details)
			if err != nil {
				t.Fatal(err)
			}
			got := string(b)
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("want %q in:\n%s", w, got)
				}
			}
			for _, a := range tt.absent {
				if strings.Contains(got, a) {
					t.Errorf("did not want %q in:\n%s", a, got)
				}
			}
		})
	}
}

func TestLocalizedStringsUnmarshal(t *testing.T) {
	const in = `<PRODUCT_DETAILS>
  <DESCRIPTION_SHORT lang="deu">Hallo</DESCRIPTION_SHORT>
  <DESCRIPTION_SHORT lang="eng">Hi</DESCRIPTION_SHORT>
  <DESCRIPTION_SHORT>Plain</DESCRIPTION_SHORT>
</PRODUCT_DETAILS>`
	var d bmecat2005.ProductDetails
	if err := xml.Unmarshal([]byte(in), &d); err != nil {
		t.Fatal(err)
	}
	want := bmecat2005.LocalizedStrings{
		{Lang: "deu", Value: "Hallo"},
		{Lang: "eng", Value: "Hi"},
		{Lang: "", Value: "Plain"},
	}
	if len(d.DescriptionShort) != len(want) {
		t.Fatalf("got %d variants, want %d: %+v", len(d.DescriptionShort), len(want), d.DescriptionShort)
	}
	for i := range want {
		if d.DescriptionShort[i] != want[i] {
			t.Errorf("variant %d = %+v, want %+v", i, d.DescriptionShort[i], want[i])
		}
	}
	if have := d.DescriptionShort.Get("eng"); have != "Hi" {
		t.Errorf("Get(eng) = %q, want %q", have, "Hi")
	}
	if have := d.DescriptionShort.Value(); have != "Hallo" {
		t.Errorf("Value() = %q, want %q", have, "Hallo")
	}
}

// TestNewLocalizedFieldsRoundTrip exercises the broader set of dtMLSTRING fields
// (beyond the descriptions) added in this change: address parts, MIME source/alt,
// feature descr/value-details/variant, catalog-group keywords, classification
// group synonyms and classification level names. Each carries two languages and
// must survive a marshal/unmarshal round trip with both variants intact.
func TestNewLocalizedFieldsRoundTrip(t *testing.T) {
	ml := func(de, en string) bmecat2005.LocalizedStrings {
		return bmecat2005.LocalizedStrings{{Lang: "deu", Value: de}, {Lang: "eng", Value: en}}
	}

	t.Run("address and mime", func(t *testing.T) {
		in := &bmecat2005.Supplier{
			Name:    "SupplyCo",
			Address: &bmecat2005.Address{Type: "supplier", City: ml("Köln", "Cologne"), Street: ml("Domstraße", "Cathedral St")},
			MimeInfo: &bmecat2005.MimeInfo{Mimes: []*bmecat2005.Mime{
				{Type: "application/pdf", Source: ml("blatt-de.pdf", "sheet-en.pdf"), Alt: ml("Datenblatt", "Datasheet"), Purpose: "data_sheet"},
			}},
		}
		var out bmecat2005.Supplier
		roundTrip(t, in, &out)
		if got := out.Address.City.Get("eng"); got != "Cologne" {
			t.Errorf("City Get(eng) = %q, want Cologne", got)
		}
		if got := out.Address.Street.Get("deu"); got != "Domstraße" {
			t.Errorf("Street Get(deu) = %q, want Domstraße", got)
		}
		m := out.MimeInfo.Mimes[0]
		if got := m.Source.Get("eng"); got != "sheet-en.pdf" {
			t.Errorf("MIME source Get(eng) = %q, want sheet-en.pdf", got)
		}
		if got := m.Alt.Get("deu"); got != "Datenblatt" {
			t.Errorf("MIME alt Get(deu) = %q, want Datenblatt", got)
		}
	})

	t.Run("feature descr, value details and variants", func(t *testing.T) {
		in := &bmecat2005.Feature{
			Name:         ml("Farbe", "Color"),
			Descr:        ml("Die Farbe", "The color"),
			ValueDetails: ml("rot-ish", "red-ish"),
			Variants: []*bmecat2005.FeatureVariants{{Variants: []*bmecat2005.FeatureVariant{
				{Value: ml("rot", "red"), SupplierAIDSupplement: "-R"},
			}}},
		}
		var out bmecat2005.Feature
		roundTrip(t, in, &out)
		if got := out.Descr.Get("eng"); got != "The color" {
			t.Errorf("FDESCR Get(eng) = %q, want The color", got)
		}
		if got := out.ValueDetails.Get("deu"); got != "rot-ish" {
			t.Errorf("FVALUE_DETAILS Get(deu) = %q, want rot-ish", got)
		}
		if got := out.Variants[0].Variants[0].Value.Get("eng"); got != "red" {
			t.Errorf("variant FVALUE Get(eng) = %q, want red", got)
		}
	})

	t.Run("catalog group keywords", func(t *testing.T) {
		in := &bmecat2005.CatalogGroup{ID: "1", Name: ml("Werkzeug", "Tools"), Keywords: ml("Schraube", "Screw")}
		var out bmecat2005.CatalogGroup
		roundTrip(t, in, &out)
		if got := out.Keywords.All("eng"); len(got) != 1 || got[0] != "Screw" {
			t.Errorf("catalog group keywords All(eng) = %v, want [Screw]", got)
		}
	})

	t.Run("classification synonyms round trip", func(t *testing.T) {
		in := &bmecat2005.ClassificationGroup{
			ID:       "1",
			Name:     ml("Werkzeug", "Tools"),
			Synonyms: []bmecat2005.ClassificationGroupSynonym{{Value: ml("Gerät", "Device")}},
		}
		var out bmecat2005.ClassificationGroup
		roundTrip(t, in, &out)
		if got := out.Synonyms[0].Value.Get("eng"); got != "Device" {
			t.Errorf("synonym Get(eng) = %q, want Device", got)
		}
	})

	t.Run("level names round trip with lang", func(t *testing.T) {
		in := &bmecat2005.ClassificationSystem{
			Name: "udf-1.0",
			LevelNames: []*bmecat2005.ClassificationSystemLevelName{
				{Level: 1, Lang: "deu", Value: "Hauptgruppe"},
				{Level: 1, Lang: "eng", Value: "Main group"},
			},
		}
		b, err := xml.Marshal(in)
		if err != nil {
			t.Fatal(err)
		}
		// The level names must be wrapped in CLASSIFICATION_SYSTEM_LEVEL_NAMES.
		if !strings.Contains(string(b), "<CLASSIFICATION_SYSTEM_LEVEL_NAMES>") {
			t.Errorf("missing CLASSIFICATION_SYSTEM_LEVEL_NAMES wrapper:\n%s", b)
		}
		var out bmecat2005.ClassificationSystem
		if err := xml.Unmarshal(b, &out); err != nil {
			t.Fatal(err)
		}
		if len(out.LevelNames) != 2 || out.LevelNames[1].Lang != "eng" || out.LevelNames[1].Value != "Main group" {
			t.Errorf("level names round trip = %+v", out.LevelNames)
		}
	})
}

func roundTrip(t *testing.T, in, out any) {
	t.Helper()
	b, err := xml.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := xml.Unmarshal(b, out); err != nil {
		t.Fatalf("unmarshal: %v\nxml: %s", err, b)
	}
}
