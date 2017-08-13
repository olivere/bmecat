package v12_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"strings"
	"testing"
	"time"

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

func makeArticleChannel(articles ...*v12.Article) <-chan *v12.Article {
	outCh := make(chan *v12.Article)
	go func() {
		defer close(outCh)
		for _, a := range articles {
			outCh <- a
		}
	}()
	return outCh
}

func TestWriteNewCatalog(t *testing.T) {
	var buf bytes.Buffer
	w := v12.NewWriter(&buf)
	w.Indent = "  "
	w.Language = "de"
	w.Header = testHeader
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
	articlesCh := makeArticleChannel(articles...)
	ctx := context.Background()
	if err := w.Do(ctx, articlesCh); err != nil {
		t.Fatal(err)
	}

	have := strings.TrimSpace(buf.String())
	data, err := ioutil.ReadFile("testdata/new_catalog.golden.xml")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(data))
	if want != have {
		diffStrings(t, want, have)
		t.Fail()
	}
}

func TestWriteUpdateProducts(t *testing.T) {
	var buf bytes.Buffer
	w := v12.NewWriter(&buf)
	w.Indent = "  "
	w.Language = "de"
	w.Transaction = v12.UpdateProducts
	w.Header = testHeader
	w.PreviousVersion = 13
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
	articlesCh := makeArticleChannel(articles...)
	ctx := context.Background()
	if err := w.Do(ctx, articlesCh); err != nil {
		t.Fatal(err)
	}

	have := strings.TrimSpace(buf.String())
	data, err := ioutil.ReadFile("testdata/update_products.golden.xml")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(data))
	if want != have {
		diffStrings(t, want, have)
		t.Fail()
	}
}

func TestWriteUpdatePrices(t *testing.T) {
	var buf bytes.Buffer
	w := v12.NewWriter(&buf)
	w.Indent = "  "
	w.Language = "de"
	w.Transaction = v12.UpdatePrices
	w.Header = testHeader
	w.PreviousVersion = 42
	articles := []*v12.Article{
		&v12.Article{
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
	articlesCh := makeArticleChannel(articles...)
	ctx := context.Background()
	if err := w.Do(ctx, articlesCh); err != nil {
		t.Fatal(err)
	}

	have := strings.TrimSpace(buf.String())
	data, err := ioutil.ReadFile("testdata/update_prices.golden.xml")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(data))
	if want != have {
		diffStrings(t, want, have)
		t.Fail()
	}
}
