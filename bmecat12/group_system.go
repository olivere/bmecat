package bmecat12

import "encoding/xml"

type GroupSystem struct {
	XMLName xml.Name `xml:"CATALOG_GROUP_SYSTEM"`

	ID   string `xml:"GROUP_SYSTEM_ID,omitempty"`
	Name string `xml:"GROUP_SYSTEM_NAME,omitempty"`

	Structure []*GroupStructure `xml:"CATALOG_STRUCTURE"`
}

// IsBlank returns true if there are no structure items in the group system.
func (gs *GroupSystem) IsBlank() bool {
	return gs == nil || len(gs.Structure) == 0
}

type GroupStructure struct {
	XMLName xml.Name `xml:"CATALOG_STRUCTURE"`

	Type     string    `xml:"type,attr,omitempty"`
	ID       string    `xml:"GROUP_ID,omitempty"`
	Name     string    `xml:"GROUP_NAME,omitempty"`
	ParentID string    `xml:"PARENT_ID,omitempty"`
	Order    int       `xml:"GROUP_ORDER,omitempty"`
	MimeInfo *MimeInfo `xml:"MIME_INFO,omitempty"`
}

func (gs *GroupStructure) IsRoot() bool {
	return gs.Type == "root"
}

func (gs *GroupStructure) IsNode() bool {
	return gs.Type == "node"
}

func (gs *GroupStructure) IsLeaf() bool {
	return gs.Type == "leaf"
}
