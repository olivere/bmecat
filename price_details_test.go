package bmecat_test

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/olivere/bmecat"
)

// catalog12PriceDetails and catalog2005PriceDetails carry the same product with
// two ARTICLE_PRICE_DETAILS / PRODUCT_PRICE_DETAILS wrappers: a dated, graduated
// block valid for the first half of 2020, and a daily-price block valid from
// mid-2020 with no end date.
const catalog12PriceDetails = `<?xml version="1.0" encoding="UTF-8"?>
<BMECAT version="1.2">
  <HEADER><CATALOG><LANGUAGE>deu</LANGUAGE><CATALOG_ID>C</CATALOG_ID><CATALOG_VERSION>1</CATALOG_VERSION></CATALOG></HEADER>
  <T_NEW_CATALOG>
    <ARTICLE>
      <SUPPLIER_AID>1000</SUPPLIER_AID>
      <ARTICLE_DETAILS><DESCRIPTION_SHORT>Widget</DESCRIPTION_SHORT></ARTICLE_DETAILS>
      <ARTICLE_PRICE_DETAILS>
        <DATETIME type="valid_start_date"><DATE>2020-01-01</DATE></DATETIME>
        <DATETIME type="valid_end_date"><DATE>2020-06-30</DATE></DATETIME>
        <ARTICLE_PRICE price_type="net_list">
          <PRICE_AMOUNT>10</PRICE_AMOUNT><PRICE_CURRENCY>EUR</PRICE_CURRENCY><LOWER_BOUND>1</LOWER_BOUND>
        </ARTICLE_PRICE>
        <ARTICLE_PRICE price_type="net_list">
          <PRICE_AMOUNT>8</PRICE_AMOUNT><PRICE_CURRENCY>EUR</PRICE_CURRENCY><LOWER_BOUND>100</LOWER_BOUND>
        </ARTICLE_PRICE>
      </ARTICLE_PRICE_DETAILS>
      <ARTICLE_PRICE_DETAILS>
        <DATETIME type="valid_start_date"><DATE>2020-07-01</DATE></DATETIME>
        <DAILY_PRICE>true</DAILY_PRICE>
        <ARTICLE_PRICE price_type="net_list">
          <PRICE_AMOUNT>12</PRICE_AMOUNT><PRICE_CURRENCY>EUR</PRICE_CURRENCY>
        </ARTICLE_PRICE>
      </ARTICLE_PRICE_DETAILS>
    </ARTICLE>
  </T_NEW_CATALOG>
</BMECAT>`

const catalog2005PriceDetails = `<?xml version="1.0" encoding="UTF-8"?>
<BMECAT version="2005" xmlns="http://www.bmecat.org/bmecat/2005">
  <HEADER><CATALOG><LANGUAGE>deu</LANGUAGE><CATALOG_ID>C</CATALOG_ID><CATALOG_VERSION>1</CATALOG_VERSION></CATALOG></HEADER>
  <T_NEW_CATALOG>
    <PRODUCT>
      <SUPPLIER_PID>1000</SUPPLIER_PID>
      <PRODUCT_DETAILS><DESCRIPTION_SHORT>Widget</DESCRIPTION_SHORT></PRODUCT_DETAILS>
      <PRODUCT_PRICE_DETAILS>
        <DATETIME type="valid_start_date"><DATE>2020-01-01</DATE></DATETIME>
        <DATETIME type="valid_end_date"><DATE>2020-06-30</DATE></DATETIME>
        <PRODUCT_PRICE price_type="net_list">
          <PRICE_AMOUNT>10</PRICE_AMOUNT><PRICE_CURRENCY>EUR</PRICE_CURRENCY><LOWER_BOUND>1</LOWER_BOUND>
        </PRODUCT_PRICE>
        <PRODUCT_PRICE price_type="net_list">
          <PRICE_AMOUNT>8</PRICE_AMOUNT><PRICE_CURRENCY>EUR</PRICE_CURRENCY><LOWER_BOUND>100</LOWER_BOUND>
        </PRODUCT_PRICE>
      </PRODUCT_PRICE_DETAILS>
      <PRODUCT_PRICE_DETAILS>
        <DATETIME type="valid_start_date"><DATE>2020-07-01</DATE></DATETIME>
        <DAILY_PRICE>true</DAILY_PRICE>
        <PRODUCT_PRICE price_type="net_list">
          <PRICE_AMOUNT>12</PRICE_AMOUNT><PRICE_CURRENCY>EUR</PRICE_CURRENCY>
        </PRODUCT_PRICE>
      </PRODUCT_PRICE_DETAILS>
    </PRODUCT>
  </T_NEW_CATALOG>
</BMECAT>`

