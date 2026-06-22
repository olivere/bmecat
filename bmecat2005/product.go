package bmecat2005

import (
	"encoding/xml"
	"strings"
	"time"
	"unicode/utf8"
)

// Product represents a product according to the BMEcat 2005 specification.
// It is the 2005 counterpart of the bmecat12 Article type: ARTICLE becomes
// PRODUCT, ARTICLE_DETAILS becomes PRODUCT_DETAILS, and so on.
type Product struct {
	XMLName xml.Name `xml:"PRODUCT"`

	Mode            string                  `xml:"mode,attr,omitempty"`
	SupplierPID     string                  `xml:"SUPPLIER_PID"`
	Details         *ProductDetails         `xml:"PRODUCT_DETAILS"`
	Features        []*ProductFeatures      `xml:"PRODUCT_FEATURES,omitempty"`
	OrderDetails    *ProductOrderDetails    `xml:"PRODUCT_ORDER_DETAILS"`
	PriceDetails    []*ProductPriceDetails  `xml:"PRODUCT_PRICE_DETAILS"`
	MimeInfo        *MimeInfo               `xml:"MIME_INFO,omitempty"`
	UDX             *UserDefinedExtensions  `xml:"USER_DEFINED_EXTENSIONS,omitempty"`
	References      []*ProductReference     `xml:"PRODUCT_REFERENCE,omitempty"`
	LogisticDetails *ProductLogisticDetails `xml:"PRODUCT_LOGISTIC_DETAILS,omitempty"`

	// CatalogGroupIDs is the list of CATALOG_STRUCTURE IDs gathered on the 1st pass of the parser.
	CatalogGroupIDs []string `xml:"-"`
}

const (
	ProductStatusBargain     = "bargain"
	ProductStatusCoreProduct = "core_product"
	ProductStatusNew         = "new"
	ProductStatusNewProduct  = "new_product"
	ProductStatusOldProduct  = "old_product"
	ProductStatusRefurbished = "refurbished"
	ProductStatusUsed        = "used"
	ProductStatusOthers      = "others"
)

type ProductStatus struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type ProductSpecialTreatmentClass struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

// InternationalPID replaces the 1.2 EAN element. It carries a type attribute
// (e.g. "gtin" or "ean") that identifies the kind of international product
// identifier.
type InternationalPID struct {
	Type  string `xml:"type,attr,omitempty"`
	Value string `xml:",chardata"`
}

type ProductDetails struct {
	DescriptionShort  LocalizedStrings    `xml:"DESCRIPTION_SHORT"`
	DescriptionLong   LocalizedStrings    `xml:"DESCRIPTION_LONG,omitempty"`
	InternationalPIDs []*InternationalPID `xml:"INTERNATIONAL_PID,omitempty"`
	// EAN is accepted when reading 2005 documents that still use the
	// 1.2-compatible EAN element. The DTD treats INTERNATIONAL_PID and EAN as
	// mutually exclusive, so prefer InternationalPIDs for new documents and do
	// not set both.
	EAN                     string                          `xml:"EAN,omitempty"`
	SupplierAltPID          string                          `xml:"SUPPLIER_ALT_PID,omitempty"`
	BuyerPIDs               []*BuyerPID                     `xml:"BUYER_PID,omitempty"`
	ManufacturerPID         string                          `xml:"MANUFACTURER_PID,omitempty"`
	ManufacturerName        string                          `xml:"MANUFACTURER_NAME,omitempty"`
	ManufacturerTypeDescr   LocalizedStrings                `xml:"MANUFACTURER_TYPE_DESCR,omitempty"`
	ERPGroupBuyer           string                          `xml:"ERP_GROUP_BUYER,omitempty"`
	ERPGroupSupplier        string                          `xml:"ERP_GROUP_SUPPLIER,omitempty"`
	DeliveryTime            *int                            `xml:"DELIVERY_TIME,omitempty"`
	SpecialTreatmentClasses []*ProductSpecialTreatmentClass `xml:"SPECIAL_TREATMENT_CLASS,omitempty"`
	Keywords                LocalizedStrings                `xml:"KEYWORD,omitempty"`
	Remarks                 LocalizedStrings                `xml:"REMARKS,omitempty"`
	Segments                LocalizedStrings                `xml:"SEGMENT,omitempty"`
	ProductOrder            int                             `xml:"PRODUCT_ORDER,omitempty"`
	ProductStatus           []*ProductStatus                `xml:"PRODUCT_STATUS,omitempty"`
}

