package bmecat_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/olivere/bmecat"
)

// catalogWithGroups is the same logical document in both versions, each
// carrying one CATALOG_STRUCTURE and a CLASSIFICATION_SYSTEM with two groups.
const catalog12Groups = `<?xml version="1.0" encoding="UTF-8"?>
<BMECAT version="1.2">
  <HEADER><CATALOG><LANGUAGE>deu</LANGUAGE><CATALOG_ID>C</CATALOG_ID><CATALOG_VERSION>1</CATALOG_VERSION></CATALOG></HEADER>
  <T_NEW_CATALOG>
    <CATALOG_STRUCTURE type="node">
      <GROUP_ID>10</GROUP_ID>
      <GROUP_NAME>Hardware</GROUP_NAME>
    </CATALOG_STRUCTURE>
    <CLASSIFICATION_SYSTEM>
      <CLASSIFICATION_SYSTEM_NAME>udf_Supplier-1.0</CLASSIFICATION_SYSTEM_NAME>
      <CLASSIFICATION_GROUPS>
        <CLASSIFICATION_GROUP type="node">
          <CLASSIFICATION_GROUP_ID>1</CLASSIFICATION_GROUP_ID>
          <CLASSIFICATION_GROUP_NAME>Root</CLASSIFICATION_GROUP_NAME>
        </CLASSIFICATION_GROUP>
        <CLASSIFICATION_GROUP type="leaf">
          <CLASSIFICATION_GROUP_ID>2</CLASSIFICATION_GROUP_ID>
          <CLASSIFICATION_GROUP_NAME>Leaf</CLASSIFICATION_GROUP_NAME>
          <CLASSIFICATION_GROUP_PARENT_ID>1</CLASSIFICATION_GROUP_PARENT_ID>
        </CLASSIFICATION_GROUP>
      </CLASSIFICATION_GROUPS>
    </CLASSIFICATION_SYSTEM>
  </T_NEW_CATALOG>
</BMECAT>`

const catalog2005Groups = `<?xml version="1.0" encoding="UTF-8"?>
<BMECAT version="2005" xmlns="http://www.bmecat.org/bmecat/2005">
  <HEADER><CATALOG><LANGUAGE>deu</LANGUAGE><CATALOG_ID>C</CATALOG_ID><CATALOG_VERSION>1</CATALOG_VERSION></CATALOG></HEADER>
  <T_NEW_CATALOG>
    <CATALOG_STRUCTURE type="node">
      <GROUP_ID>10</GROUP_ID>
      <GROUP_NAME>Hardware</GROUP_NAME>
    </CATALOG_STRUCTURE>
    <CLASSIFICATION_SYSTEM>
      <CLASSIFICATION_SYSTEM_NAME>udf_Supplier-1.0</CLASSIFICATION_SYSTEM_NAME>
      <CLASSIFICATION_GROUPS>
        <CLASSIFICATION_GROUP type="node">
          <CLASSIFICATION_GROUP_ID>1</CLASSIFICATION_GROUP_ID>
          <CLASSIFICATION_GROUP_NAME>Root</CLASSIFICATION_GROUP_NAME>
        </CLASSIFICATION_GROUP>
        <CLASSIFICATION_GROUP type="leaf">
          <CLASSIFICATION_GROUP_ID>2</CLASSIFICATION_GROUP_ID>
          <CLASSIFICATION_GROUP_NAME>Leaf</CLASSIFICATION_GROUP_NAME>
          <CLASSIFICATION_GROUP_PARENT_ID>1</CLASSIFICATION_GROUP_PARENT_ID>
        </CLASSIFICATION_GROUP>
      </CLASSIFICATION_GROUPS>
    </CLASSIFICATION_SYSTEM>
  </T_NEW_CATALOG>
</BMECAT>`

// groupCollector implements the catalog and classification group handlers, as
// well as completion, so a single value exercises the adapter dispatch.
type groupCollector struct {
	catalogGroups []*bmecat.CatalogGroup
	classifGroups []*bmecat.ClassificationGroup
	completed     int
}

func (g *groupCollector) HandleCatalogGroup(cg *bmecat.CatalogGroup) error {
	g.catalogGroups = append(g.catalogGroups, cg)
	return nil
}

