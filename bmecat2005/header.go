package bmecat2005

import (
	"encoding/xml"
	"time"
)

const (
	DateTimeGenerationDate = "generation_date"
)

type Header struct {
	XMLName xml.Name `xml:"HEADER"`

	GeneratorInfo string                 `xml:"GENERATOR_INFO,omitempty"`
	Catalog       *Catalog               `xml:"CATALOG"`
	Buyer         *Buyer                 `xml:"BUYER,omitempty"`
	Agreements    []*Agreement           `xml:"AGREEMENT,omitempty"`
	Supplier      *Supplier              `xml:"SUPPLIER,omitempty"`
	UDX           *UserDefinedExtensions `xml:"USER_DEFINED_EXTENSIONS,omitempty"`

	NumberOfProducts                  int `xml:"-"`
	NumberOfCatalogGroups             int `xml:"-"`
	NumberOfClassificationGroups      int `xml:"-"`
	NumberOfProductToCatalogGroupMaps int `xml:"-"`
}

type Catalog struct {
	XMLName xml.Name `xml:"CATALOG"`

	Language    string           `xml:"LANGUAGE"`
	ID          string           `xml:"CATALOG_ID"`
	Version     string           `xml:"CATALOG_VERSION"`
	Name        LocalizedStrings `xml:"CATALOG_NAME,omitempty"`
	GenDate     *DateTime        `xml:"DATETIME,omitempty"`
	Territories []string         `xml:"TERRITORY,omitempty"`
	Currency    string           `xml:"CURRENCY,omitempty"`
	MimeRoot    string           `xml:"MIME_ROOT,omitempty"`
	PriceFlags  []PriceFlag      `xml:"PRICE_FLAG,omitempty"`
}

const (
	PriceFlagInclFreight   = "incl_freight"
	PriceFlagInclPacking   = "incl_packing"
	PriceFlagInclAssurance = "incl_assurance"
	PriceFlagInclDuty      = "incl_duty"
)

type PriceFlag struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

var (
	CatalogIncludesFreight   = PriceFlag{Type: PriceFlagInclFreight, Value: "true"}
	CatalogIncludesPacking   = PriceFlag{Type: PriceFlagInclPacking, Value: "true"}
	CatalogIncludesAssurance = PriceFlag{Type: PriceFlagInclAssurance, Value: "true"}
	CatalogIncludesDuty      = PriceFlag{Type: PriceFlagInclDuty, Value: "true"}
)

type IDRef struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type Buyer struct {
	ID      *IDRef   `xml:"BUYER_ID,omitempty"`
	Name    string   `xml:"BUYER_NAME"`
	Address *Address `xml:"ADDRESS,omitempty"`
}

type Address struct {
	Type      string           `xml:"type,attr"`
	Name      LocalizedStrings `xml:"NAME,omitempty"`
	Name2     LocalizedStrings `xml:"NAME2,omitempty"`
	Name3     LocalizedStrings `xml:"NAME3,omitempty"`
	Contact   LocalizedStrings `xml:"CONTACT,omitempty"`
	Street    LocalizedStrings `xml:"STREET,omitempty"`
	Zip       LocalizedStrings `xml:"ZIP,omitempty"`
	BoxNo     LocalizedStrings `xml:"BOXNO,omitempty"`
	ZipBox    LocalizedStrings `xml:"ZIPBOX,omitempty"`
	City      LocalizedStrings `xml:"CITY,omitempty"`
	State     LocalizedStrings `xml:"STATE,omitempty"`
	Country   LocalizedStrings `xml:"COUNTRY,omitempty"`
	Phone     LocalizedStrings `xml:"PHONE,omitempty"`
	Fax       LocalizedStrings `xml:"FAX,omitempty"`
	Email     string           `xml:"EMAIL,omitempty"`
	PublicKey string           `xml:"PUBLIC_KEY,omitempty"`
	URL       string           `xml:"URL,omitempty"`
	Remarks   LocalizedStrings `xml:"ADDRESS_REMARKS,omitempty"`
}

const (
	DateTimeAgreementStartDate = "agreement_start_date"
	DateTimeAgreementEndDate   = "agreement_end_date"
)

type Agreement struct {
	Type    string      `xml:"type,attr,omitempty"`
	Default string      `xml:"default,attr,omitempty"`
	ID      string      `xml:"AGREEMENT_ID"`
	Dates   []*DateTime `xml:"DATETIME,omitempty"`
}

func (a *Agreement) StartDate() time.Time {
	var date *DateTime

	for _, d := range a.Dates {
		if d.Type == DateTimeAgreementStartDate {
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

func (a *Agreement) EndDate() time.Time {
	var date *DateTime

	for _, d := range a.Dates {
		if d.Type == DateTimeAgreementEndDate {
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

type Supplier struct {
	ID       *IDRef    `xml:"SUPPLIER_ID"`
	Name     string    `xml:"SUPPLIER_NAME"`
	Address  *Address  `xml:"ADDRESS,omitempty"`
	MimeInfo *MimeInfo `xml:"MIME_INFO,omitempty"`
}