type BuyerPID struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type ProductFeatures struct {
	FeatureSystemName string           `xml:"REFERENCE_FEATURE_SYSTEM_NAME,omitempty"`
	FeatureGroupID    string           `xml:"REFERENCE_FEATURE_GROUP_ID,omitempty"`
	FeatureGroupName  LocalizedStrings `xml:"REFERENCE_FEATURE_GROUP_NAME,omitempty"`
	Features          []*Feature       `xml:"FEATURE,omitempty"`
}

func (pf ProductFeatures) IsEclass() bool {
	return strings.HasPrefix(strings.ToUpper(pf.FeatureSystemName), "ECLASS")
}

func (pf ProductFeatures) IsUnspsc() bool {
	return strings.HasPrefix(strings.ToUpper(pf.FeatureSystemName), "UNSPSC")
}

func (pf ProductFeatures) Version() string {
	parts := strings.SplitN(pf.FeatureSystemName, "-", 2)
	if len(parts) == 2 && utf8.RuneCountInString(parts[1]) > 0 {
		return parts[1]
	}
	return ""
}

type Feature struct {
	Name         LocalizedStrings   `xml:"FNAME"`
	Variants     []*FeatureVariants `xml:"VARIANTS,omitempty"`
	Values       LocalizedStrings   `xml:"FVALUE,omitempty"`
	Unit         string             `xml:"FUNIT,omitempty"`
	Order        int                `xml:"FORDER,omitempty"`
	Descr        LocalizedStrings   `xml:"FDESCR,omitempty"`
	ValueDetails LocalizedStrings   `xml:"FVALUE_DETAILS,omitempty"`
}

type FeatureVariants struct {
	Variants []*FeatureVariant `xml:"VARIANT"`
	Order    int               `xml:"VORDER,omitempty"`
}

type FeatureVariant struct {
	Value                 LocalizedStrings `xml:"FVALUE"`
	SupplierAIDSupplement string           `xml:"SUPPLIER_AID_SUPPLEMENT"`
}

type ProductOrderDetails struct {
	OrderUnit        string  `xml:"ORDER_UNIT"`
	ContentUnit      string  `xml:"CONTENT_UNIT,omitempty"`
	NoCuPerOu        float64 `xml:"NO_CU_PER_OU,omitempty"`
	PriceQuantity    float64 `xml:"PRICE_QUANTITY,omitempty"`
	QuantityMin      float64 `xml:"QUANTITY_MIN,omitempty"`
	QuantityInterval float64 `xml:"QUANTITY_INTERVAL,omitempty"`
	QuantityMax      float64 `xml:"QUANTITY_MAX,omitempty"`
}

const (
	DateTimeValidStartDate = "valid_start_date"
	DateTimeValidEndDate   = "valid_end_date"
)

type ProductPriceDetails struct {
	Dates            []*DateTime     `xml:"DATETIME,omitempty"`
	DailyPriceString string          `xml:"DAILY_PRICE,omitempty"`
	Prices           []*ProductPrice `xml:"PRODUCT_PRICE"`
}

func (ppd *ProductPriceDetails) ValidStartDate() time.Time {
	var date *DateTime

	for _, d := range ppd.Dates {
		if d.Type == DateTimeValidStartDate {
			date = d
			break
		}
	}

	if date == nil {
		return DefaultStartDate
	}

	time, err := date.Time()
	if err != nil {
		return DefaultStartDate
	}
	return time
}