func (g *groupCollector) HandleClassificationGroup(cg *bmecat.ClassificationGroup) error {
	g.classifGroups = append(g.classifGroups, cg)
	return nil
}

func (g *groupCollector) HandleComplete() { g.completed++ }

func TestReadGroupsDispatch(t *testing.T) {
	for _, tt := range []struct {
		name string
		doc  string
	}{
		{"1.2", catalog12Groups},
		{"2005", catalog2005Groups},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := &groupCollector{}
			r := bmecat.NewReader(bytes.NewReader([]byte(tt.doc)))
			if err := r.Do(context.Background(), g); err != nil {
				t.Fatal(err)
			}

			if want, have := 1, len(g.catalogGroups); want != have {
				t.Fatalf("want %d catalog group(s), have %d", want, have)
			}
			cg := g.catalogGroups[0]
			if want, have := "10", cg.ID; want != have {
				t.Errorf("want catalog group ID %q, have %q", want, have)
			}
			if want, have := "Hardware", cg.Name.Value(); want != have {
				t.Errorf("want catalog group name %q, have %q", want, have)
			}
			if !cg.IsNode() {
				t.Errorf("want catalog group IsNode() == true for type %q", cg.Type)
			}

			if want, have := 2, len(g.classifGroups); want != have {
				t.Fatalf("want %d classification group(s), have %d", want, have)
			}
			if want, have := "1", g.classifGroups[0].ID; want != have {
				t.Errorf("want classification group ID %q, have %q", want, have)
			}
			if !g.classifGroups[0].IsNode() {
				t.Errorf("want first classification group IsNode() == true")
			}
			if !g.classifGroups[1].IsLeaf() {
				t.Errorf("want second classification group IsLeaf() == true")
			}
			if want, have := "1", g.classifGroups[1].ParentID; want != have {
				t.Errorf("want classification group parent ID %q, have %q", want, have)
			}

			if want, have := 1, g.completed; want != have {
				t.Errorf("want HandleComplete called %d time(s), have %d", want, have)
			}
		})
	}
}

func TestNeutralCatalogGroupPredicates(t *testing.T) {
	tests := []struct {
		typ    string
		isRoot bool
		isNode bool
		isLeaf bool
	}{
		{"root", true, false, false},
		{"node", false, true, false},
		{"leaf", false, false, true},
		{"", false, false, false},
	}
	for _, tt := range tests {
		cg := &bmecat.CatalogGroup{Type: tt.typ}
		if got := cg.IsRoot(); got != tt.isRoot {
			t.Errorf("IsRoot(%q) = %v, want %v", tt.typ, got, tt.isRoot)
		}
		if got := cg.IsNode(); got != tt.isNode {
			t.Errorf("IsNode(%q) = %v, want %v", tt.typ, got, tt.isNode)
		}
		if got := cg.IsLeaf(); got != tt.isLeaf {
			t.Errorf("IsLeaf(%q) = %v, want %v", tt.typ, got, tt.isLeaf)
		}
	}
}

func TestNeutralClassificationGroupPredicates(t *testing.T) {
	if !(&bmecat.ClassificationGroup{Type: "node"}).IsNode() {
		t.Error("IsNode() = false for node, want true")
	}
	if (&bmecat.ClassificationGroup{Type: "leaf"}).IsNode() {
		t.Error("IsNode() = true for leaf, want false")
	}
	if !(&bmecat.ClassificationGroup{Type: "leaf"}).IsLeaf() {
		t.Error("IsLeaf() = false for leaf, want true")
	}
	if (&bmecat.ClassificationGroup{Type: "node"}).IsLeaf() {
		t.Error("IsLeaf() = true for node, want false")
	}
}

func TestNeutralFeaturesIsUnspsc(t *testing.T) {
	if !(bmecat.Features{SystemName: "UNSPSC-13.2"}).IsUnspsc() {
		t.Error("IsUnspsc() = false for UNSPSC system, want true")
	}
	if (bmecat.Features{SystemName: "ECLASS-5.1"}).IsUnspsc() {
		t.Error("IsUnspsc() = true for eCl@ss system, want false")
	}
}

