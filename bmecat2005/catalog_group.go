package bmecat2005

import "encoding/xml"

type CatalogGroup struct {
	XMLName xml.Name `xml:"CATALOG_STRUCTURE"`

	Type        string           `xml:"type,attr,omitempty"`
	ID          string           `xml:"GROUP_ID"`
	Name        LocalizedStrings `xml:"GROUP_NAME"`
	Description LocalizedStrings `xml:"GROUP_DESCRIPTION,omitempty"`
	ParentID    *string          `xml:"PARENT_ID,omitempty"`
	Order       int              `xml:"GROUP_ORDER,omitempty"`
	// MIME_INFO
	// USER_DEFINED_EXTENSIONS
	Keywords LocalizedStrings `xml:"KEYWORD,omitempty"`
}

func (cg *CatalogGroup) IsRoot() bool {
	return cg.Type == "root"
}

func (cg *CatalogGroup) IsNode() bool {
	return cg.Type == "node"
}

func (cg *CatalogGroup) IsLeaf() bool {
	return cg.Type == "leaf"
}

// ProductToCatalogGroupMap is the BMEcat 2005 equivalent of the 1.2
// ARTICLE_TO_CATALOGGROUP_MAP: it links a product (PROD_ID) to a catalog
// group (CATALOG_GROUP_ID).
type ProductToCatalogGroupMap struct {
	XMLName xml.Name `xml:"PRODUCT_TO_CATALOGGROUP_MAP"`

	ProductID      string `xml:"PROD_ID"`
	CatalogGroupID string `xml:"CATALOG_GROUP_ID"`
}
