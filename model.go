package bmecat

import (
	"strings"
	"time"
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

// Transaction identifies the document-level BMEcat transaction: a full catalog
// (NewCatalog) or an incremental update (UpdateProducts / UpdatePrices). It is
// the wrapping T_NEW_CATALOG / T_UPDATE_PRODUCTS / T_UPDATE_PRICES element, not
// the per-product Product.Mode.
type Transaction int

const (
	// NewCatalog is a full catalog: T_NEW_CATALOG.
	NewCatalog Transaction = iota
	// UpdateProducts is an incremental product update: T_UPDATE_PRODUCTS.
	UpdateProducts
	// UpdatePrices is an incremental price update: T_UPDATE_PRICES.
	UpdatePrices
)

// String returns the transaction element name as it appears in a BMEcat
// document, e.g. "T_NEW_CATALOG".
func (t Transaction) String() string {
	switch t {
	case UpdateProducts:
		return "T_UPDATE_PRODUCTS"
	case UpdatePrices:
		return "T_UPDATE_PRICES"
	default:
		return "T_NEW_CATALOG"
	}
}

// IsUpdate reports whether the transaction is an incremental update
// (T_UPDATE_PRODUCTS or T_UPDATE_PRICES) rather than a full catalog. A consumer
// that only supports full catalogs can reject updates with this.
func (t Transaction) IsUpdate() bool {
	return t == UpdateProducts || t == UpdatePrices
}

// transactionFromElement maps a transaction start-element name onto a
// Transaction. The second result is false for any other element.
func transactionFromElement(name string) (Transaction, bool) {
	switch name {
	case "T_NEW_CATALOG":
		return NewCatalog, true
	case "T_UPDATE_PRODUCTS":
		return UpdateProducts, true
	case "T_UPDATE_PRICES":
		return UpdatePrices, true
	default:
		return 0, false
	}
}

// Header is the version-neutral view of a BMEcat HEADER. It exposes the fields
// that 1.2 and 2005 have in common.
type Header struct {
	// Version is the BMEcat version the document was read from.
	Version Version
	// Transaction is the document-level transaction the products are wrapped
	// in: NewCatalog for a full catalog, UpdateProducts / UpdatePrices for an
	// incremental update. It is distinct from the per-product Product.Mode.
	Transaction Transaction

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
	// ManufacturerTypeDescr is MANUFACTURER_TYPE_DESCR.
	ManufacturerTypeDescr string
	Keywords              []string
	// Remarks is REMARKS.
	Remarks string
	// Segments is the list of SEGMENT values.
	Segments []string

	// ERPGroupBuyer and ERPGroupSupplier are ERP_GROUP_BUYER /
	// ERP_GROUP_SUPPLIER.
	ERPGroupBuyer    string
	ERPGroupSupplier string
	// DeliveryTime is DELIVERY_TIME (the lead time). It is nil when the source
	// document omits it.
	DeliveryTime *int
	// BuyerIDs are the buyer-specific product identifiers: BUYER_AID (1.2) /
	// BUYER_PID (2005).
	BuyerIDs []*TypedValue
	// SpecialTreatmentClasses are SPECIAL_TREATMENT_CLASS entries.
	SpecialTreatmentClasses []*TypedValue
	// Status are the ARTICLE_STATUS (1.2) / PRODUCT_STATUS (2005) entries.
	Status []*TypedValue

	Features []*Features

	// Order detail fields, from ARTICLE_ORDER_DETAILS (1.2) /
	// PRODUCT_ORDER_DETAILS (2005).
	OrderUnit        string
	ContentUnit      string
	NoCuPerOu        float64
	PriceQuantity    float64
	QuantityMin      float64
	QuantityInterval float64
	// QuantityMax is QUANTITY_MAX. It exists only in BMEcat 2005; for 1.2 it is
	// always zero, because 1.2 has no QUANTITY_MAX element.
	QuantityMax float64

	// Prices is the flattened list of every price block across all
	// PriceDetails wrappers, in document order. It is a convenience view that
	// loses the wrapper boundary and validity dates; consumers that need those
	// should read PriceDetails instead.
	Prices []*Price
	// PriceDetails preserves the ARTICLE_PRICE_DETAILS (1.2) /
	// PRODUCT_PRICE_DETAILS (2005) wrapper grouping, including each wrapper's
	// validity dates. Use it to pick the currently-valid block or to detect
	// price calendars (several dated wrappers).
	PriceDetails []*PriceDetails
	Mimes        []*Mime

	// UDX carries USER_DEFINED_EXTENSIONS as neutral name/value pairs. Callers
	// that need raw or nested UDX XML should read the bmecat12/bmecat2005
	// packages directly.
	UDX []*UDXField

	// CatalogGroupIDs lists the catalog group IDs this product is mapped to.
	CatalogGroupIDs []string
}

// TypedValue is the version-neutral view of an element that carries a type
// attribute and a character-data value, such as BUYER_AID/BUYER_PID,
// SPECIAL_TREATMENT_CLASS, and ARTICLE_STATUS/PRODUCT_STATUS.
type TypedValue struct {
	Type  string
	Value string
}

// UDXField is a single user-defined extension, the version-neutral view of a
// USER_DEFINED_EXTENSIONS child element. Name is the field name without the
// "UDX." prefix (e.g. "SYSTEM.CUSTOM_FIELD1").
type UDXField struct {
	Name  string
	Value string
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

// PriceDetails is the version-neutral view of an ARTICLE_PRICE_DETAILS (1.2) /
// PRODUCT_PRICE_DETAILS (2005) wrapper. It groups one or more Prices that share
// a validity window, preserving the wrapper boundary that Product.Prices
// flattens away.
type PriceDetails struct {
	// ValidStart and ValidEnd are the wrapper's validity dates, read from its
	// valid_start_date / valid_end_date DATETIME entries. They are nil when the
	// source document omits the date; unlike the version-specific
	// ValidStartDate/ValidEndDate accessors, no sentinel default is substituted,
	// so a consumer can distinguish "no date" from a real bound.
	ValidStart *time.Time
	ValidEnd   *time.Time
	// IsDailyPrice reports whether the wrapper is flagged as a daily price
	// (DAILY_PRICE).
	IsDailyPrice bool
	// Prices are the price blocks grouped under this wrapper.
	Prices []*Price
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

// ClassificationSystem is the version-neutral view of a CLASSIFICATION_SYSTEM
// element: the classification (e.g. eCl@ss or a supplier UDF system) and its
// tree of ClassificationGroups. Unlike the streamed product list it is bounded
// and known up front, so the write path takes it as Writer configuration via
// WithClassificationSystem rather than over a channel; Writer emits it before
// the product stream in a T_NEW_CATALOG document.
//
// It carries the fields BMEcat 1.2 and 2005 share. Group-level details the
// neutral ClassificationGroup does not model (the level attribute, synonyms)
// are not emitted; callers that need them should use the bmecat12 / bmecat2005
// packages directly.
type ClassificationSystem struct {
	Name        string
	FullName    string
	Version     string
	Description string
	Levels      int
	LevelNames  []*ClassificationSystemLevelName
	Groups      []*ClassificationGroup
}

// IsBlank reports whether the system carries no groups. A blank system is
// omitted by the writer, mirroring the bmecat12 / bmecat2005 behavior.
func (cs *ClassificationSystem) IsBlank() bool {
	return cs == nil || len(cs.Groups) == 0
}

// ClassificationSystemLevelName is the version-neutral view of a
// CLASSIFICATION_SYSTEM_LEVEL_NAME element: the human-readable name of one
// level in the classification hierarchy.
type ClassificationSystemLevelName struct {
	Level int
	Name  string
}