func TestVersionString(t *testing.T) {
	if got, want := bmecat.Version12.String(), "1.2"; got != want {
		t.Errorf("Version12.String() = %q, want %q", got, want)
	}
	if got, want := bmecat.Version2005.String(), "2005"; got != want {
		t.Errorf("Version2005.String() = %q, want %q", got, want)
	}
}

func TestReadWithProgress(t *testing.T) {
	for _, tt := range []struct {
		name string
		doc  string
	}{
		{"1.2", catalog12},
		{"2005", catalog2005},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var maxPass int
			c := &collector{}
			r := bmecat.NewReader(
				bytes.NewReader([]byte(tt.doc)),
				bmecat.WithReaderProgress(func(pass int, _ int64) {
					if pass > maxPass {
						maxPass = pass
					}
				}),
			)
			if err := r.Do(context.Background(), c); err != nil {
				t.Fatal(err)
			}
			if maxPass < 1 {
				t.Errorf("want progress callback invoked at least once, max pass %d", maxPass)
			}
		})
	}
}

// catalog12Full and catalog2005Full carry the same product with every
// order-detail and detail field the neutral Product exposes. The 2005 document
// additionally sets QUANTITY_MAX, which has no 1.2 equivalent.
const catalog12Full = `<?xml version="1.0" encoding="UTF-8"?>
<BMECAT version="1.2">
  <HEADER><CATALOG><LANGUAGE>deu</LANGUAGE><CATALOG_ID>C</CATALOG_ID><CATALOG_VERSION>1</CATALOG_VERSION></CATALOG></HEADER>
  <T_NEW_CATALOG>
    <ARTICLE>
      <SUPPLIER_AID>1000</SUPPLIER_AID>
      <ARTICLE_DETAILS>
        <DESCRIPTION_SHORT>Widget</DESCRIPTION_SHORT>
        <BUYER_AID type="buyer">BPN-1</BUYER_AID>
        <MANUFACTURER_TYPE_DESCR>Type-X</MANUFACTURER_TYPE_DESCR>
        <ERP_GROUP_BUYER>EGB</ERP_GROUP_BUYER>
        <ERP_GROUP_SUPPLIER>EGS</ERP_GROUP_SUPPLIER>
        <DELIVERY_TIME>5</DELIVERY_TIME>
        <SPECIAL_TREATMENT_CLASS type="WEEE">cat1</SPECIAL_TREATMENT_CLASS>
        <KEYWORD>tool</KEYWORD>
        <REMARKS>handle with care</REMARKS>
        <SEGMENT>seg-a</SEGMENT>
        <ARTICLE_STATUS type="new">live</ARTICLE_STATUS>
      </ARTICLE_DETAILS>
      <ARTICLE_ORDER_DETAILS>
        <ORDER_UNIT>C62</ORDER_UNIT>
        <CONTENT_UNIT>PCE</CONTENT_UNIT>
        <NO_CU_PER_OU>10</NO_CU_PER_OU>
        <PRICE_QUANTITY>100</PRICE_QUANTITY>
        <QUANTITY_MIN>5</QUANTITY_MIN>
        <QUANTITY_INTERVAL>5</QUANTITY_INTERVAL>
      </ARTICLE_ORDER_DETAILS>
      <USER_DEFINED_EXTENSIONS>
        <UDX.SYSTEM.CUSTOM_FIELD1>custom</UDX.SYSTEM.CUSTOM_FIELD1>
      </USER_DEFINED_EXTENSIONS>
    </ARTICLE>
  </T_NEW_CATALOG>
</BMECAT>`

