package bmecat

import (
	"strings"
	"unicode/utf8"
)

// Version identifies the BMEcat version a document was written in.
type Version int

const (
	// Version12 is BMEcat 1.2.
	Version12 Version = iota
	// Version2005 is BMEcat 2005 (also known as BMEcat 2.0).
	Version2005
)

// String returns the version attribute as it appears in a BMEcat document.
func (v Version) String() string {
	switch v {
	case Version2005:
		return "2005"
	default:
		return "1.2"
	}
}

// Header is the version-neutral view of a BMEcat HEADER. It exposes the fields
// that 1.2 and 2005 have in common.
type Header struct {
	// Version is the BMEcat version the document was read from.
	Version Version

	GeneratorInfo string
	Catalog       *Catalog
	Buyer         *Buyer
	Supplier      *Supplier

	NumberOfProducts             int
	NumberOfCatalogGroups        int
	NumberOfClassificationGroups int
}

// Catalog is the version-neutral view of the CATALOG element.
type Catalog struct {
	Language    string
	ID          string
	Version     string
	Name        string
	Currency    string
	Territories []string
}

// Buyer is the version-neutral view of the BUYER element.
type Buyer struct {
	ID   string
	Name string
}

// Supplier is the version-neutral view of the SUPPLIER element.
type Supplier struct {
	ID   string
	Name string
}

// Product is the version-neutral view of a catalog product. It maps a 1.2
// ARTICLE and a 2005 PRODUCT onto a single shape, so a caller can ingest both
// versions through one mapping.
type Product struct {
	// ID is SUPPLIER_AID (1.2) or SUPPLIER_PID (2005).
	ID string
	// Mode is the product's transaction mode ("new", "update" or "delete").
	// It is typically only set in T_UPDATE_PRODUCTS catalogs.
	Mode string
	// GTIN is the global trade item number: EAN (1.2) or the first
	// INTERNATIONAL_PID (2005). See GTIN handling in the package docs.
	GTIN string

	DescriptionShort string
	DescriptionLong  string
	SupplierAltID    string
	ManufacturerID   string
	ManufacturerName string
	Keywords         []string

	Features  []*Features
	OrderUnit string
	Prices    []*Price
	Mimes     []*Mime

	// CatalogGroupIDs lists the catalog group IDs this product is mapped to.
	CatalogGroupIDs []string
}

// Features is the version-neutral view of ARTICLE_FEATURES (1.2) /
// PRODUCT_FEATURES (2005).
type Features struct {
	SystemName string
	GroupID    string
	GroupName  string
	Features   []*Feature
}

// IsEclass reports whether the feature system is an eCl@ss classification.
func (f Features) IsEclass() bool {
	return strings.HasPrefix(strings.ToUpper(f.SystemName), "ECLASS")
}

// IsUnspsc reports whether the feature system is a UNSPSC classification.
func (f Features) IsUnspsc() bool {
	return strings.HasPrefix(strings.ToUpper(f.SystemName), "UNSPSC")
}

// Version returns the classification system version, e.g. "5.1" for
// "ECLASS-5.1", or the empty string when no version is encoded.
func (f Features) Version() string {
	parts := strings.SplitN(f.SystemName, "-", 2)
	if len(parts) == 2 && utf8.RuneCountInString(parts[1]) > 0 {
		return parts[1]
	}
	return ""
}

// Feature is the version-neutral view of a single FEATURE.
type Feature struct {
	Name   string
	Values []string
	Unit   string
}

// Price is the version-neutral view of an ARTICLE_PRICE (1.2) / PRODUCT_PRICE
// (2005).
type Price struct {
	Type       string
	Amount     float64
	Currency   string
	Tax        *float64
	Factor     float64
	LowerBound float64
	Territory  []string
}

// Mime is the version-neutral view of a MIME element.
type Mime struct {
	Type    string
	Source  string
	Descr   string
	Purpose string
	Order   int
}

// CatalogGroup is the version-neutral view of a CATALOG_STRUCTURE element.
type CatalogGroup struct {
	Type        string
	ID          string
	Name        string
	Description string
	ParentID    *string
	Order       int
}

func (cg *CatalogGroup) IsRoot() bool { return cg.Type == "root" }
func (cg *CatalogGroup) IsNode() bool { return cg.Type == "node" }
func (cg *CatalogGroup) IsLeaf() bool { return cg.Type == "leaf" }

// ClassificationGroup is the version-neutral view of a CLASSIFICATION_GROUP
// element.
type ClassificationGroup struct {
	Type        string
	ID          string
	Name        string
	Description string
	ParentID    string
}

func (cg *ClassificationGroup) IsNode() bool { return cg.Type == "node" }
func (cg *ClassificationGroup) IsLeaf() bool { return cg.Type == "leaf" }