func (ppd *ProductPriceDetails) ValidEndDate() time.Time {
	var date *DateTime

	for _, d := range ppd.Dates {
		if d.Type == DateTimeValidEndDate {
			date = d
			break
		}
	}

	if date == nil {
		return DefaultEndDate
	}

	time, err := date.Time()
	if err != nil {
		return DefaultEndDate
	}
	return time
}

func (ppd ProductPriceDetails) IsDailyPrice() bool {
	value := strings.ToUpper(ppd.DailyPriceString)
	return value == "TRUE" || value == "1" || value == "T"
}

const (
	ProductPriceTypeNetList        = "net_list"
	ProductPriceTypeGrosList       = "gros_list"
	ProductPriceTypeNetCustomer    = "net_customer"
	ProductPriceTypeNRP            = "nrp"
	ProductPriceTypeNetCustomerExp = "net_customer_exp"
)

type ProductPrice struct {
	Type       string        `xml:"price_type,attr,omitempty"`
	Amount     float64       `xml:"PRICE_AMOUNT"`
	Currency   string        `xml:"PRICE_CURRENCY,omitempty"`
	TaxDetails []*TaxDetails `xml:"TAX_DETAILS,omitempty"`
	Tax        *float64      `xml:"TAX,omitempty"`
	Factor     float64       `xml:"PRICE_FACTOR,omitempty"`
	LowerBound float64       `xml:"LOWER_BOUND,omitempty"`
	Territory  []string      `xml:"TERRITORY,omitempty"`
}

// TaxDetails models the BMEcat 2005 TAX_DETAILS element, which replaces the
// bare TAX element from 1.2 when richer tax information is required. A
// ProductPrice may carry either Tax or TaxDetails, mirroring the choice in
// the specification.
type TaxDetails struct {
	TaxCategory string   `xml:"TAX_CATEGORY,omitempty"`
	TaxType     string   `xml:"TAX_TYPE,omitempty"`
	Tax         *float64 `xml:"TAX,omitempty"`
}

const (
	ProductReferenceTypeSparepart     = "sparepart"
	ProductReferenceTypeSimilar       = "similar"
	ProductReferenceTypeFollowup      = "followup"
	ProductReferenceTypeMandatory     = "mandatory"
	ProductReferenceTypeSelect        = "select"
	ProductReferenceTypeDiffOrderUnit = "diff_orderunit"
	ProductReferenceTypeAccessories   = "accessories"
	ProductReferenceTypeConsistsOf    = "consists_of"
	ProductReferenceTypeBaseProduct   = "base_product"
	ProductReferenceTypeOthers        = "others"
)

type ProductReference struct {
	Type           string  `xml:"type,attr"`
	Quantity       float64 `xml:"quantity,attr,omitempty"`
	ProdIDTo       string  `xml:"PROD_ID_TO"`
	CatalogID      string  `xml:"CATALOG_ID,omitempty"`
	CatalogVersion string  `xml:"CATALOG_VERSION,omitempty"`
}

// ProductLogisticDetails models the BMEcat 2005 PRODUCT_LOGISTIC_DETAILS
// element. It has no 1.2 equivalent and is one of the more commonly used
// 2005 additions.
type ProductLogisticDetails struct {
	CustomsTariffNumbers []*CustomsTariffNumber `xml:"CUSTOMS_TARIFF_NUMBER,omitempty"`
	StatisticsFactor     float64                `xml:"STATISTICS_FACTOR,omitempty"`
	CountriesOfOrigin    []string               `xml:"COUNTRY_OF_ORIGIN,omitempty"`
	Dimensions           *ProductDimensions     `xml:"PRODUCT_DIMENSIONS,omitempty"`
}

type CustomsTariffNumber struct {
	CustomsNumber string   `xml:"CUSTOMS_NUMBER"`
	Territory     []string `xml:"TERRITORY,omitempty"`
}

type ProductDimensions struct {
	Volume float64 `xml:"VOLUME,omitempty"`
	Weight float64 `xml:"WEIGHT,omitempty"`
	Length float64 `xml:"LENGTH,omitempty"`
	Width  float64 `xml:"WIDTH,omitempty"`
	Depth  float64 `xml:"DEPTH,omitempty"`
}
