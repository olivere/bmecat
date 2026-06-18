package bmecat2005

import (
	"encoding/xml"
)

type ClassificationSystem struct {
	XMLName xml.Name `xml:"CLASSIFICATION_SYSTEM"`

	Name        string                           `xml:"CLASSIFICATION_SYSTEM_NAME"`
	FullName    string                           `xml:"CLASSIFICATION_SYSTEM_FULLNAME,omitempty"`
	Version     string                           `xml:"CLASSIFICATION_SYSTEM_VERSION,omitempty"`
	Description string                           `xml:"CLASSIFICATION_SYSTEM_DESCR,omitempty"`
	Levels      int                              `xml:"CLASSIFICATION_SYSTEM_LEVELS,omitempty"`
	LevelNames  []*ClassificationSystemLevelName `xml:"CLASSIFICATION_SYSTEM_LEVEL_NAMES,omitempty"`
	// ALLOWED_VALUES
	// UNITS
	// CLASSIFICATION_SYSTEM_FEATURE_TEMPLATES
	Groups []*ClassificationGroup `xml:"CLASSIFICATION_GROUPS>CLASSIFICATION_GROUP,omitempty"`
}

// IsBlank returns true if there are no groups in the classification system.
func (cs *ClassificationSystem) IsBlank() bool {
	return cs == nil || len(cs.Groups) == 0
}

type ClassificationSystemLevelName struct {
	XMLName xml.Name `xml:"CLASSIFICATION_SYSTEM_LEVEL_NAME"`

	Level int    `xml:"level,attr"`
	Value string `xml:",innerxml"`
}

type ClassificationGroup struct {
	XMLName xml.Name `xml:"CLASSIFICATION_GROUP"`

	Type        string                       `xml:"type,attr,omitempty"`
	Level       *int                         `xml:"level,attr,omitempty"`
	ID          string                       `xml:"CLASSIFICATION_GROUP_ID"`
	Name        string                       `xml:"CLASSIFICATION_GROUP_NAME"`
	Description string                       `xml:"CLASSIFICATION_GROUP_DESCR,omitempty"`
	Synonyms    []ClassificationGroupSynonym `xml:"CLASSIFICATION_GROUP_SYNONYMS,omitempty"`
	// CLASSIFICATION_GROUP_FEATURE_TEMPLATES
	ParentID string `xml:"CLASSIFICATION_GROUP_PARENT_ID,omitempty"`
}

type ClassificationGroupSynonym struct {
	Value string `xml:"SYNONYM,omitempty"`
}

func (cg *ClassificationGroup) IsNode() bool {
	return cg.Type == "node"
}

func (cg *ClassificationGroup) IsLeaf() bool {
	return cg.Type == "leaf"
}
