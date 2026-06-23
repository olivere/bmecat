package bmecat

import (
	"context"

	"github.com/olivere/bmecat/bmecat2005"
)

// transactionToV2005 maps the neutral Transaction onto the bmecat2005
// Transaction. As with transactionToV12, the conversion is explicit so the
// packages stay decoupled from each other's iota order.
func transactionToV2005(t Transaction) bmecat2005.Transaction {
	switch t {
	case UpdateProducts:
		return bmecat2005.UpdateProducts
	case UpdatePrices:
		return bmecat2005.UpdatePrices
	default:
		return bmecat2005.NewCatalog
	}
}

// v2005CatalogWriter adapts a neutral CatalogWriter to the bmecat2005
// CatalogWriter contract that bmecat2005.Writer.Do drives, converting each
// product on the fly.
type v2005CatalogWriter struct {
	tx             bmecat2005.Transaction
	prevVersion    int
	neutral        CatalogWriter
	classification *ClassificationSystem
}

func (c *v2005CatalogWriter) Transaction() bmecat2005.Transaction { return c.tx }
func (c *v2005CatalogWriter) PreviousVersion() int                { return c.prevVersion }

func (c *v2005CatalogWriter) Header() *bmecat2005.Header {
	return neutralHeaderToV2005(c.neutral.Header())
}

func (c *v2005CatalogWriter) Language() string {
	if h := c.neutral.Header(); h != nil && h.Catalog != nil {
		return h.Catalog.Language
	}
	return ""
}

// ClassificationSystem converts the neutral classification system configured on
// the Writer to the bmecat2005 type, or returns nil to omit CLASSIFICATION_SYSTEM
// when none was configured (or it carries no groups).
func (c *v2005CatalogWriter) ClassificationSystem() *bmecat2005.ClassificationSystem {
	return neutralClassificationSystemToV2005(c.classification)
}

// Products bridges the neutral product stream to the bmecat2005 product stream:
// it reads neutral products from the caller's channel, converts each to a
// bmecat2005.Product, and forwards it, so only one product is in flight at a
// time. Producer errors and ctx cancellation are propagated.
func (c *v2005CatalogWriter) Products(ctx context.Context) (<-chan *bmecat2005.Product, <-chan error) {
	out := make(chan *bmecat2005.Product)
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
				case out <- neutralProductToV2005(p):
				case <-ctx.Done():
					errc <- ctx.Err()
					return
				}
			}
		}
	}()
	return out, errc
}

// neutralClassificationSystemToV2005 converts the neutral classification system
// to the bmecat2005 type. It returns nil for a nil or blank system, so the
// writer omits CLASSIFICATION_SYSTEM exactly as it does for a native bmecat2005
// source.
func neutralClassificationSystemToV2005(cs *ClassificationSystem) *bmecat2005.ClassificationSystem {
	if cs.IsBlank() {
		return nil
	}
	out := &bmecat2005.ClassificationSystem{
		Name:        cs.Name,
		FullName:    localizedToV2005(cs.FullName),
		Version:     cs.Version,
		Description: localizedToV2005(cs.Description),
		Levels:      cs.Levels,
	}
	for _, ln := range cs.LevelNames {
		if ln == nil {
			continue
		}
		out.LevelNames = append(out.LevelNames, &bmecat2005.ClassificationSystemLevelName{
			Level: ln.Level,
			Lang:  ln.Lang,
			Value: ln.Name,
		})
	}
	for _, g := range cs.Groups {
		if g == nil {
			continue
		}
		out.Groups = append(out.Groups, &bmecat2005.ClassificationGroup{
			Type:        g.Type,
			ID:          g.ID,
			Name:        localizedToV2005(g.Name),
			Description: localizedToV2005(g.Description),
			ParentID:    g.ParentID,
		})
	}
	return out
}

func neutralHeaderToV2005(h *Header) *bmecat2005.Header {
	if h == nil {
		return nil
	}
	out := &bmecat2005.Header{GeneratorInfo: h.GeneratorInfo}
	if c := h.Catalog; c != nil {
		out.Catalog = &bmecat2005.Catalog{
			Language:    c.Language,
			ID:          c.ID,
			Version:     c.Version,
			Name:        localizedToV2005(c.Name),
			Currency:    c.Currency,
			Territories: c.Territories,
		}
	}
	if b := h.Buyer; b != nil {
		out.Buyer = &bmecat2005.Buyer{Name: b.Name}
		if b.ID != "" {
			out.Buyer.ID = &bmecat2005.IDRef{Value: b.ID}
		}
	}
	if s := h.Supplier; s != nil {
		out.Supplier = &bmecat2005.Supplier{Name: s.Name}
		if s.ID != "" {
			out.Supplier.ID = &bmecat2005.IDRef{Value: s.ID}
		}
	}
	return out
}

