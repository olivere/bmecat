package bmecat

import (
	"io"
	"time"

	"github.com/olivere/bmecat/bmecat12"
)

// v12Adapter implements the bmecat12 handler interfaces, converts each
// version-specific element into the neutral type, and forwards it to the
// caller's neutral handler. It implements all bmecat12 callbacks; each one is
// a no-op when the caller did not provide the matching neutral handler.
type v12Adapter struct {
	header       HeaderHandler
	product      ProductHandler
	catalogGroup CatalogGroupHandler
	classifGroup ClassificationGroupHandler
	complete     CompletionHandler

	// transaction is the document-level transaction detected in phase 1; it is
	// stamped onto the neutral Header, which the version-specific Header lacks.
	transaction Transaction

	// headerErr retains a non-EOF error returned by the neutral HeaderHandler.
	// The bmecat12 reader currently swallows such errors (see issue #16), so
	// the facade surfaces it after Do returns to honor the HeaderHandler
	// contract.
	headerErr error
}

func newV12Adapter(handler any, transaction Transaction) *v12Adapter {
	a := &v12Adapter{transaction: transaction}
	if h, ok := handler.(HeaderHandler); ok {
		a.header = h
	}
	if h, ok := handler.(ProductHandler); ok {
		a.product = h
	}
	if h, ok := handler.(CatalogGroupHandler); ok {
		a.catalogGroup = h
	}
	if h, ok := handler.(ClassificationGroupHandler); ok {
		a.classifGroup = h
	}
	if h, ok := handler.(CompletionHandler); ok {
		a.complete = h
	}
	return a
}

func (a *v12Adapter) HandleHeader(h *bmecat12.Header) error {
	if a.header == nil {
		return nil
	}
	hdr := convertV12Header(h)
	hdr.Transaction = a.transaction
	err := a.header.HandleHeader(hdr)
	if err != nil && err != io.EOF {
		a.headerErr = err
	}
	return err
}

func (a *v12Adapter) HandleArticle(art *bmecat12.Article) error {
	if a.product == nil {
		return nil
	}
	return a.product.HandleProduct(convertV12Product(art))
}

func (a *v12Adapter) HandleCatalogGroup(cg *bmecat12.CatalogGroup) error {
	if a.catalogGroup == nil {
		return nil
	}
	return a.catalogGroup.HandleCatalogGroup(&CatalogGroup{
		Type:        cg.Type,
		ID:          cg.ID,
		Name:        localizedFromV12(cg.Name),
		Description: localizedFromV12(cg.Description),
		ParentID:    cg.ParentID,
		Order:       cg.Order,
	})
}

func (a *v12Adapter) HandleClassificationGroup(cg *bmecat12.ClassificationGroup) error {
	if a.classifGroup == nil {
		return nil
	}
	return a.classifGroup.HandleClassificationGroup(&ClassificationGroup{
		Type:        cg.Type,
		ID:          cg.ID,
		Name:        localizedFromV12(cg.Name),
		Description: localizedFromV12(cg.Description),
		ParentID:    cg.ParentID,
	})
}

func (a *v12Adapter) HandleComplete() {
	if a.complete != nil {
		a.complete.HandleComplete()
	}
}

func convertV12Header(h *bmecat12.Header) *Header {
	if h == nil {
		return nil
	}
	out := &Header{
		Version:                      Version12,
		GeneratorInfo:                h.GeneratorInfo,
		NumberOfProducts:             h.NumberOfArticles,
		NumberOfCatalogGroups:        h.NumberOfCatalogGroups,
		NumberOfClassificationGroups: h.NumberOfClassificationGroups,
	}
	if c := h.Catalog; c != nil {
		out.Catalog = &Catalog{
			Language:    c.Language,
			ID:          c.ID,
			Version:     c.Version,
			Name:        localizedFromV12(c.Name),
			Currency:    c.Currency,
			Territories: c.Territories,
		}
	}
	if b := h.Buyer; b != nil {
		out.Buyer = &Buyer{Name: b.Name}
		if b.ID != nil {
			out.Buyer.ID = b.ID.Value
		}
	}
	if s := h.Supplier; s != nil {
		out.Supplier = &Supplier{Name: s.Name}
		if s.ID != nil {
			out.Supplier.ID = s.ID.Value
		}
	}
	return out
}

