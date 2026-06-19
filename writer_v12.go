package bmecat

import (
	"context"

	"github.com/olivere/bmecat/bmecat12"
)

// transactionToV12 maps the neutral Transaction onto the bmecat12 Transaction.
// The two enums share the same constants, but converting explicitly keeps the
// packages decoupled and avoids depending on their iota order matching.
func transactionToV12(t Transaction) bmecat12.Transaction {
	switch t {
	case UpdateProducts:
		return bmecat12.UpdateProducts
	case UpdatePrices:
		return bmecat12.UpdatePrices
	default:
		return bmecat12.NewCatalog
	}
}

// v12CatalogWriter adapts a neutral CatalogWriter to the bmecat12 CatalogWriter
// contract that bmecat12.Writer.Do drives, converting each product on the fly.
type v12CatalogWriter struct {
	tx          bmecat12.Transaction
	prevVersion int
	neutral     CatalogWriter
}

func (c *v12CatalogWriter) Transaction() bmecat12.Transaction { return c.tx }
func (c *v12CatalogWriter) PreviousVersion() int              { return c.prevVersion }

func (c *v12CatalogWriter) Header() *bmecat12.Header {
	return neutralHeaderToV12(c.neutral.Header())
}

func (c *v12CatalogWriter) Language() string {
	if h := c.neutral.Header(); h != nil && h.Catalog != nil {
		return h.Catalog.Language
	}
	return ""
}

// ClassificationSystem returns nil: the neutral model has no classification
// system, so the neutral writer does not emit CLASSIFICATION_SYSTEM.
func (c *v12CatalogWriter) ClassificationSystem() *bmecat12.ClassificationSystem { return nil }

// Articles bridges the neutral product stream to the bmecat12 article stream:
// it reads neutral products from the caller's channel, converts each to a
// bmecat12.Article, and forwards it, so only one product is in flight at a
// time. Producer errors and ctx cancellation are propagated.
func (c *v12CatalogWriter) Articles(ctx context.Context) (<-chan *bmecat12.Article, <-chan error) {
	out := make(chan *bmecat12.Article)
	errc := make(chan error, 1)

	src, srcErr := c.neutral.Products(ctx)
	if src == nil {
		close(out)
		return out, errc
	}

	// out is closed only on clean completion. On an error or cancellation the
	// goroutine sends to errc and returns with out left open, so the version
	// writer's select can only proceed via errc — a closed out and a ready errc
	// are weighed equally by that select, so closing out here would let a clean
	// EOF win the race and silently swallow the error.
	go func() {
		for {
			select {
			case <-ctx.Done():
				errc <- ctx.Err()
				return
			case err := <-srcErr:
				if err != nil {
					errc <- err
					return
				}
				close(out)
				return
			case p, ok := <-src:
				if !ok {
					// The product channel closed. A producer that closes it via
					// defer may have sent its error a moment earlier, so check
					// once more before finishing cleanly.
					select {
					case err := <-srcErr:
						if err != nil {
							errc <- err
							return
						}
					default:
					}
					close(out)
					return
				}
				if p == nil {
					continue
				}
				select {
				case out <- neutralProductToV12(p):
				case <-ctx.Done():
					errc <- ctx.Err()
					return
				}
			}
		}
	}()
	return out, errc
}

func neutralHeaderToV12(h *Header) *bmecat12.Header {
	if h == nil {
		return nil
	}
	out := &bmecat12.Header{GeneratorInfo: h.GeneratorInfo}
	if c := h.Catalog; c != nil {
		out.Catalog = &bmecat12.Catalog{
			Language:    c.Language,
			ID:          c.ID,
			Version:     c.Version,
			Name:        c.Name,
			Currency:    c.Currency,
			Territories: c.Territories,
		}
	}
	if b := h.Buyer; b != nil {
		out.Buyer = &bmecat12.Buyer{Name: b.Name}
		if b.ID != "" {
			out.Buyer.ID = &bmecat12.IDRef{Value: b.ID}
		}
	}
	if s := h.Supplier; s != nil {
		out.Supplier = &bmecat12.Supplier{Name: s.Name}
		if s.ID != "" {
			out.Supplier.ID = &bmecat12.IDRef{Value: s.ID}
		}
	}
	return out
}