// localizedToV2005 converts a neutral LocalizedStrings into the bmecat2005 one,
// preserving variant order and language.
func localizedToV2005(in LocalizedStrings) bmecat2005.LocalizedStrings {
	if len(in) == 0 {
		return nil
	}
	out := make(bmecat2005.LocalizedStrings, len(in))
	for i, ls := range in {
		out[i] = bmecat2005.LocalizedString{Lang: ls.Lang, Value: ls.Value}
	}
	return out
}

func neutralProductToV2005(p *Product) *bmecat2005.Product {
	prod := &bmecat2005.Product{
		Mode:        p.Mode,
		SupplierPID: p.ID,
		Details: &bmecat2005.ProductDetails{
			DescriptionShort:      localizedToV2005(p.DescriptionShort),
			DescriptionLong:       localizedToV2005(p.DescriptionLong),
			SupplierAltPID:        p.SupplierAltID,
			ManufacturerPID:       p.ManufacturerID,
			ManufacturerName:      p.ManufacturerName,
			ManufacturerTypeDescr: localizedToV2005(p.ManufacturerTypeDescr),
			ERPGroupBuyer:         p.ERPGroupBuyer,
			ERPGroupSupplier:      p.ERPGroupSupplier,
			DeliveryTime:          p.DeliveryTime,
			Keywords:              localizedToV2005(p.Keywords),
			Remarks:               localizedToV2005(p.Remarks),
			Segments:              localizedToV2005(p.Segments),
		},
	}
	// INTERNATIONAL_PID (the 2005 replacement for EAN). PIDs takes precedence
	// when set so a read-modify-write preserves every typed identifier;
	// otherwise emit a single gtin-typed PID from the GTIN convenience field.
	if len(p.PIDs) > 0 {
		pids := make([]*bmecat2005.InternationalPID, 0, len(p.PIDs))
		for _, pid := range p.PIDs {
			if pid != nil {
				pids = append(pids, &bmecat2005.InternationalPID{Type: pid.Type, Value: pid.Value})
			}
		}
		prod.Details.InternationalPIDs = pids
	} else if p.GTIN != "" {
		prod.Details.InternationalPIDs = []*bmecat2005.InternationalPID{{Type: "gtin", Value: p.GTIN}}
	}
	prod.Details.BuyerPIDs = make([]*bmecat2005.BuyerPID, 0, len(p.BuyerIDs))
	for _, b := range p.BuyerIDs {
		prod.Details.BuyerPIDs = append(prod.Details.BuyerPIDs, &bmecat2005.BuyerPID{Type: b.Type, Value: b.Value})
	}
	prod.Details.SpecialTreatmentClasses = make([]*bmecat2005.ProductSpecialTreatmentClass, 0, len(p.SpecialTreatmentClasses))
	for _, s := range p.SpecialTreatmentClasses {
		prod.Details.SpecialTreatmentClasses = append(prod.Details.SpecialTreatmentClasses, &bmecat2005.ProductSpecialTreatmentClass{Type: s.Type, Value: s.Value})
	}
	prod.Details.ProductStatus = make([]*bmecat2005.ProductStatus, 0, len(p.Status))
	for _, s := range p.Status {
		prod.Details.ProductStatus = append(prod.Details.ProductStatus, &bmecat2005.ProductStatus{Type: s.Type, Value: s.Value})
	}
	if od := neutralOrderDetailsToV2005(p); od != nil {
		prod.OrderDetails = od
	}
	prod.Features = make([]*bmecat2005.ProductFeatures, 0, len(p.Features))
	for _, f := range p.Features {
		prod.Features = append(prod.Features, neutralFeaturesToV2005(f))
	}
	prod.PriceDetails = neutralPriceDetailsToV2005(p)
	if mi := neutralMimesToV2005(p.Mimes); mi != nil {
		prod.MimeInfo = mi
	}
	if udx := neutralUDXToV2005(p.UDX); udx != nil {
		prod.UDX = udx
	}
	return prod
}

