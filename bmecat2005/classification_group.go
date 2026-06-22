package bmecat2005

import (
	"encoding/xml"
)

type ClassificationSystem struct {
	XMLName xml.Name `xml:"CLASSIFICATION_SYSTEM"`

	Name        string                         `xml:"CLASSIFICATION_SYSTEM_NAME"`
	FullName    LocalizedStrings               `xml:"CLASSIFICATION_SYSTEM_FULLNAME,omitempty"`
	Version     string                         `xml:"CLASSIFICATION_SYSTEM_VERSION,omitempty"`
	Description LocalizedStrings               `xml:"CLASSIFICATION_SYSTEM_DESCR,omitempty"`
	Levels      int                            `xml:"CLASSIFICATION_SYSTEM_LEVELS,omitempty"`
	LevelNames  ClassificationSystemLevelNames `xml:"CLASSIFICATION_SYSTEM_LEVEL_NAMES,omitempty"`
	// ALLOWED_VALUES
	// UNITS
	// CLASSIFICATION_SYSTEM_FEATURE_TEMPLATES
	Groups []*ClassificationGroup `xml:"CLASSIFICATION_GROUPS>CLASSIFICATION_GROUP,omitempty"`
}

// IsBlank returns true if there are no groups in the classification system.
func (cs *ClassificationSystem) IsBlank() bool {
	return cs == nil || len(cs.Groups) == 0
}

// ClassificationSystemLevelNames is the CLASSIFICATION_SYSTEM_LEVEL_NAMES
// wrapper around the CLASSIFICATION_SYSTEM_LEVEL_NAME list. It marshals the
// wrapper only when it holds entries, so an empty list emits nothing (the DTD
// requires at least one name inside the wrapper).
type ClassificationSystemLevelNames []*ClassificationSystemLevelName

func (s ClassificationSystemLevelNames) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if len(s) == 0 {
		return nil
	}
	type wrapper struct {
		Names []*ClassificationSystemLevelName `xml:"CLASSIFICATION_SYSTEM_LEVEL_NAME"`
	}
	return e.EncodeElement(wrapper{Names: s}, start)
}

func (s *ClassificationSystemLevelNames) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var w struct {
		Names []*ClassificationSystemLevelName `xml:"CLASSIFICATION_SYSTEM_LEVEL_NAME"`
	}
	if err := d.DecodeElement(&w, &start); err != nil {
		return err
	}
	*s = w.Names
	return nil
}

type ClassificationSystemLevelName struct {
	XMLName xml.Name `xml:"CLASSIFICATION_SYSTEM_LEVEL_NAME"`

	Level int `xml:"level,attr"`
	// Lang is the xml lang attribute; CLASSIFICATION_SYSTEM_LEVEL_NAME is
	// localized in 2005, so the same level may appear once per language.
	Lang  string `xml:"lang,attr,omitempty"`
	Value string `xml:",chardata"`
}

type ClassificationGroup struct {
	XMLName xml.Name `xml:"CLASSIFICATION_GROUP"`

	Type        string                       `xml:"type,attr,omitempty"`
	Level       *int                         `xml:"level,attr,omitempty"`
	ID          string                       `xml:"CLASSIFICATION_GROUP_ID"`
	Name        LocalizedStrings             `xml:"CLASSIFICATION_GROUP_NAME"`
	Description LocalizedStrings             `xml:"CLASSIFICATION_GROUP_DESCR,omitempty"`
	Synonyms    []ClassificationGroupSynonym `xml:"CLASSIFICATION_GROUP_SYNONYMS,omitempty"`
	// CLASSIFICATION_GROUP_FEATURE_TEMPLATES
	ParentID string `xml:"CLASSIFICATION_GROUP_PARENT_ID,omitempty"`
}

type ClassificationGroupSynonym struct {
	Value LocalizedStrings `xml:"SYNONYM,omitempty"`
}

func (cg *ClassificationGroup) IsNode() bool {
	return cg.Type == "node"
}

func (cg *ClassificationGroup) IsLeaf() bool {
	return cg.Type == "leaf"
}
