package bmecat12

import (
	"encoding/xml"
	"strings"
	"time"
	"unicode/utf8"
)

// Article represents a product according to the BMEcat 1.2 specification.
type Article struct {
	XMLName xml.Name `xml:"ARTICLE"`

	Mode         string                 `xml:"mode,attr,omitempty"`
	SupplierAID  string                 `xml:"SUPPLIER_AID"`
	Details      *ArticleDetails        `xml:"ARTICLE_DETAILS"`
	Features     []*ArticleFeatures     `xml:"ARTICLE_FEATURES,omitempty"`
	OrderDetails *ArticleOrderDetails   `xml:"ARTICLE_ORDER_DETAILS"`
	PriceDetails []*ArticlePriceDetails `xml:"ARTICLE_PRICE_DETAILS"`
	MimeInfo     *MimeInfo              `xml:"MIME_INFO,omitempty"`
	UDX          *UserDefinedExtensions `xml:"USER_DEFINED_EXTENSIONS,omitempty"`
	References   []*ArticleReference    `xml:"ARTICLE_REFERENCE,omitempty"`

	// CatalogGroupIDs is the list of CATALOG_STRUCTURE IDs gathered on the 1st pass of the parser.
	CatalogGroupIDs []string `xml:"-"`
}

const (
	ArticleStatusBargain     = "bargain"
	ArticleStatusNewArticle  = "new_article"
	ArticleStatusOldArticle  = "old_article"
	ArticleStatusNew         = "new"
	ArticleStatusUsed        = "used"
	ArticleStatusRefurbished = "refurbished"
	ArticleStatusCoreArticle = "core_article"
	ArticleStatusOthers      = "others"
)

type ArticleStatus struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type ArticleSpecialTreatmentClass struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type ArticleDetails struct {
	DescriptionShort        string                          `xml:"DESCRIPTION_SHORT"`
	DescriptionLong         string                          `xml:"DESCRIPTION_LONG,omitempty"`
	EAN                     string                          `xml:"EAN,omitempty"`
	SupplierAltAID          string                          `xml:"SUPPLIER_ALT_AID,omitempty"`
	BuyerAIDs               []*BuyerAID                     `xml:"BUYER_AID,omitempty"`
	ManufacturerAID         string                          `xml:"MANUFACTURER_AID,omitempty"`
	ManufacturerName        string                          `xml:"MANUFACTURER_NAME,omitempty"`
	ManufacturerTypeDescr   string                          `xml:"MANUFACTURER_TYPE_DESCR,omitempty"`
	ERPGroupBuyer           string                          `xml:"ERP_GROUP_BUYER,omitempty"`
	ERPGroupSupplier        string                          `xml:"ERP_GROUP_SUPPLIER,omitempty"`
	DeliveryTime            float32                         `xml:"DELIVERY_TIME,omitempty"`
	SpecialTreatmentClasses []*ArticleSpecialTreatmentClass `xml:"SPECIAL_TREATMENT_CLASS,omitempty"`
	Keywords                []string                        `xml:"KEYWORD,omitempty"`
	Remarks                 string                          `xml:"REMARKS,omitempty"`
	Segments                []string                        `xml:"SEGMENT,omitempty"`
	ArticleOrder            int                             `xml:"ARTICLE_ORDER,omitempty"`
	ArticleStatus           []*ArticleStatus                `xml:"ARTICLE_STATUS,omitempty"`
}

type BuyerAID struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type ArticleFeatures struct {
	FeatureSystemName string     `xml:"REFERENCE_FEATURE_SYSTEM_NAME,omitempty"`
	FeatureGroupID    string     `xml:"REFERENCE_FEATURE_GROUP_ID,omitempty"`
	FeatureGroupName  string     `xml:"REFERENCE_FEATURE_GROUP_NAME,omitempty"`
	Features          []*Feature `xml:"FEATURE,omitempty"`
}

func (af ArticleFeatures) IsEclass() bool {
	return strings.HasPrefix(strings.ToUpper(af.FeatureSystemName), "ECLASS")
}

func (af ArticleFeatures) IsUnspsc() bool {
	return strings.HasPrefix(strings.ToUpper(af.FeatureSystemName), "UNSPSC")
}

