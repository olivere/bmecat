package bmecat12_test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/olivere/bmecat/bmecat12"
)

var (
	testHeader = &bmecat12.Header{
		GeneratorInfo: "BMEcat Generator",
		Catalog: &bmecat12.Catalog{
			Language:    "deu",
			ID:          "CAT1",
			Version:     "1.0",
			Name:        "Katalogbezeichnung",
			GenDate:     bmecat12.NewDateTime(bmecat12.DateTimeGenerationDate, time.Date(2000, 10, 24, 20, 38, 00, 0, time.UTC)),
			Territories: []string{"DE", "AT"},
			Currency:    "EUR",
			MimeRoot:    "https://example.com/images",
			PriceFlags: []bmecat12.PriceFlag{
				bmecat12.CatalogIncludesPacking,
			},
		},
		Buyer: &bmecat12.Buyer{
			ID:   &bmecat12.IDRef{Type: "buyer", Value: "BUYCO"},
			Name: "BuyCo Inc.",
		},
		Agreements: []*bmecat12.Agreement{
			&bmecat12.Agreement{
				ID: "23/97",
				Dates: []*bmecat12.DateTime{
					bmecat12.NewDateTime(bmecat12.DateTimeAgreementStartDate, time.Date(1999, 3, 17, 0, 0, 0, 0, time.UTC)),
					bmecat12.NewDateTime(bmecat12.DateTimeAgreementStartDate, time.Date(2002, 5, 31, 0, 0, 0, 0, time.UTC)),
				},
			},
		},
		Supplier: &bmecat12.Supplier{
			ID:   &bmecat12.IDRef{Type: "supplier", Value: "SUPPLYCO"},
			Name: "SupplyCo Ltd.",
			Address: &bmecat12.Address{
				Type: "supplier",
				City: "London",
			},
			MimeInfo: &bmecat12.MimeInfo{
				Mimes: []*bmecat12.Mime{
					&bmecat12.Mime{Type: "image/jpeg", Source: "supplier_logo.jpg", Purpose: "logo"},
				},
			},
		},
		UDX: &bmecat12.UserDefinedExtensions{
			Fields: []*bmecat12.UserDefinedExtensionField{
				&bmecat12.UserDefinedExtensionField{
					Name:  "SYSTEM.CUSTOM_FIELD1",
					Value: "A",
				},
				&bmecat12.UserDefinedExtensionField{
					Name:  "SYSTEM.CUSTOM_FIELD3",
					Value: "C",
				},
				&bmecat12.UserDefinedExtensionField{
					Name:  "WALLMEDIEN.PROPERTIES",
					Raw:   true,
					Value: `<UDX.WALLMEDIEN.PROPERTY><UDX.WALLMEDIEN.PROPERTY.NAME>EXTCONFIGFORM</UDX.WALLMEDIEN.PROPERTY.NAME><UDX.WALLMEDIEN.PROPERTY.VALUE>ADV_Relevanz</UDX.WALLMEDIEN.PROPERTY.VALUE></UDX.WALLMEDIEN.PROPERTY>`,
				},
			},
		},
	}
)

type catalogWriter struct {
	tx                   bmecat12.Transaction
	language             string
	prevVersion          int
	header               *bmecat12.Header
	classificationSystem *bmecat12.ClassificationSystem
	articles             []*bmecat12.Article
}

func (w catalogWriter) Transaction() bmecat12.Transaction {
	return w.tx
}

func (w catalogWriter) Language() string {
	return w.language
}

func (w catalogWriter) PreviousVersion() int {
	return w.prevVersion
}

func (w catalogWriter) Header() *bmecat12.Header {
	return w.header
}

func (w catalogWriter) ClassificationSystem() *bmecat12.ClassificationSystem {
	return w.classificationSystem
}

func (w catalogWriter) Articles(ctx context.Context) (<-chan *bmecat12.Article, <-chan error) {
	if len(w.articles) == 0 {
		return nil, nil
	}
	outCh := make(chan *bmecat12.Article)
	errCh := make(chan error, 1)
	go func() {
		defer close(outCh)
		defer close(errCh)
		for _, a := range w.articles {
			outCh <- a
			select {
			default:
			case <-ctx.Done():
				return
			}
		}
	}()
	return outCh, errCh
}