// neutralOrderDetailsToV2005 returns the PRODUCT_ORDER_DETAILS for a product,
// or nil when the product carries no order detail. Unlike 1.2, 2005 has
// QUANTITY_MAX, so QuantityMax is carried here.
func neutralOrderDetailsToV2005(p *Product) *bmecat2005.ProductOrderDetails {
	if p.OrderUnit == "" && p.ContentUnit == "" && p.NoCuPerOu == 0 &&
		p.PriceQuantity == 0 && p.QuantityMin == 0 && p.QuantityInterval == 0 &&
		p.QuantityMax == 0 {
		return nil
	}
	return &bmecat2005.ProductOrderDetails{
		OrderUnit:        p.OrderUnit,
		ContentUnit:      p.ContentUnit,
		NoCuPerOu:        p.NoCuPerOu,
		PriceQuantity:    p.PriceQuantity,
		QuantityMin:      p.QuantityMin,
		QuantityInterval: p.QuantityInterval,
		QuantityMax:      p.QuantityMax,
	}
}

func neutralFeaturesToV2005(f *Features) *bmecat2005.ProductFeatures {
	if f == nil {
		return nil
	}
	out := &bmecat2005.ProductFeatures{
		FeatureSystemName: f.SystemName,
		FeatureGroupID:    f.GroupID,
		FeatureGroupName:  localizedToV2005(f.GroupName),
	}
	out.Features = make([]*bmecat2005.Feature, 0, len(f.Features))
	for _, ft := range f.Features {
		out.Features = append(out.Features, &bmecat2005.Feature{
			Name:   localizedToV2005(ft.Name),
			Values: localizedToV2005(ft.Values),
			Unit:   ft.Unit,
		})
	}
	return out
}

// neutralPriceDetailsToV2005 builds the PRODUCT_PRICE_DETAILS wrappers for a
// product. When the product carries PriceDetails it emits one wrapper per
// entry, preserving validity dates and grouping; otherwise it falls back to the
// flattened Prices in a single wrapper, so callers that only set Prices keep
// the previous behavior. It returns nil when the product has no prices at all.
func neutralPriceDetailsToV2005(p *Product) []*bmecat2005.ProductPriceDetails {
	if len(p.PriceDetails) > 0 {
		out := make([]*bmecat2005.ProductPriceDetails, 0, len(p.PriceDetails))
		for _, pd := range p.PriceDetails {
			if pd == nil {
				continue
			}
			out = append(out, &bmecat2005.ProductPriceDetails{
				Dates:            validDatesToV2005(pd),
				DailyPriceString: dailyPriceString(pd.IsDailyPrice),
				Prices:           neutralPricesToV2005(pd.Prices),
			})
		}
		return out
	}
	if prices := neutralPricesToV2005(p.Prices); prices != nil {
		return []*bmecat2005.ProductPriceDetails{{Prices: prices}}
	}
	return nil
}

// validDatesToV2005 builds the DATETIME entries for a wrapper's validity dates,
// omitting any that are unset.
func validDatesToV2005(pd *PriceDetails) []*bmecat2005.DateTime {
	var dates []*bmecat2005.DateTime
	if pd.ValidStart != nil {
		if dt := bmecat2005.NewDateTime(bmecat2005.DateTimeValidStartDate, *pd.ValidStart); dt != nil {
			dates = append(dates, dt)
		}
	}
	if pd.ValidEnd != nil {
		if dt := bmecat2005.NewDateTime(bmecat2005.DateTimeValidEndDate, *pd.ValidEnd); dt != nil {
			dates = append(dates, dt)
		}
	}
	return dates
}

func neutralPricesToV2005(prices []*Price) []*bmecat2005.ProductPrice {
	if len(prices) == 0 {
		return nil
	}
	out := make([]*bmecat2005.ProductPrice, 0, len(prices))
	for _, p := range prices {
		out = append(out, &bmecat2005.ProductPrice{
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

func neutralMimesToV2005(mimes []*Mime) *bmecat2005.MimeInfo {
	if len(mimes) == 0 {
		return nil
	}
	mi := &bmecat2005.MimeInfo{Mimes: make([]*bmecat2005.Mime, 0, len(mimes))}
	for _, m := range mimes {
		mi.Mimes = append(mi.Mimes, &bmecat2005.Mime{
			Type:    m.Type,
			Source:  localizedToV2005(m.Source),
			Descr:   localizedToV2005(m.Descr),
			Purpose: m.Purpose,
			Order:   m.Order,
		})
	}
	return mi
}

func neutralUDXToV2005(fields []*UDXField) *bmecat2005.UserDefinedExtensions {
	if len(fields) == 0 {
		return nil
	}
	udx := &bmecat2005.UserDefinedExtensions{}
	for _, f := range fields {
		udx.Fields.Add(f.Name, f.Value)
	}
	return udx
}
