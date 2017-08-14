package v12_test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/olivere/nullable"

	"github.com/olivere/bmecat/v12"
)

var (
	testHeader = &v12.Header{
		GeneratorInfo: "BMEcat Generator",
		Catalog: &v12.Catalog{
			Language:    "de",
			ID:          "CAT1",
			Version:     "1.0.0",
			Name:        "Katalogbezeichnung",
			GenDate:     v12.NewDateTime(v12.DateTimeGenerationDate, time.Date(2000, 10, 24, 20, 38, 00, 0, time.UTC)),
			Territories: []string{"DE", "AT"},
			Currency:    "EUR",
			MimeRoot:    "https://example.com/images",
			PriceFlags: []v12.PriceFlag{
				v12.CatalogIncludesPacking,
			},
		},
		Buyer: &v12.Buyer{
			ID:   &v12.IDRef{Type: "buyer", Value: "BUYCO"},
			Name: "BuyCo Inc.",
		},
		Agreements: []*v12.Agreement{
			&v12.Agreement{
				ID: "23/97",
				Dates: []*v12.DateTime{
					v12.NewDateTime(v12.DateTimeAgreementStartDate, time.Date(1999, 3, 17, 0, 0, 0, 0, time.UTC)),
					v12.NewDateTime(v12.DateTimeAgreementStartDate, time.Date(2002, 5, 31, 0, 0, 0, 0, time.UTC)),
				},
			},
		},
		Supplier: &v12.Supplier{
			ID:   &v12.IDRef{Type: "supplier", Value: "SUPPLYCO"},
			Name: "SupplyCo Ltd.",
			Address: &v12.Address{
				Type: "supplier",
				City: "London",
			},
			MimeInfo: &v12.MimeInfo{
				Mimes: []*v12.Mime{
					&v12.Mime{Type: "image/jpeg", Source: "supplier_logo.jpg", Purpose: "logo"},
				},
			},
		},
		UDX: &v12.UserDefinedExtensions{
			Fields: []*v12.UserDefinedExtensionField{
				&v12.UserDefinedExtensionField{
					Name:  "SYSTEM.CUSTOM_FIELD1",
					Value: "A",
				},
				&v12.UserDefinedExtensionField{
					Name:  "SYSTEM.CUSTOM_FIELD3",
					Value: "C",
				},
			},
		},
	}
)

type catalogWriter struct {
	tx                   v12.Transaction
	language             string
	prevVersion          int
	header               *v12.Header
	classificationSystem *v12.ClassificationSystem
	articles             []*v12.Article
}

func (w catalogWriter) Transaction() v12.Transaction {
	return w.tx
}

func (w catalogWriter) Language() string {
	return w.language
}

func (w catalogWriter) PreviousVersion() int {
	return w.prevVersion
}

func (w catalogWriter) Header() *v12.Header {
	return w.header
}

func (w catalogWriter) ClassificationSystem() *v12.ClassificationSystem {
	return w.classificationSystem
}

func (w catalogWriter) Articles() <-chan *v12.Article {
	if len(w.articles) == 0 {
		return nil
	}
	ch := make(chan *v12.Article)
	go func() {
		defer close(ch)
		for _, a := range w.articles {
			ch <- a
		}
	}()
	return ch
}