func TestWriteNewCatalog(t *testing.T) {
	classSys := &bmecat12.ClassificationSystem{
		Name:     "udf_Supplier-1.0",
		FullName: testHeader.Supplier.Name,
		Groups: []*bmecat12.ClassificationGroup{
			{
				ID:   "1",
				Name: "Hardware",
				Type: "node",
			},
			{
				ID:       "2",
				Name:     "Notebook",
				ParentID: "1",
				Type:     "node",
			},
			{
				ID:       "3",
				Name:     "Desktop",
				ParentID: "1",
				Type:     "node",
			},
			{
				ID:       "4",
				Name:     "PC",
				ParentID: "2",
				Type:     "leaf",
			},
			{
				ID:       "5",
				Name:     "Mac",
				ParentID: "2",
				Type:     "leaf",
			},
		},
	}
	articles := []*bmecat12.Article{
		&bmecat12.Article{
			SupplierAID: "1000",
			Details: &bmecat12.ArticleDetails{
				DescriptionShort: `Apple MacBook Pro 13"`,
				DescriptionLong:  `Das Kraftpaket unter den Notebooks.`,
				EAN:              "8712670911213",
				SupplierAltAID:   "ALT-1000",
				BuyerAIDs: []*bmecat12.BuyerAID{
					&bmecat12.BuyerAID{Type: "KMF", Value: "78787"},
				},
				ManufacturerAID:  "MPN",
				ManufacturerName: "Microsoft",
				DeliveryTime:     1.5,
				SpecialTreatmentClasses: []*bmecat12.ArticleSpecialTreatmentClass{
					&bmecat12.ArticleSpecialTreatmentClass{
						Type:  "GGVS",
						Value: "1201",
					},
				},
				Keywords: []string{"Notebook", "Hardware"},
				Remarks:  "Noch heute bestellen!",
				ArticleStatus: []*bmecat12.ArticleStatus{
					&bmecat12.ArticleStatus{Type: bmecat12.ArticleStatusCoreArticle, Value: "Kernsortiment"},
				},
			},
			Features: []*bmecat12.ArticleFeatures{
				&bmecat12.ArticleFeatures{
					FeatureSystemName: "ECLASS-5.1",
					FeatureGroupID:    "19010203",
					Features: []*bmecat12.Feature{
						&bmecat12.Feature{
							Name:   "Netzspannung",
							Values: []string{"110", "220"},
							Unit:   "VLT",
						},
					},
				},
				&bmecat12.ArticleFeatures{
					FeatureSystemName: "udf_Supplier-1.0",
					FeatureGroupID:    "5",
				},
			},
			OrderDetails: &bmecat12.ArticleOrderDetails{
				OrderUnit:     "BOX",
				NoCuPerOu:     6.0,
				ContentUnit:   "PCE",
				PriceQuantity: 1,
				QuantityMin:   1,
			},
			PriceDetails: []*bmecat12.ArticlePriceDetails{
				&bmecat12.ArticlePriceDetails{
					Dates: []*bmecat12.DateTime{
						bmecat12.NewDateTime(bmecat12.DateTimeValidStartDate, time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)),
						bmecat12.NewDateTime(bmecat12.DateTimeValidEndDate, time.Date(2001, 7, 31, 0, 0, 0, 0, time.UTC)),
					},
					Prices: []*bmecat12.ArticlePrice{
						&bmecat12.ArticlePrice{
							Type:       bmecat12.ArticlePriceTypeNetCustomer,
							Amount:     1499.50,
							Currency:   "EUR",
							Tax:        0.19,
							Factor:     1.0,
							LowerBound: 1,
							Territory:  []string{"DE", "AT"},
						},
						&bmecat12.ArticlePrice{
							Type:       bmecat12.ArticlePriceTypeNetCustomer,
							Amount:     1300.90,
							Currency:   "EUR",
							Tax:        0.19,
							Factor:     1.0,
							LowerBound: 100,
							Territory:  []string{"DE", "AT"},
						},
					},
				},
			},
			// MIME_INFO
			MimeInfo: &bmecat12.MimeInfo{
				Mimes: []*bmecat12.Mime{
					&bmecat12.Mime{
						Type:    "image/jpeg",
						Source:  "55-K-31.jpg",
						Descr:   "Frontansicht des Notebooks",
						Purpose: bmecat12.MimePurposeNormal,
					},
				},
			},
			// USER_DEFINED_EXTENSIONS
			// ARTICLE_REFERENCE
			References: []*bmecat12.ArticleReference{
				&bmecat12.ArticleReference{
					Type:    bmecat12.ArticleReferenceTypeSimilar,
					ArtIDTo: "2000",
				},
			},
		},
	}
	cw := catalogWriter{
		tx:                   bmecat12.NewCatalog,
		language:             "de",
		prevVersion:          0,
		header:               testHeader,
		classificationSystem: classSys,
		articles:             articles,
	}

	var buf bytes.Buffer
	w := bmecat12.NewWriter(&buf, bmecat12.WithIndent("  "))
	ctx := context.Background()
	if err := w.Do(ctx, cw); err != nil {
		t.Fatal(err)
	}

	have := strings.TrimSpace(buf.String())
	data, err := ioutil.ReadFile("testdata/new_catalog.golden.xml")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(data))
	if want != have {
		// fmt.Println(have)
		diffStrings(t, want, have)
		t.Fail()
	}
}