const catalog2005Full = `<?xml version="1.0" encoding="UTF-8"?>
<BMECAT version="2005" xmlns="http://www.bmecat.org/bmecat/2005">
  <HEADER><CATALOG><LANGUAGE>deu</LANGUAGE><CATALOG_ID>C</CATALOG_ID><CATALOG_VERSION>1</CATALOG_VERSION></CATALOG></HEADER>
  <T_NEW_CATALOG>
    <PRODUCT>
      <SUPPLIER_PID>1000</SUPPLIER_PID>
      <PRODUCT_DETAILS>
        <DESCRIPTION_SHORT>Widget</DESCRIPTION_SHORT>
        <BUYER_PID type="buyer">BPN-1</BUYER_PID>
        <MANUFACTURER_TYPE_DESCR>Type-X</MANUFACTURER_TYPE_DESCR>
        <ERP_GROUP_BUYER>EGB</ERP_GROUP_BUYER>
        <ERP_GROUP_SUPPLIER>EGS</ERP_GROUP_SUPPLIER>
        <DELIVERY_TIME>5</DELIVERY_TIME>
        <SPECIAL_TREATMENT_CLASS type="WEEE">cat1</SPECIAL_TREATMENT_CLASS>
        <KEYWORD>tool</KEYWORD>
        <REMARKS>handle with care</REMARKS>
        <SEGMENT>seg-a</SEGMENT>
        <PRODUCT_STATUS type="new">live</PRODUCT_STATUS>
      </PRODUCT_DETAILS>
      <PRODUCT_ORDER_DETAILS>
        <ORDER_UNIT>C62</ORDER_UNIT>
        <CONTENT_UNIT>PCE</CONTENT_UNIT>
        <NO_CU_PER_OU>10</NO_CU_PER_OU>
        <PRICE_QUANTITY>100</PRICE_QUANTITY>
        <QUANTITY_MIN>5</QUANTITY_MIN>
        <QUANTITY_INTERVAL>5</QUANTITY_INTERVAL>
        <QUANTITY_MAX>500</QUANTITY_MAX>
      </PRODUCT_ORDER_DETAILS>
      <USER_DEFINED_EXTENSIONS>
        <UDX.SYSTEM.CUSTOM_FIELD1>custom</UDX.SYSTEM.CUSTOM_FIELD1>
      </USER_DEFINED_EXTENSIONS>
    </PRODUCT>
  </T_NEW_CATALOG>
</BMECAT>`

func TestNeutralProductOrderAndDetailFields(t *testing.T) {
	for _, tt := range []struct {
		name            string
		doc             string
		wantQuantityMax float64
	}{
		{"1.2", catalog12Full, 0}, // 1.2 has no QUANTITY_MAX
		{"2005", catalog2005Full, 500},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := read(t, tt.doc)
			if want, have := 1, len(c.products); want != have {
				t.Fatalf("want %d product(s), have %d", want, have)
			}
			p := c.products[0]

			// Order details.
			if want, have := "C62", p.OrderUnit; want != have {
				t.Errorf("OrderUnit = %q, want %q", have, want)
			}
			if want, have := "PCE", p.ContentUnit; want != have {
				t.Errorf("ContentUnit = %q, want %q", have, want)
			}
			if want, have := 10.0, p.NoCuPerOu; want != have {
				t.Errorf("NoCuPerOu = %v, want %v", have, want)
			}
			if want, have := 100.0, p.PriceQuantity; want != have {
				t.Errorf("PriceQuantity = %v, want %v", have, want)
			}
			if want, have := 5.0, p.QuantityMin; want != have {
				t.Errorf("QuantityMin = %v, want %v", have, want)
			}
			if want, have := 5.0, p.QuantityInterval; want != have {
				t.Errorf("QuantityInterval = %v, want %v", have, want)
			}
			if want, have := tt.wantQuantityMax, p.QuantityMax; want != have {
				t.Errorf("QuantityMax = %v, want %v", have, want)
			}

			// Article/product details.
			if want, have := "Type-X", p.ManufacturerTypeDescr.Value(); want != have {
				t.Errorf("ManufacturerTypeDescr = %q, want %q", have, want)
			}
			if want, have := "EGB", p.ERPGroupBuyer; want != have {
				t.Errorf("ERPGroupBuyer = %q, want %q", have, want)
			}
			if want, have := "EGS", p.ERPGroupSupplier; want != have {
				t.Errorf("ERPGroupSupplier = %q, want %q", have, want)
			}
			if p.DeliveryTime == nil {
				t.Errorf("DeliveryTime = nil, want 5")
			} else if want, have := 5, *p.DeliveryTime; want != have {
				t.Errorf("DeliveryTime = %d, want %d", have, want)
			}
			if want, have := "handle with care", p.Remarks.Value(); want != have {
				t.Errorf("Remarks = %q, want %q", have, want)
			}
			if want, have := []string{"seg-a"}, p.Segments.All(""); len(have) != 1 || have[0] != want[0] {
				t.Errorf("Segments = %v, want %v", have, want)
			}
			if want, have := 1, len(p.BuyerIDs); want != have {
				t.Fatalf("len(BuyerIDs) = %d, want %d", have, want)
			}
			if want, have := (bmecat.TypedValue{Type: "buyer", Value: "BPN-1"}), *p.BuyerIDs[0]; want != have {
				t.Errorf("BuyerIDs[0] = %+v, want %+v", have, want)
			}
			if want, have := 1, len(p.SpecialTreatmentClasses); want != have {
				t.Fatalf("len(SpecialTreatmentClasses) = %d, want %d", have, want)
			}
			if want, have := (bmecat.TypedValue{Type: "WEEE", Value: "cat1"}), *p.SpecialTreatmentClasses[0]; want != have {
				t.Errorf("SpecialTreatmentClasses[0] = %+v, want %+v", have, want)
			}
			if want, have := 1, len(p.Status); want != have {
				t.Fatalf("len(Status) = %d, want %d", have, want)
			}
			if want, have := (bmecat.TypedValue{Type: "new", Value: "live"}), *p.Status[0]; want != have {
				t.Errorf("Status[0] = %+v, want %+v", have, want)
			}

			// UDX.
			if want, have := 1, len(p.UDX); want != have {
				t.Fatalf("len(UDX) = %d, want %d", have, want)
			}
			if want, have := (bmecat.UDXField{Name: "SYSTEM.CUSTOM_FIELD1", Value: "custom"}), *p.UDX[0]; want != have {
				t.Errorf("UDX[0] = %+v, want %+v", have, want)
			}
		})
	}
}

