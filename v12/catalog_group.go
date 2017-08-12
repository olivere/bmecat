package v12

import "encoding/xml"

type CatalogGroup struct {
	XMLName xml.Name `xml:"CATALOG_STRUCTURE"`

	Type        string  `xml:"type,attr,omitempty"`
	ID          string  `xml:"GROUP_ID"`
	Name        string  `xml:"GROUP_NAME"`
	Description string  `xml:"GROUP_DESCRIPTION,omitempty"`
	ParentID    *string `xml:"PARENT_ID,omitempty"`
	Order       int     `xml:"GROUP_ORDER,omitempty"`
	// MIME_INFO
	// USER_DEFINED_EXTENSIONS
	Keywords []string `xml:"KEYWORD,omitempty"`
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

type ArticleToCatalogGroupMap struct {
	XMLName xml.Name `xml:"ARTICLE_TO_CATALOGGROUP_MAP"`

	ArticleID      string `xml:"ART_ID"`
	CatalogGroupID string `xml:"CATALOG_GROUP_ID"`
}