func TestWriteUpdateProducts(t *testing.T) {
	classSys := &bmecat12.ClassificationSystem{
		Name:     "udf_Supplier-1.0",
		FullName: testHeader.Supplier.Name,
		Groups: []*bmecat12.ClassificationGroup{
			{
				ID:   "1",
				Name: "Hardware",
				Type: "node",
			},
			{
				ID:       "2",
				Name:     "Notebook",
				ParentID: "1",
				Type:     "node",
			},
			{
				ID:       "3",
				Name:     "Desktop",
				ParentID: "1",
				Type:     "node",
			},
			{
				ID:       "4",
				Name:     "PC",
				ParentID: "2",
				Type:     "leaf",
			},
			{
				ID:       "5",
				Name:     "Mac",
				ParentID: "2",
				Type:     "leaf",
			},
		},
	}
	articles := []*bmecat12.Article{
		&bmecat12.Article{
			Mode:        "update",
			SupplierAID: "1000",
			Details: &bmecat12.ArticleDetails{
				DescriptionShort: `Apple MacBook Pro 13"`,
				DescriptionLong:  `Das Kraftpaket unter den Notebooks.`,
				EAN:              "8712670911213",
				SupplierAltAID:   "ALT-1000",
				BuyerAIDs: []*bmecat12.BuyerAID{
					&bmecat12.BuyerAID{Type: "KMF", Value: "78787"},
				},
				ManufacturerAID:  "MPN",
				ManufacturerName: "Microsoft",
				DeliveryTime:     1.5,
				SpecialTreatmentClasses: []*bmecat12.ArticleSpecialTreatmentClass{
					&bmecat12.ArticleSpecialTreatmentClass{
						Type:  "GGVS",
						Value: "1201",
					},
				},
				Keywords: []string{"Notebook", "Hardware"},
				Remarks:  "Noch heute bestellen!",
				ArticleStatus: []*bmecat12.ArticleStatus{
					&bmecat12.ArticleStatus{Type: bmecat12.ArticleStatusCoreArticle, Value: "Kernsortiment"},
				},
			},
			Features: []*bmecat12.ArticleFeatures{
				&bmecat12.ArticleFeatures{
					FeatureSystemName: "ECLASS-5.1",
					FeatureGroupID:    "19010203",
					Features: []*bmecat12.Feature{
						&bmecat12.Feature{
							Name:   "Netzspannung",
							Values: []string{"110", "220"},
							Unit:   "VLT",
						},
					},
				},
				&bmecat12.ArticleFeatures{
					FeatureSystemName: "udf_Supplier-1.0",
					FeatureGroupID:    "5",
				},
			},
			OrderDetails: &bmecat12.ArticleOrderDetails{
				OrderUnit:     "BOX",
				NoCuPerOu:     6.0,
				ContentUnit:   "PCE",
				PriceQuantity: 1,
				QuantityMin:   1,
			},
			PriceDetails: []*bmecat12.ArticlePriceDetails{
				&bmecat12.ArticlePriceDetails{
					Dates: []*bmecat12.DateTime{
						bmecat12.NewDateTime(bmecat12.DateTimeValidStartDate, time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)),
						bmecat12.NewDateTime(bmecat12.DateTimeValidEndDate, time.Date(2001, 7, 31, 0, 0, 0, 0, time.UTC)),
					},
					Prices: []*bmecat12.ArticlePrice{
						&bmecat12.ArticlePrice{
							Type:       bmecat12.ArticlePriceTypeNetCustomer,
							Amount:     1499.50,
							Currency:   "EUR",
							Tax:        0.19,
							Factor:     1.0,
							LowerBound: 1,
							Territory:  []string{"DE", "AT"},
						},
						&bmecat12.ArticlePrice{
							Type:       bmecat12.ArticlePriceTypeNetCustomer,
							Amount:     1300.90,
							Currency:   "EUR",
							Tax:        0.19,
							Factor:     1.0,
							LowerBound: 100,
							Territory:  []string{"DE", "AT"},
						},
					},
				},
			},
			// MIME_INFO
			MimeInfo: &bmecat12.MimeInfo{
				Mimes: []*bmecat12.Mime{
					&bmecat12.Mime{
						Type:    "image/jpeg",
						Source:  "55-K-31.jpg",
						Descr:   "Frontansicht des Notebooks",
						Purpose: bmecat12.MimePurposeNormal,
					},
				},
			},
			// USER_DEFINED_EXTENSIONS
			// ARTICLE_REFERENCE
			References: []*bmecat12.ArticleReference{
				&bmecat12.ArticleReference{
					Type:    bmecat12.ArticleReferenceTypeSimilar,
					ArtIDTo: "2000",
				},
			},
		},
		&bmecat12.Article{
			Mode:        "delete",
			SupplierAID: "2000",
		},
	}
	cw := catalogWriter{
		tx:                   bmecat12.UpdateProducts,
		language:             "de",
		prevVersion:          13,
		header:               testHeader,
		classificationSystem: classSys,
		articles:             articles,
	}

	var buf bytes.Buffer
	w := bmecat12.NewWriter(&buf, bmecat12.WithIndent("  "))

	ctx := context.Background()
	if err := w.Do(ctx, cw); err != nil {
		t.Fatal(err)
	}

	have := strings.TrimSpace(buf.String())
	data, err := ioutil.ReadFile("testdata/update_products.golden.xml")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(data))
	if want != have {
		fmt.Println(have)
		// diffStrings(t, want, have)
		t.Fail()
	}
}