// TestNeutralGTINFromV2005 verifies that the 2005 adapter resolves Product.GTIN
// from the INTERNATIONAL_PID whose type is gtin/ean, rather than blindly taking
// the first PID. See issue #51.
func TestNeutralGTINFromV2005(t *testing.T) {
	doc := func(pids, ean string) string {
		return `<?xml version="1.0" encoding="UTF-8"?>
<BMECAT version="2005" xmlns="http://www.bmecat.org/bmecat/2005">
  <HEADER><CATALOG><LANGUAGE>eng</LANGUAGE><CATALOG_ID>C</CATALOG_ID><CATALOG_VERSION>1</CATALOG_VERSION></CATALOG></HEADER>
  <T_NEW_CATALOG>
    <PRODUCT>
      <SUPPLIER_PID>P1</SUPPLIER_PID>
      <PRODUCT_DETAILS>
        <DESCRIPTION_SHORT>Widget</DESCRIPTION_SHORT>
        ` + pids + ean + `
      </PRODUCT_DETAILS>
    </PRODUCT>
  </T_NEW_CATALOG>
</BMECAT>`
	}
	for _, tt := range []struct {
		name string
		pids string
		ean  string
		want string
	}{
		{
			name: "typed gtin preceded by non-gtin PID",
			pids: `<INTERNATIONAL_PID type="supplier_specific">ACME-001</INTERNATIONAL_PID>
        <INTERNATIONAL_PID type="gtin">4006381333931</INTERNATIONAL_PID>`,
			want: "4006381333931",
		},
		{
			name: "typed ean preferred over leading non-gtin PID",
			pids: `<INTERNATIONAL_PID type="supplier_specific">ACME-001</INTERNATIONAL_PID>
        <INTERNATIONAL_PID type="EAN">4006381333931</INTERNATIONAL_PID>`,
			want: "4006381333931",
		},
		{
			name: "untyped PID falls back to first non-empty",
			pids: `<INTERNATIONAL_PID>4006381333931</INTERNATIONAL_PID>`,
			want: "4006381333931",
		},
		{
			name: "no typed gtin/ean falls back to first PID",
			pids: `<INTERNATIONAL_PID type="supplier_specific">ACME-001</INTERNATIONAL_PID>`,
			want: "ACME-001",
		},
		{
			name: "legacy EAN element when no PID present",
			ean:  `<EAN>4006381333931</EAN>`,
			want: "4006381333931",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := read(t, doc(tt.pids, tt.ean))
			if want, have := 1, len(c.products); want != have {
				t.Fatalf("want %d product(s), have %d", want, have)
			}
			if want, have := tt.want, c.products[0].GTIN; want != have {
				t.Errorf("GTIN = %q, want %q", have, want)
			}
		})
	}
}
