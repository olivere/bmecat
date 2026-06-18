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
			if want, have := "Hardware", cg.Name; want != have {
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