func TestWriteUpdatePrices(t *testing.T) {
	articles := []*bmecat12.Article{
		{
			SupplierAID: "1000",
			PriceDetails: []*bmecat12.ArticlePriceDetails{
				&bmecat12.ArticlePriceDetails{
					Dates: []*bmecat12.DateTime{
						bmecat12.NewDateTime(bmecat12.DateTimeValidStartDate, time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)),
						bmecat12.NewDateTime(bmecat12.DateTimeValidEndDate, time.Date(2001, 7, 31, 0, 0, 0, 0, time.UTC)),
					},
					Prices: []*bmecat12.ArticlePrice{
						&bmecat12.ArticlePrice{
							Type:       bmecat12.ArticlePriceTypeNetCustomer,
							Amount:     1499.50,
							Currency:   "EUR",
							Tax:        0.19,
							Factor:     1.0,
							LowerBound: 1,
							Territory:  []string{"DE", "AT"},
						},
						&bmecat12.ArticlePrice{
							Type:       bmecat12.ArticlePriceTypeNetCustomer,
							Amount:     1300.90,
							Currency:   "EUR",
							Tax:        0.19,
							Factor:     1.0,
							LowerBound: 100,
							Territory:  []string{"DE", "AT"},
						},
					},
				},
			},
		},
	}
	cw := catalogWriter{
		tx:                   bmecat12.UpdatePrices,
		language:             "de",
		prevVersion:          42,
		header:               testHeader,
		classificationSystem: nil,
		articles:             articles,
	}

	var buf bytes.Buffer
	w := bmecat12.NewWriter(&buf, bmecat12.WithIndent("  "))

	ctx := context.Background()
	if err := w.Do(ctx, cw); err != nil {
		t.Fatal(err)
	}

	have := strings.TrimSpace(buf.String())
	data, err := ioutil.ReadFile("testdata/update_prices.golden.xml")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(data))
	if want != have {
		// fmt.Println(have)
		diffStrings(t, want, have)
		t.Fail()
	}
}

