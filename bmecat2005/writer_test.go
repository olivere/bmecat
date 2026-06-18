package bmecat2005_test

import (
	"bytes"
	"context"
	"flag"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/olivere/bmecat/bmecat2005"
)

// update regenerates the golden files when set, e.g.
//
//	go test ./bmecat2005/ -run TestWrite -update
var update = flag.Bool("update", false, "update golden files")

// intp returns a pointer to the given int, for use in optional fields.
func intp(v int) *int { return &v }

// float64p returns a pointer to the given float64, for use in optional fields.
func float64p(v float64) *float64 { return &v }

var testHeader = &bmecat2005.Header{
	GeneratorInfo: "BMEcat Generator",
	Catalog: &bmecat2005.Catalog{
		Language:    "deu",
		ID:          "CAT1",
		Version:     "1.0",
		Name:        "Katalogbezeichnung",
		GenDate:     bmecat2005.NewDateTime(bmecat2005.DateTimeGenerationDate, time.Date(2000, 10, 24, 20, 38, 0o0, 0, time.UTC)),
		Territories: []string{"DE", "AT"},
		Currency:    "EUR",
		MimeRoot:    "https://example.com/images",
		PriceFlags: []bmecat2005.PriceFlag{
			bmecat2005.CatalogIncludesPacking,
		},
	},
	Buyer: &bmecat2005.Buyer{
		ID:   &bmecat2005.IDRef{Type: "buyer", Value: "BUYCO"},
		Name: "BuyCo Inc.",
	},
	Agreements: []*bmecat2005.Agreement{
		{
			ID: "23/97",
			Dates: []*bmecat2005.DateTime{
				bmecat2005.NewDateTime(bmecat2005.DateTimeAgreementStartDate, time.Date(1999, 3, 17, 0, 0, 0, 0, time.UTC)),
				bmecat2005.NewDateTime(bmecat2005.DateTimeAgreementStartDate, time.Date(2002, 5, 31, 0, 0, 0, 0, time.UTC)),
			},
		},
	},
	Supplier: &bmecat2005.Supplier{
		ID:   &bmecat2005.IDRef{Type: "supplier", Value: "SUPPLYCO"},
		Name: "SupplyCo Ltd.",
		Address: &bmecat2005.Address{
			Type: "supplier",
			City: "London",
		},
		MimeInfo: &bmecat2005.MimeInfo{
			Mimes: []*bmecat2005.Mime{
				{Type: "image/jpeg", Source: "supplier_logo.jpg", Purpose: "logo"},
			},
		},
	},
	UDX: &bmecat2005.UserDefinedExtensions{
		Fields: []*bmecat2005.UserDefinedExtensionField{
			{
				Name:  "SYSTEM.CUSTOM_FIELD1",
				Value: "A",
			},
			{
				Name:  "SYSTEM.CUSTOM_FIELD3",
				Value: "C",
			},
			{
				Name:  "WALLMEDIEN.PROPERTIES",
				Raw:   true,
				Value: `<UDX.WALLMEDIEN.PROPERTY><UDX.WALLMEDIEN.PROPERTY.NAME>EXTCONFIGFORM</UDX.WALLMEDIEN.PROPERTY.NAME><UDX.WALLMEDIEN.PROPERTY.VALUE>ADV_Relevanz</UDX.WALLMEDIEN.PROPERTY.VALUE></UDX.WALLMEDIEN.PROPERTY>`,
			},
		},
	},
}

type catalogWriter struct {
	tx                   bmecat2005.Transaction
	language             string
	prevVersion          int
	header               *bmecat2005.Header
	classificationSystem *bmecat2005.ClassificationSystem
	products             []*bmecat2005.Product
}

func (w catalogWriter) Transaction() bmecat2005.Transaction {
	return w.tx
}

func (w catalogWriter) Language() string {
	return w.language
}

func (w catalogWriter) PreviousVersion() int {
	return w.prevVersion
}

func (w catalogWriter) Header() *bmecat2005.Header {
	return w.header
}

func (w catalogWriter) ClassificationSystem() *bmecat2005.ClassificationSystem {
	return w.classificationSystem
}

func (w catalogWriter) Products(ctx context.Context) (<-chan *bmecat2005.Product, <-chan error) {
	if len(w.products) == 0 {
		return nil, nil
	}
	outCh := make(chan *bmecat2005.Product)
	errCh := make(chan error, 1)
	go func() {
		defer close(outCh)
		defer close(errCh)
		for _, p := range w.products {
			outCh <- p
			select {
			default:
			case <-ctx.Done():
				return
			}
		}
	}()
	return outCh, errCh
}

// sampleClassificationSystem returns a classification system used by several
// writer tests.
func sampleClassificationSystem() *bmecat2005.ClassificationSystem {
	return &bmecat2005.ClassificationSystem{
		Name:     "udf_Supplier-1.0",
		FullName: testHeader.Supplier.Name,
		Groups: []*bmecat2005.ClassificationGroup{
			{ID: "1", Name: "Hardware", Type: "node"},
			{ID: "2", Name: "Notebook", ParentID: "1", Type: "node"},
			{ID: "3", Name: "Desktop", ParentID: "1", Type: "node"},
			{ID: "4", Name: "PC", ParentID: "2", Type: "leaf"},
			{ID: "5", Name: "Mac", ParentID: "2", Type: "leaf"},
		},
	}
}