func TestWriteNewCatalog(t *testing.T) {
	classSys := &v12.ClassificationSystem{
		Name:     "udf_Supplier-1.0",
		FullName: testHeader.Supplier.Name,
		Groups: []*v12.ClassificationGroup{
			{
				ID:   "1",
				Name: "Hardware",
				Type: "node",
			},
			{
				ID:       "2",
				Name:     "Notebook",
				ParentID: nullable.StringPtr("1"),
				Type:     "node",
			},
			{
				ID:       "3",
				Name:     "Desktop",
				ParentID: nullable.StringPtr("1"),
				Type:     "node",
			},
			{
				ID:       "4",
				Name:     "PC",
				ParentID: nullable.StringPtr("2"),
				Type:     "leaf",
			},
			{
				ID:       "5",
				Name:     "Mac",
				ParentID: nullable.StringPtr("2"),
				Type:     "leaf",
			},
		},
	}
	articles := []*v12.Article{
		&v12.Article{
			SupplierAID: "1000",
			Details: &v12.ArticleDetails{
				DescriptionShort: `Apple MacBook Pro 13"`,
				DescriptionLong:  `Das Kraftpaket unter den Notebooks.`,
				EAN:              "8712670911213",
				SupplierAltAID:   "ALT-1000",
				BuyerAIDs: []*v12.BuyerAID{
					&v12.BuyerAID{Type: "KMF", Value: "78787"},
				},
				ManufacturerAID:  "MPN",
				ManufacturerName: "Microsoft",
				DeliveryTime:     1.5,
				SpecialTreatmentClasses: []*v12.ArticleSpecialTreatmentClass{
					&v12.ArticleSpecialTreatmentClass{
						Type:  "GGVS",
						Value: "1201",
					},
				},
				Keywords: []string{"Notebook", "Hardware"},
				Remarks:  "Noch heute bestellen!",
				ArticleStatus: []*v12.ArticleStatus{
					&v12.ArticleStatus{Type: v12.ArticleStatusCoreArticle, Value: "Kernsortiment"},
				},
			},
			Features: []*v12.ArticleFeatures{
				&v12.ArticleFeatures{
					FeatureSystemName: "ECLASS-5.1",
					FeatureGroupID:    "19010203",
					Features: []*v12.Feature{
						&v12.Feature{
							Name:   "Netzspannung",
							Values: []string{"110", "220"},
							Unit:   "VLT",
						},
					},
				},
				&v12.ArticleFeatures{
					FeatureSystemName: "udf_Supplier-1.0",
					FeatureGroupID:    "5",
				},
			},
			OrderDetails: &v12.ArticleOrderDetails{
				OrderUnit:     "BOX",
				NoCuPerOu:     6.0,
				ContentUnit:   "PCE",
				PriceQuantity: 1,
				QuantityMin:   1,
			},
			PriceDetails: []*v12.ArticlePriceDetails{
				&v12.ArticlePriceDetails{
					Dates: []*v12.DateTime{
						v12.NewDateTime(v12.DateTimeValidStartDate, time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)),
						v12.NewDateTime(v12.DateTimeValidEndDate, time.Date(2001, 7, 31, 0, 0, 0, 0, time.UTC)),
					},
					Prices: []*v12.ArticlePrice{
						&v12.ArticlePrice{
							Type:       v12.ArticlePriceTypeNetCustomer,
							Amount:     1499.50,
							Currency:   "EUR",
							Tax:        0.19,
							Factor:     1.0,
							LowerBound: 1,
							Territory:  []string{"DE", "AT"},
						},
						&v12.ArticlePrice{
							Type:       v12.ArticlePriceTypeNetCustomer,
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
			MimeInfo: &v12.MimeInfo{
				Mimes: []*v12.Mime{
					&v12.Mime{
						Type:    "image/jpeg",
						Source:  "55-K-31.jpg",
						Descr:   "Frontansicht des Notebooks",
						Purpose: v12.MimePurposeNormal,
					},
				},
			},
			// USER_DEFINED_EXTENSIONS
			// ARTICLE_REFERENCE
			References: []*v12.ArticleReference{
				&v12.ArticleReference{
					Type:    v12.ArticleReferenceTypeSimilar,
					ArtIDTo: "2000",
				},
			},
		},
	}
	cw := catalogWriter{
		tx:                   v12.NewCatalog,
		language:             "de",
		prevVersion:          0,
		header:               testHeader,
		classificationSystem: classSys,
		articles:             articles,
	}

	var buf bytes.Buffer
	w := v12.NewWriter(&buf, v12.WithIndent("  "))
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
	classSys := &v12.ClassificationSystem{
		Name:     "udf_Supplier-1.0",
		FullName: testHeader.Supplier.Name,
		Groups: []*v12.ClassificationGroup{
			{
				ID:   "1",
				Name: "Hardware",
				Type: "node",
			},
			{
				ID:       "2",
				Name:     "Notebook",
				ParentID: nullable.StringPtr("1"),
				Type:     "node",
			},
			{
				ID:       "3",
				Name:     "Desktop",
				ParentID: nullable.StringPtr("1"),
				Type:     "node",
			},
			{
				ID:       "4",
				Name:     "PC",
				ParentID: nullable.StringPtr("2"),
				Type:     "leaf",
			},
			{
				ID:       "5",
				Name:     "Mac",
				ParentID: nullable.StringPtr("2"),
				Type:     "leaf",
			},
		},
	}
	articles := []*v12.Article{
		&v12.Article{
			Mode:        "update",
			SupplierAID: "1000",
			Details: &v12.ArticleDetails{
				DescriptionShort: `Apple MacBook Pro 13"`,
				DescriptionLong:  `Das Kraftpaket unter den Notebooks.`,
				EAN:              "8712670911213",
				SupplierAltAID:   "ALT-1000",
				BuyerAIDs: []*v12.BuyerAID{
					&v12.BuyerAID{Type: "KMF", Value: "78787"},
				},
				ManufacturerAID:  "MPN",
				ManufacturerName: "Microsoft",
				DeliveryTime:     1.5,
				SpecialTreatmentClasses: []*v12.ArticleSpecialTreatmentClass{
					&v12.ArticleSpecialTreatmentClass{
						Type:  "GGVS",
						Value: "1201",
					},
				},
				Keywords: []string{"Notebook", "Hardware"},
				Remarks:  "Noch heute bestellen!",
				ArticleStatus: []*v12.ArticleStatus{
					&v12.ArticleStatus{Type: v12.ArticleStatusCoreArticle, Value: "Kernsortiment"},
				},
			},
			Features: []*v12.ArticleFeatures{
				&v12.ArticleFeatures{
					FeatureSystemName: "ECLASS-5.1",
					FeatureGroupID:    "19010203",
					Features: []*v12.Feature{
						&v12.Feature{
							Name:   "Netzspannung",
							Values: []string{"110", "220"},
							Unit:   "VLT",
						},
					},
				},
				&v12.ArticleFeatures{
					FeatureSystemName: "udf_Supplier-1.0",
					FeatureGroupID:    "5",
				},
			},
			OrderDetails: &v12.ArticleOrderDetails{
				OrderUnit:     "BOX",
				NoCuPerOu:     6.0,
				ContentUnit:   "PCE",
				PriceQuantity: 1,
				QuantityMin:   1,
			},
			PriceDetails: []*v12.ArticlePriceDetails{
				&v12.ArticlePriceDetails{
					Dates: []*v12.DateTime{
						v12.NewDateTime(v12.DateTimeValidStartDate, time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)),
						v12.NewDateTime(v12.DateTimeValidEndDate, time.Date(2001, 7, 31, 0, 0, 0, 0, time.UTC)),
					},
					Prices: []*v12.ArticlePrice{
						&v12.ArticlePrice{
							Type:       v12.ArticlePriceTypeNetCustomer,
							Amount:     1499.50,
							Currency:   "EUR",
							Tax:        0.19,
							Factor:     1.0,
							LowerBound: 1,
							Territory:  []string{"DE", "AT"},
						},
						&v12.ArticlePrice{
							Type:       v12.ArticlePriceTypeNetCustomer,
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
			MimeInfo: &v12.MimeInfo{
				Mimes: []*v12.Mime{
					&v12.Mime{
						Type:    "image/jpeg",
						Source:  "55-K-31.jpg",
						Descr:   "Frontansicht des Notebooks",
						Purpose: v12.MimePurposeNormal,
					},
				},
			},
			// USER_DEFINED_EXTENSIONS
			// ARTICLE_REFERENCE
			References: []*v12.ArticleReference{
				&v12.ArticleReference{
					Type:    v12.ArticleReferenceTypeSimilar,
					ArtIDTo: "2000",
				},
			},
		},
		&v12.Article{
			Mode:        "delete",
			SupplierAID: "2000",
		},
	}
	cw := catalogWriter{
		tx:                   v12.UpdateProducts,
		language:             "de",
		prevVersion:          13,
		header:               testHeader,
		classificationSystem: classSys,
		articles:             articles,
	}

	var buf bytes.Buffer
	w := v12.NewWriter(&buf, v12.WithIndent("  "))

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
	articles := []*v12.Article{
		{
			SupplierAID: "1000",
			PriceDetails: []*v12.ArticlePriceDetails{
				&v12.ArticlePriceDetails{
					Dates: []*v12.DateTime{
						v12.NewDateTime(v12.DateTimeValidStartDate, time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)),
						v12.NewDateTime(v12.DateTimeValidEndDate, time.Date(2001, 7, 31, 0, 0, 0, 0, time.UTC)),
					},
					Prices: []*v12.ArticlePrice{
						&v12.ArticlePrice{
							Type:       v12.ArticlePriceTypeNetCustomer,
							Amount:     1499.50,
							Currency:   "EUR",
							Tax:        0.19,
							Factor:     1.0,
							LowerBound: 1,
							Territory:  []string{"DE", "AT"},
						},
						&v12.ArticlePrice{
							Type:       v12.ArticlePriceTypeNetCustomer,
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
		tx:                   v12.UpdatePrices,
		language:             "de",
		prevVersion:          42,
		header:               testHeader,
		classificationSystem: nil,
		articles:             articles,
	}

	var buf bytes.Buffer
	w := v12.NewWriter(&buf, v12.WithIndent("  "))

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
