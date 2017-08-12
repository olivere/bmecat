package v12

import "encoding/xml"

type ClassificationGroup struct {
	XMLName xml.Name `xml:"CLASSIFICATION_GROUP"`

	Type        string   `xml:"type,attr,omitempty"`
	Level       *int     `xml:"level,attr,omitempty"`
	ID          string   `xml:"CLASSIFICATION_GROUP_ID"`
	Name        string   `xml:"CLASSIFICATION_GROUP_NAME"`
	Description string   `xml:"CLASSIFICATION_GROUP_DESCR,omitempty"`
	Synonyms    []string `xml:"CLASSIFICATION_GROUP_SYNONYMS>SYNONYM,omitempty"`
	// CLASSIFICATION_GROUP_FEATURE_TEMPLATES
	ParentID *string `xml:"CLASSIFICATION_GROUP_PARENT_ID,omitempty"`
}

func (cg *ClassificationGroup) IsNode() bool {
	return cg.Type == "node"
}

func (cg *ClassificationGroup) IsLeaf() bool {
	return cg.Type == "leaf"
}