func neutralProductToV12(p *Product) *bmecat12.Article {
	a := &bmecat12.Article{
		Mode:        p.Mode,
		SupplierAID: p.ID,
		Details: &bmecat12.ArticleDetails{
			DescriptionShort:      p.DescriptionShort,
			DescriptionLong:       p.DescriptionLong,
			EAN:                   p.GTIN,
			SupplierAltAID:        p.SupplierAltID,
			ManufacturerAID:       p.ManufacturerID,
			ManufacturerName:      p.ManufacturerName,
			ManufacturerTypeDescr: p.ManufacturerTypeDescr,
			ERPGroupBuyer:         p.ERPGroupBuyer,
			ERPGroupSupplier:      p.ERPGroupSupplier,
			DeliveryTime:          p.DeliveryTime,
			Keywords:              p.Keywords,
			Remarks:               p.Remarks,
			Segments:              p.Segments,
		},
	}
	for _, b := range p.BuyerIDs {
		a.Details.BuyerAIDs = append(a.Details.BuyerAIDs, &bmecat12.BuyerAID{Type: b.Type, Value: b.Value})
	}
	for _, s := range p.SpecialTreatmentClasses {
		a.Details.SpecialTreatmentClasses = append(a.Details.SpecialTreatmentClasses, &bmecat12.ArticleSpecialTreatmentClass{Type: s.Type, Value: s.Value})
	}
	for _, s := range p.Status {
		a.Details.ArticleStatus = append(a.Details.ArticleStatus, &bmecat12.ArticleStatus{Type: s.Type, Value: s.Value})
	}
	if od := neutralOrderDetailsToV12(p); od != nil {
		a.OrderDetails = od
	}
	for _, f := range p.Features {
		a.Features = append(a.Features, neutralFeaturesToV12(f))
	}
	if prices := neutralPricesToV12(p.Prices); prices != nil {
		a.PriceDetails = []*bmecat12.ArticlePriceDetails{{Prices: prices}}
	}
	if mi := neutralMimesToV12(p.Mimes); mi != nil {
		a.MimeInfo = mi
	}
	if udx := neutralUDXToV12(p.UDX); udx != nil {
		a.UDX = udx
	}
	return a
}

// neutralOrderDetailsToV12 returns the ARTICLE_ORDER_DETAILS for a product, or
// nil when the product carries no order detail (so the reader's nil-guarded
// round-trip is preserved). QuantityMax has no 1.2 element and is dropped.
func neutralOrderDetailsToV12(p *Product) *bmecat12.ArticleOrderDetails {
	if p.OrderUnit == "" && p.ContentUnit == "" && p.NoCuPerOu == 0 &&
		p.PriceQuantity == 0 && p.QuantityMin == 0 && p.QuantityInterval == 0 {
		return nil
	}
	return &bmecat12.ArticleOrderDetails{
		OrderUnit:        p.OrderUnit,
		ContentUnit:      p.ContentUnit,
		NoCuPerOu:        p.NoCuPerOu,
		PriceQuantity:    p.PriceQuantity,
		QuantityMin:      p.QuantityMin,
		QuantityInterval: p.QuantityInterval,
	}
}

func neutralFeaturesToV12(f *Features) *bmecat12.ArticleFeatures {
	if f == nil {
		return nil
	}
	out := &bmecat12.ArticleFeatures{
		FeatureSystemName: f.SystemName,
		FeatureGroupID:    f.GroupID,
		FeatureGroupName:  f.GroupName,
	}
	for _, ft := range f.Features {
		out.Features = append(out.Features, &bmecat12.Feature{
			Name:   ft.Name,
			Values: ft.Values,
			Unit:   ft.Unit,
		})
	}
	return out
}

func neutralPricesToV12(prices []*Price) []*bmecat12.ArticlePrice {
	if len(prices) == 0 {
		return nil
	}
	out := make([]*bmecat12.ArticlePrice, 0, len(prices))
	for _, p := range prices {
		out = append(out, &bmecat12.ArticlePrice{
			Type:       p.Type,
			Amount:     p.Amount,
			Currency:   p.Currency,
			Tax:        p.Tax,
			Factor:     p.Factor,
			LowerBound: p.LowerBound,
			Territory:  p.Territory,
		})
	}
	return out
}

func neutralMimesToV12(mimes []*Mime) *bmecat12.MimeInfo {
	if len(mimes) == 0 {
		return nil
	}
	mi := &bmecat12.MimeInfo{}
	for _, m := range mimes {
		mi.Mimes = append(mi.Mimes, &bmecat12.Mime{
			Type:    m.Type,
			Source:  m.Source,
			Descr:   m.Descr,
			Purpose: m.Purpose,
			Order:   m.Order,
		})
	}
	return mi
}

func neutralUDXToV12(fields []*UDXField) *bmecat12.UserDefinedExtensions {
	if len(fields) == 0 {
		return nil
	}
	udx := &bmecat12.UserDefinedExtensions{}
	for _, f := range fields {
		udx.Fields.Add(f.Name, f.Value)
	}
	return udx
}