func convertV12Product(a *bmecat12.Article) *Product {
	if a == nil {
		return nil
	}
	out := &Product{
		ID:              a.SupplierAID,
		Mode:            a.Mode,
		CatalogGroupIDs: a.CatalogGroupIDs,
	}
	if d := a.Details; d != nil {
		out.GTIN = d.EAN
		out.DescriptionShort = localizedFromV12(d.DescriptionShort)
		out.DescriptionLong = localizedFromV12(d.DescriptionLong)
		out.SupplierAltID = d.SupplierAltAID
		out.ManufacturerID = d.ManufacturerAID
		out.ManufacturerName = d.ManufacturerName
		out.ManufacturerTypeDescr = localizedFromV12(d.ManufacturerTypeDescr)
		out.ERPGroupBuyer = d.ERPGroupBuyer
		out.ERPGroupSupplier = d.ERPGroupSupplier
		out.DeliveryTime = d.DeliveryTime
		out.Keywords = localizedSliceFromV12(d.Keywords)
		out.Remarks = localizedFromV12(d.Remarks)
		out.Segments = localizedSliceFromV12(d.Segments)
		for _, b := range d.BuyerAIDs {
			if b != nil {
				out.BuyerIDs = append(out.BuyerIDs, &TypedValue{Type: b.Type, Value: b.Value})
			}
		}
		for _, s := range d.SpecialTreatmentClasses {
			if s != nil {
				out.SpecialTreatmentClasses = append(out.SpecialTreatmentClasses, &TypedValue{Type: s.Type, Value: s.Value})
			}
		}
		for _, s := range d.ArticleStatus {
			if s != nil {
				out.Status = append(out.Status, &TypedValue{Type: s.Type, Value: s.Value})
			}
		}
	}
	if od := a.OrderDetails; od != nil {
		out.OrderUnit = od.OrderUnit
		out.ContentUnit = od.ContentUnit
		out.NoCuPerOu = od.NoCuPerOu
		out.PriceQuantity = od.PriceQuantity
		out.QuantityMin = od.QuantityMin
		out.QuantityInterval = od.QuantityInterval
		// QuantityMax has no BMEcat 1.2 equivalent; it stays zero.
	}
	out.UDX = convertV12UDX(a.UDX)
	for _, f := range a.Features {
		out.Features = append(out.Features, convertV12Features(f))
	}
	for _, pd := range a.PriceDetails {
		npd := convertV12PriceDetails(pd)
		if npd == nil {
			continue
		}
		out.PriceDetails = append(out.PriceDetails, npd)
		out.Prices = append(out.Prices, npd.Prices...)
	}
	if mi := a.MimeInfo; mi != nil {
		for _, m := range mi.Mimes {
			out.Mimes = append(out.Mimes, &Mime{
				Type:    m.Type,
				Source:  localizedFromV12(m.Source),
				Descr:   localizedFromV12(m.Descr),
				Purpose: m.Purpose,
				Order:   m.Order,
			})
		}
	}
	return out
}

// localizedFromV12 lifts a scalar BMEcat 1.2 value into the neutral
// LocalizedStrings. BMEcat 1.2 has no per-element lang attribute, so the result
// carries a single language-less variant (or nil for an empty value).
func localizedFromV12(s string) LocalizedStrings {
	if s == "" {
		return nil
	}
	return Localized(s)
}

// localizedSliceFromV12 lifts a list of scalar BMEcat 1.2 values (e.g. KEYWORD)
// into the neutral LocalizedStrings, each as a language-less variant.
func localizedSliceFromV12(values []string) LocalizedStrings {
	if len(values) == 0 {
		return nil
	}
	return Localized(values...)
}

func convertV12PriceDetails(pd *bmecat12.ArticlePriceDetails) *PriceDetails {
	if pd == nil {
		return nil
	}
	out := &PriceDetails{
		ValidStart:   v12ValidDate(pd.Dates, bmecat12.DateTimeValidStartDate),
		ValidEnd:     v12ValidDate(pd.Dates, bmecat12.DateTimeValidEndDate),
		IsDailyPrice: pd.IsDailyPrice(),
	}
	for _, p := range pd.Prices {
		if p == nil {
			continue
		}
		out.Prices = append(out.Prices, &Price{
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

// v12ValidDate returns the parsed value of the first DATETIME of the given type,
// or nil when it is absent or unparseable. Unlike ArticlePriceDetails'
// ValidStartDate/ValidEndDate, it does not substitute a sentinel default, so the
// neutral model can distinguish "no date" from a real bound.
func v12ValidDate(dates []*bmecat12.DateTime, typ string) *time.Time {
	for _, d := range dates {
		if d == nil || d.Type != typ {
			continue
		}
		t, err := d.Time()
		if err != nil {
			return nil
		}
		return &t
	}
	return nil
}

func convertV12Features(f *bmecat12.ArticleFeatures) *Features {
	if f == nil {
		return nil
	}
	out := &Features{
		SystemName: f.FeatureSystemName,
		GroupID:    f.FeatureGroupID,
		GroupName:  localizedFromV12(f.FeatureGroupName),
	}
	for _, ft := range f.Features {
		out.Features = append(out.Features, &Feature{
			Name:   localizedFromV12(ft.Name),
			Values: localizedSliceFromV12(ft.Values),
			Unit:   ft.Unit,
		})
	}
	return out
}

func convertV12UDX(udx *bmecat12.UserDefinedExtensions) []*UDXField {
	if udx == nil {
		return nil
	}
	var out []*UDXField
	for _, f := range udx.Fields {
		if f != nil {
			out = append(out, &UDXField{Name: f.Name, Value: f.Value})
		}
	}
	return out
}