// sampleProduct returns a fully populated product exercising the renamed
// 2005 elements as well as a few 2005-only additions (INTERNATIONAL_PID with
// a type attribute and PRODUCT_LOGISTIC_DETAILS).
func sampleProduct() *bmecat2005.Product {
	return &bmecat2005.Product{
		SupplierPID: "1000",
		Details: &bmecat2005.ProductDetails{
			DescriptionShort: `Apple MacBook Pro 13"`,
			DescriptionLong:  `Das Kraftpaket unter den Notebooks.`,
			InternationalPIDs: []*bmecat2005.InternationalPID{
				{Type: "gtin", Value: "8712670911213"},
			},
			SupplierAltPID: "ALT-1000",
			BuyerPIDs: []*bmecat2005.BuyerPID{
				{Type: "KMF", Value: "78787"},
			},
			ManufacturerPID:  "MPN",
			ManufacturerName: "Microsoft",
			DeliveryTime:     intp(2),
			SpecialTreatmentClasses: []*bmecat2005.ProductSpecialTreatmentClass{
				{Type: "GGVS", Value: "1201"},
			},
			Keywords: []string{"Notebook", "Hardware"},
			Remarks:  "Noch heute bestellen!",
			ProductStatus: []*bmecat2005.ProductStatus{
				{Type: bmecat2005.ProductStatusCoreProduct, Value: "Kernsortiment"},
			},
		},
		Features: []*bmecat2005.ProductFeatures{
			{
				FeatureSystemName: "ECLASS-5.1",
				FeatureGroupID:    "19010203",
				Features: []*bmecat2005.Feature{
					{
						Name:   "Netzspannung",
						Values: []string{"110", "220"},
						Unit:   "VLT",
					},
				},
			},
			{
				FeatureSystemName: "udf_Supplier-1.0",
				FeatureGroupID:    "5",
			},
		},
		OrderDetails: &bmecat2005.ProductOrderDetails{
			OrderUnit:     "BOX",
			NoCuPerOu:     6.0,
			ContentUnit:   "PCE",
			PriceQuantity: 1,
			QuantityMin:   1,
		},
		PriceDetails: []*bmecat2005.ProductPriceDetails{
			{
				Dates: []*bmecat2005.DateTime{
					bmecat2005.NewDateTime(bmecat2005.DateTimeValidStartDate, time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)),
					bmecat2005.NewDateTime(bmecat2005.DateTimeValidEndDate, time.Date(2001, 7, 31, 0, 0, 0, 0, time.UTC)),
				},
				Prices: []*bmecat2005.ProductPrice{
					{
						Type:       bmecat2005.ProductPriceTypeNetCustomer,
						Amount:     1499.50,
						Currency:   "EUR",
						Tax:        float64p(0.19),
						Factor:     1.0,
						LowerBound: 1,
						Territory:  []string{"DE", "AT"},
					},
					{
						Type:       bmecat2005.ProductPriceTypeNetCustomer,
						Amount:     1300.90,
						Currency:   "EUR",
						Tax:        float64p(0.19),
						Factor:     1.0,
						LowerBound: 100,
						Territory:  []string{"DE", "AT"},
					},
				},
			},
		},
		MimeInfo: &bmecat2005.MimeInfo{
			Mimes: []*bmecat2005.Mime{
				{
					Type:    "image/jpeg",
					Source:  "55-K-31.jpg",
					Descr:   "Frontansicht des Notebooks",
					Purpose: bmecat2005.MimePurposeNormal,
				},
			},
		},
		References: []*bmecat2005.ProductReference{
			{
				Type:     bmecat2005.ProductReferenceTypeSimilar,
				ProdIDTo: "2000",
			},
		},
		LogisticDetails: &bmecat2005.ProductLogisticDetails{
			CustomsTariffNumbers: []*bmecat2005.CustomsTariffNumber{
				{CustomsNumber: "84713000"},
			},
			CountriesOfOrigin: []string{"CN"},
			Dimensions: &bmecat2005.ProductDimensions{
				Weight: 1.37,
			},
		},
	}
}

// assertGolden compares have against the golden file at path, regenerating it
// when -update is set.
func assertGolden(t *testing.T, path, have string) {
	t.Helper()
	if *update {
		if err := os.WriteFile(path, []byte(have+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(data))
	if want != have {
		diffStrings(t, want, have)
	}
}

func TestWriteNewCatalog(t *testing.T) {
	cw := catalogWriter{
		tx:                   bmecat2005.NewCatalog,
		language:             "de",
		prevVersion:          0,
		header:               testHeader,
		classificationSystem: sampleClassificationSystem(),
		products:             []*bmecat2005.Product{sampleProduct()},
	}

	var buf bytes.Buffer
	w := bmecat2005.NewWriter(&buf, bmecat2005.WithIndent("  "))
	if err := w.Do(context.Background(), cw); err != nil {
		t.Fatal(err)
	}

	assertGolden(t, "testdata/new_catalog.golden.xml", strings.TrimSpace(buf.String()))
}

func TestWriteNewCatalogWithBlankClassificationSystem(t *testing.T) {
	classSys := &bmecat2005.ClassificationSystem{
		Name:     "udf_Supplier-1.0",
		FullName: testHeader.Supplier.Name,
		Groups:   []*bmecat2005.ClassificationGroup{}, // no groups
	}
	cw := catalogWriter{
		tx:                   bmecat2005.NewCatalog,
		language:             "de",
		prevVersion:          0,
		header:               testHeader,
		classificationSystem: classSys,
		products:             []*bmecat2005.Product{sampleProduct()},
	}

	var buf bytes.Buffer
	w := bmecat2005.NewWriter(&buf, bmecat2005.WithIndent("  "))
	if err := w.Do(context.Background(), cw); err != nil {
		t.Fatal(err)
	}

	assertGolden(t, "testdata/new_catalog_with_blank_classification_system.golden.xml", strings.TrimSpace(buf.String()))
}