func (af ArticleFeatures) Version() string {
	parts := strings.SplitN(af.FeatureSystemName, "-", 2)
	if len(parts) == 2 && utf8.RuneCountInString(parts[1]) > 0 {
		return parts[1]
	}
	return ""
}

type Feature struct {
	Name         string             `xml:"FNAME"`
	Variants     []*FeatureVariants `xml:"VARIANTS,omitempty"`
	Values       []string           `xml:"FVALUE,omitempty"`
	Unit         string             `xml:"FUNIT,omitempty"`
	Order        int                `xml:"FORDER,omitempty"`
	Descr        string             `xml:"FDESCR,omitempty"`
	ValueDetails string             `xml:"FVALUE_DETAILS,omitempty"`
}

type FeatureVariants struct {
	Variants []*FeatureVariant `xml:"VARIANT"`
	Order    int               `xml:"VORDER,omitempty"`
}

type FeatureVariant struct {
	Value                 string `xml:"FVALUE"`
	SupplierAIDSupplement string `xml:"SUPPLIER_AID_SUPPLEMENT"`
}

type ArticleOrderDetails struct {
	OrderUnit        string  `xml:"ORDER_UNIT"`
	ContentUnit      string  `xml:"CONTENT_UNIT,omitempty"`
	NoCuPerOu        float64 `xml:"NO_CU_PER_OU,omitempty"`
	PriceQuantity    float64 `xml:"PRICE_QUANTITY,omitempty"`
	QuantityMin      float64 `xml:"QUANTITY_MIN,omitempty"`
	QuantityInterval float64 `xml:"QUANTITY_INTERVAL,omitempty"`
}

const (
	DateTimeValidStartDate = "valid_start_date"
	DateTimeValidEndDate   = "valid_end_date"
)

type ArticlePriceDetails struct {
	Dates            []*DateTime     `xml:"DATETIME,omitempty"`
	DailyPriceString string          `xml:"DAILY_PRICE,omitempty"`
	Prices           []*ArticlePrice `xml:"ARTICLE_PRICE"`
}

func (apd *ArticlePriceDetails) ValidStartDate() time.Time {
	var date *DateTime

	for _, d := range apd.Dates {
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

func (apd *ArticlePriceDetails) ValidEndDate() time.Time {
	var date *DateTime

	for _, d := range apd.Dates {
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

func (apd ArticlePriceDetails) IsDailyPrice() bool {
	value := strings.ToUpper(apd.DailyPriceString)
	return value == "TRUE" || value == "1" || value == "T"
}

const (
	ArticlePriceTypeNetList        = "net_list"
	ArticlePriceTypeGrosList       = "gros_list"
	ArticlePriceTypeNetCustomer    = "net_customer"
	ArticlePriceTypeNRP            = "nrp"
	ArticlePriceTypeNetCustomerExp = "net_customer_exp"
)

type ArticlePrice struct {
	Type       string   `xml:"price_type,attr,omitempty"`
	Amount     float64  `xml:"PRICE_AMOUNT"`
	Currency   string   `xml:"PRICE_CURRENCY,omitempty"`
	Tax        float64  `xml:"TAX,omitempty"`
	Factor     float64  `xml:"PRICE_FACTOR,omitempty"`
	LowerBound float64  `xml:"LOWER_BOUND,omitempty"`
	Territory  []string `xml:"TERRITORY,omitempty"`
}

const (
	ArticleReferenceTypeSparepart     = "sparepart"
	ArticleReferenceTypeSimilar       = "similar"
	ArticleReferenceTypeFollowup      = "followup"
	ArticleReferenceTypeMandatory     = "mandatory"
	ArticleReferenceTypeSelect        = "select"
	ArticleReferenceTypeDiffOrderUnit = "diff_orderunit"
	ArticleReferenceTypeAccessories   = "accessories"
	ArticleReferenceTypeConsistsOf    = "consists_of"
	ArticleReferenceTypeOthers        = "others"
)

type ArticleReference struct {
	Type           string  `xml:"type,attr"`
	Quantity       float64 `xml:"quantity,attr,omitempty"`
	ArtIDTo        string  `xml:"ART_ID_TO"`
	CatalogID      string  `xml:"CATALOG_ID,omitempty"`
	CatalogVersion string  `xml:"CATALOG_VERSION,omitempty"`
}
