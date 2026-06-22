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
	tx             bmecat12.Transaction
	prevVersion    int
	neutral        CatalogWriter
	classification *ClassificationSystem
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

// ClassificationSystem converts the neutral classification system configured on
// the Writer to the bmecat12 type, or returns nil to omit CLASSIFICATION_SYSTEM
// when none was configured (or it carries no groups).
func (c *v12CatalogWriter) ClassificationSystem() *bmecat12.ClassificationSystem {
	return neutralClassificationSystemToV12(c.classification, c.Language())
}

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
				case out <- neutralProductToV12(p, c.Language()):
				case <-ctx.Done():
					errc <- ctx.Err()
					return
				}
			}
		}
	}()
	return out, errc
}

// neutralClassificationSystemToV12 converts the neutral classification system
// to the bmecat12 type. It returns nil for a nil or blank system, so the writer
// omits CLASSIFICATION_SYSTEM exactly as it does for a native bmecat12 source.
func neutralClassificationSystemToV12(cs *ClassificationSystem, lang string) *bmecat12.ClassificationSystem {
	if cs.IsBlank() {
		return nil
	}
	out := &bmecat12.ClassificationSystem{
		Name:        cs.Name,
		FullName:    mlToV12(cs.FullName, lang),
		Version:     cs.Version,
		Description: mlToV12(cs.Description, lang),
		Levels:      cs.Levels,
	}
	for _, ln := range cs.LevelNames {
		if ln == nil {
			continue
		}
		// 1.2 has no per-element lang. Drop variants in a language other than the
		// catalog's so a multi-language level is not duplicated; keep language-less
		// entries always.
		if ln.Lang != "" && lang != "" && ln.Lang != lang {
			continue
		}
		out.LevelNames = append(out.LevelNames, &bmecat12.ClassificationSystemLevelName{
			Level: ln.Level,
			Value: ln.Name,
		})
	}
	for _, g := range cs.Groups {
		if g == nil {
			continue
		}
		out.Groups = append(out.Groups, &bmecat12.ClassificationGroup{
			Type:        g.Type,
			ID:          g.ID,
			Name:        mlToV12(g.Name, lang),
			Description: mlToV12(g.Description, lang),
			ParentID:    g.ParentID,
		})
	}
	return out
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
			Name:        mlToV12(c.Name, c.Language),
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

// collapseLangV12 resolves the single language a BMEcat 1.2 (single-language)
// document should carry: the requested catalog language if any variant uses it,
// otherwise the first variant's own language. Both v12 collapse helpers resolve
// the language the same way, so a single value and a repeating list stay in the
// same language even when no catalog language is configured.
func collapseLangV12(in LocalizedStrings, lang string) string {
	for _, ls := range in {
		if ls.Lang == lang {
			return lang
		}
	}
	if len(in) > 0 {
		return in[0].Lang
	}
	return lang
}

// mlToV12 collapses a neutral LocalizedStrings to a single BMEcat 1.2 scalar
// value. BMEcat 1.2 has no per-element lang attribute, so it emits the variant
// matching the catalog language, falling back to the first variant. Writing a
// multi-language neutral catalog to 1.2 is therefore lossy by design.
func mlToV12(in LocalizedStrings, lang string) string {
	return in.Get(collapseLangV12(in, lang))
}

// mlSliceToV12 collapses a neutral LocalizedStrings for a repeating element
// (such as KEYWORD) to the BMEcat 1.2 scalar list for a single language, so a
// multi-language list does not leak every language's values into 1.2 output.
func mlSliceToV12(in LocalizedStrings, lang string) []string {
	return in.All(collapseLangV12(in, lang))
}

func neutralProductToV12(p *Product, lang string) *bmecat12.Article {
	a := &bmecat12.Article{
		Mode:        p.Mode,
		SupplierAID: p.ID,
		Details: &bmecat12.ArticleDetails{
			DescriptionShort:      mlToV12(p.DescriptionShort, lang),
			DescriptionLong:       mlToV12(p.DescriptionLong, lang),
			EAN:                   p.GTIN,
			SupplierAltAID:        p.SupplierAltID,
			ManufacturerAID:       p.ManufacturerID,
			ManufacturerName:      p.ManufacturerName,
			ManufacturerTypeDescr: mlToV12(p.ManufacturerTypeDescr, lang),
			ERPGroupBuyer:         p.ERPGroupBuyer,
			ERPGroupSupplier:      p.ERPGroupSupplier,
			DeliveryTime:          p.DeliveryTime,
			Keywords:              mlSliceToV12(p.Keywords, lang),
			Remarks:               mlToV12(p.Remarks, lang),
			Segments:              mlSliceToV12(p.Segments, lang),
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
		a.Features = append(a.Features, neutralFeaturesToV12(f, lang))
	}
	a.PriceDetails = neutralPriceDetailsToV12(p)
	if mi := neutralMimesToV12(p.Mimes, lang); mi != nil {
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

func neutralFeaturesToV12(f *Features, lang string) *bmecat12.ArticleFeatures {
	if f == nil {
		return nil
	}
	out := &bmecat12.ArticleFeatures{
		FeatureSystemName: f.SystemName,
		FeatureGroupID:    f.GroupID,
		FeatureGroupName:  mlToV12(f.GroupName, lang),
	}
	for _, ft := range f.Features {
		out.Features = append(out.Features, &bmecat12.Feature{
			Name:   mlToV12(ft.Name, lang),
			Values: mlSliceToV12(ft.Values, lang),
			Unit:   ft.Unit,
		})
	}
	return out
}

// neutralPriceDetailsToV12 builds the ARTICLE_PRICE_DETAILS wrappers for a
// product. When the product carries PriceDetails it emits one wrapper per
// entry, preserving validity dates and grouping; otherwise it falls back to the
// flattened Prices in a single wrapper, so callers that only set Prices keep
// the previous behavior. It returns nil when the product has no prices at all.
func neutralPriceDetailsToV12(p *Product) []*bmecat12.ArticlePriceDetails {
	if len(p.PriceDetails) > 0 {
		var out []*bmecat12.ArticlePriceDetails
		for _, pd := range p.PriceDetails {
			if pd == nil {
				continue
			}
			out = append(out, &bmecat12.ArticlePriceDetails{
				Dates:            validDatesToV12(pd),
				DailyPriceString: dailyPriceString(pd.IsDailyPrice),
				Prices:           neutralPricesToV12(pd.Prices),
			})
		}
		return out
	}
	if prices := neutralPricesToV12(p.Prices); prices != nil {
		return []*bmecat12.ArticlePriceDetails{{Prices: prices}}
	}
	return nil
}

// validDatesToV12 builds the DATETIME entries for a wrapper's validity dates,
// omitting any that are unset.
func validDatesToV12(pd *PriceDetails) []*bmecat12.DateTime {
	var dates []*bmecat12.DateTime
	if pd.ValidStart != nil {
		if dt := bmecat12.NewDateTime(bmecat12.DateTimeValidStartDate, *pd.ValidStart); dt != nil {
			dates = append(dates, dt)
		}
	}
	if pd.ValidEnd != nil {
		if dt := bmecat12.NewDateTime(bmecat12.DateTimeValidEndDate, *pd.ValidEnd); dt != nil {
			dates = append(dates, dt)
		}
	}
	return dates
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

func neutralMimesToV12(mimes []*Mime, lang string) *bmecat12.MimeInfo {
	if len(mimes) == 0 {
		return nil
	}
	mi := &bmecat12.MimeInfo{}
	for _, m := range mimes {
		mi.Mimes = append(mi.Mimes, &bmecat12.Mime{
			Type:    m.Type,
			Source:  mlToV12(m.Source, lang),
			Descr:   mlToV12(m.Descr, lang),
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