// TestReadPriceDetails covers issue #36: the neutral read model preserves the
// price-details wrapper grouping and each wrapper's validity dates, distinguishes
// an absent date (nil) from a real bound, and still exposes the flattened Prices.
func TestReadPriceDetails(t *testing.T) {
	for _, tt := range []struct {
		name string
		doc  string
	}{
		{"1.2", catalog12PriceDetails},
		{"2005", catalog2005PriceDetails},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := read(t, tt.doc)
			if len(c.products) != 1 {
				t.Fatalf("want one product, have %d", len(c.products))
			}
			p := c.products[0]

			// Flattened convenience view spans both wrappers.
			if want, have := 3, len(p.Prices); want != have {
				t.Fatalf("len(Prices) = %d, want %d", have, want)
			}

			// Wrapper grouping is preserved.
			if want, have := 2, len(p.PriceDetails); want != have {
				t.Fatalf("len(PriceDetails) = %d, want %d", have, want)
			}

			// First wrapper: dated, graduated, not a daily price.
			first := p.PriceDetails[0]
			wantStart := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
			wantEnd := time.Date(2020, 6, 30, 0, 0, 0, 0, time.UTC)
			if first.ValidStart == nil || !first.ValidStart.Equal(wantStart) {
				t.Errorf("PriceDetails[0].ValidStart = %v, want %v", first.ValidStart, wantStart)
			}
			if first.ValidEnd == nil || !first.ValidEnd.Equal(wantEnd) {
				t.Errorf("PriceDetails[0].ValidEnd = %v, want %v", first.ValidEnd, wantEnd)
			}
			if first.IsDailyPrice {
				t.Errorf("PriceDetails[0].IsDailyPrice = true, want false")
			}
			if want, have := 2, len(first.Prices); want != have {
				t.Errorf("PriceDetails[0] len(Prices) = %d, want %d", have, want)
			}

			// Second wrapper: open-ended (no end date) and a daily price.
			second := p.PriceDetails[1]
			wantStart2 := time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC)
			if second.ValidStart == nil || !second.ValidStart.Equal(wantStart2) {
				t.Errorf("PriceDetails[1].ValidStart = %v, want %v", second.ValidStart, wantStart2)
			}
			if second.ValidEnd != nil {
				t.Errorf("PriceDetails[1].ValidEnd = %v, want nil (absent date)", second.ValidEnd)
			}
			if !second.IsDailyPrice {
				t.Errorf("PriceDetails[1].IsDailyPrice = false, want true")
			}
			if want, have := 1, len(second.Prices); want != have {
				t.Errorf("PriceDetails[1] len(Prices) = %d, want %d", have, want)
			}
		})
	}
}

// TestPriceDetailsRoundTrip writes a product carrying two dated price-details
// wrappers and reads it back, asserting the grouping, validity dates and daily
// flag survive the round trip for both versions.
func TestPriceDetailsRoundTrip(t *testing.T) {
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2020, 6, 30, 0, 0, 0, 0, time.UTC)
	start2 := time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC)

	product := &bmecat.Product{
		ID:               "1000",
		DescriptionShort: bmecat.Localized("Widget"),
		PriceDetails: []*bmecat.PriceDetails{
			{
				ValidStart: &start,
				ValidEnd:   &end,
				Prices: []*bmecat.Price{
					{Type: "net_list", Amount: 10, Currency: "EUR", LowerBound: 1},
					{Type: "net_list", Amount: 8, Currency: "EUR", LowerBound: 100},
				},
			},
			{
				ValidStart:   &start2,
				IsDailyPrice: true,
				Prices: []*bmecat.Price{
					{Type: "net_list", Amount: 12, Currency: "EUR"},
				},
			},
		},
	}

	for _, version := range []bmecat.Version{bmecat.Version12, bmecat.Version2005} {
		t.Run(version.String(), func(t *testing.T) {
			var buf bytes.Buffer
			cw := &sliceCatalogWriter{header: fullHeader(), products: []*bmecat.Product{product}}
			if err := bmecat.NewWriter(&buf, bmecat.WithVersion(version)).Do(context.Background(), cw); err != nil {
				t.Fatal(err)
			}

			c := read(t, buf.String())
			if len(c.products) != 1 {
				t.Fatalf("want one product, have %d", len(c.products))
			}
			if !reflect.DeepEqual(product.PriceDetails, c.products[0].PriceDetails) {
				t.Errorf("PriceDetails round trip mismatch:\nwant %s\nhave %s",
					formatPriceDetails(product.PriceDetails), formatPriceDetails(c.products[0].PriceDetails))
			}
		})
	}
}

// formatPriceDetails renders PriceDetails for readable test failures, since the
// default %+v prints the validity-date pointers as addresses.
func formatPriceDetails(pds []*bmecat.PriceDetails) string {
	var b strings.Builder
	for i, pd := range pds {
		fmt.Fprintf(&b, "[%d] start=%s end=%s daily=%t prices=%d ",
			i, formatTimePtr(pd.ValidStart), formatTimePtr(pd.ValidEnd), pd.IsDailyPrice, len(pd.Prices))
	}
	return b.String()
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return "<nil>"
	}
	return t.Format("2006-01-02")
}