func TestWriteNewCatalogWithBlankClassificationSystem(t *testing.T) {
	classSys := &bmecat12.ClassificationSystem{
		Name:     "udf_Supplier-1.0",
		FullName: testHeader.Supplier.Name,
		Groups:   []*bmecat12.ClassificationGroup{}, // no groups
	}
	articles := []*bmecat12.Article{
		&bmecat12.Article{
			SupplierAID: "1000",
			Details: &bmecat12.ArticleDetails{
				DescriptionShort: `Apple MacBook Pro 13"`,
				DescriptionLong:  `Das Kraftpaket unter den Notebooks.`,
				EAN:              "8712670911213",
				SupplierAltAID:   "ALT-1000",
				BuyerAIDs: []*bmecat12.BuyerAID{
					&bmecat12.BuyerAID{Type: "KMF", Value: "78787"},
				},
				ManufacturerAID:  "MPN",
				ManufacturerName: "Microsoft",
				DeliveryTime:     1.5,
				SpecialTreatmentClasses: []*bmecat12.ArticleSpecialTreatmentClass{
					&bmecat12.ArticleSpecialTreatmentClass{
						Type:  "GGVS",
						Value: "1201",
					},
				},
				Keywords: []string{"Notebook", "Hardware"},
				Remarks:  "Noch heute bestellen!",
				ArticleStatus: []*bmecat12.ArticleStatus{
					&bmecat12.ArticleStatus{Type: bmecat12.ArticleStatusCoreArticle, Value: "Kernsortiment"},
				},
			},
			Features: []*bmecat12.ArticleFeatures{
				&bmecat12.ArticleFeatures{
					FeatureSystemName: "ECLASS-5.1",
					FeatureGroupID:    "19010203",
					Features: []*bmecat12.Feature{
						&bmecat12.Feature{
							Name:   "Netzspannung",
							Values: []string{"110", "220"},
							Unit:   "VLT",
						},
					},
				},
			},
			OrderDetails: &bmecat12.ArticleOrderDetails{
				OrderUnit:     "BOX",
				NoCuPerOu:     6.0,
				ContentUnit:   "PCE",
				PriceQuantity: 1,
				QuantityMin:   1,
			},
			PriceDetails: []*bmecat12.ArticlePriceDetails{
				&bmecat12.ArticlePriceDetails{
					Dates: []*bmecat12.DateTime{
						bmecat12.NewDateTime(bmecat12.DateTimeValidStartDate, time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)),
						bmecat12.NewDateTime(bmecat12.DateTimeValidEndDate, time.Date(2001, 7, 31, 0, 0, 0, 0, time.UTC)),
					},
					Prices: []*bmecat12.ArticlePrice{
						&bmecat12.ArticlePrice{
							Type:       bmecat12.ArticlePriceTypeNetCustomer,
							Amount:     1499.50,
							Currency:   "EUR",
							Tax:        0.19,
							Factor:     1.0,
							LowerBound: 1,
							Territory:  []string{"DE", "AT"},
						},
						&bmecat12.ArticlePrice{
							Type:       bmecat12.ArticlePriceTypeNetCustomer,
							Amount:     1300.90,
							Currency:   "EUR",
							Tax:        0.19,
							Factor:     1.0,
							LowerBound: 100,
							Territory:  []string{"DE", "AT"},
						},
					},
				},
			},
			// MIME_INFO
			MimeInfo: &bmecat12.MimeInfo{
				Mimes: []*bmecat12.Mime{
					&bmecat12.Mime{
						Type:    "image/jpeg",
						Source:  "55-K-31.jpg",
						Descr:   "Frontansicht des Notebooks",
						Purpose: bmecat12.MimePurposeNormal,
					},
				},
			},
			// USER_DEFINED_EXTENSIONS
			// ARTICLE_REFERENCE
			References: []*bmecat12.ArticleReference{
				&bmecat12.ArticleReference{
					Type:    bmecat12.ArticleReferenceTypeSimilar,
					ArtIDTo: "2000",
				},
			},
		},
	}
	cw := catalogWriter{
		tx:                   bmecat12.NewCatalog,
		language:             "de",
		prevVersion:          0,
		header:               testHeader,
		classificationSystem: classSys,
		articles:             articles,
	}

	var buf bytes.Buffer
	w := bmecat12.NewWriter(&buf, bmecat12.WithIndent("  "))
	ctx := context.Background()
	if err := w.Do(ctx, cw); err != nil {
		t.Fatal(err)
	}

	have := strings.TrimSpace(buf.String())
	data, err := ioutil.ReadFile("testdata/new_catalog_with_blank_classification_system.golden.xml")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(data))
	if want != have {
		// fmt.Println(have)
		diffStrings(t, want, have)
		t.Fail()
	}
}
