package bmecat

import (
	"strings"
	"time"

	"github.com/olivere/bmecat/bmecat2005"
)

// v2005Adapter implements the bmecat2005 handler interfaces, converts each
// version-specific element into the neutral type, and forwards it to the
// caller's neutral handler. It implements all bmecat2005 callbacks; each one
// is a no-op when the caller did not provide the matching neutral handler.
type v2005Adapter struct {
	header       HeaderHandler
	product      ProductHandler
	catalogGroup CatalogGroupHandler
	classifGroup ClassificationGroupHandler
	complete     CompletionHandler

	// transaction is the document-level transaction detected in phase 1; it is
	// stamped onto the neutral Header, which the version-specific Header lacks.
	transaction Transaction
}

func newV2005Adapter(handler any, transaction Transaction) *v2005Adapter {
	a := &v2005Adapter{transaction: transaction}
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

func (a *v2005Adapter) HandleHeader(h *bmecat2005.Header) error {
	if a.header == nil {
		return nil
	}
	hdr := convertV2005Header(h)
	hdr.Transaction = a.transaction
	return a.header.HandleHeader(hdr)
}

func (a *v2005Adapter) HandleProduct(p *bmecat2005.Product) error {
	if a.product == nil {
		return nil
	}
	return a.product.HandleProduct(convertV2005Product(p))
}

func (a *v2005Adapter) HandleCatalogGroup(cg *bmecat2005.CatalogGroup) error {
	if a.catalogGroup == nil {
		return nil
	}
	return a.catalogGroup.HandleCatalogGroup(&CatalogGroup{
		Type:        cg.Type,
		ID:          cg.ID,
		Name:        localizedFromV2005(cg.Name),
		Description: localizedFromV2005(cg.Description),
		ParentID:    cg.ParentID,
		Order:       cg.Order,
	})
}

func (a *v2005Adapter) HandleClassificationGroup(cg *bmecat2005.ClassificationGroup) error {
	if a.classifGroup == nil {
		return nil
	}
	return a.classifGroup.HandleClassificationGroup(&ClassificationGroup{
		Type:        cg.Type,
		ID:          cg.ID,
		Name:        localizedFromV2005(cg.Name),
		Description: localizedFromV2005(cg.Description),
		ParentID:    cg.ParentID,
	})
}

func (a *v2005Adapter) HandleComplete() {
	if a.complete != nil {
		a.complete.HandleComplete()
	}
}

func convertV2005Header(h *bmecat2005.Header) *Header {
	if h == nil {
		return nil
	}
	out := &Header{
		Version:                      Version2005,
		GeneratorInfo:                h.GeneratorInfo,
		NumberOfProducts:             h.NumberOfProducts,
		NumberOfCatalogGroups:        h.NumberOfCatalogGroups,
		NumberOfClassificationGroups: h.NumberOfClassificationGroups,
	}
	if c := h.Catalog; c != nil {
		out.Catalog = &Catalog{
			Language:    c.Language,
			ID:          c.ID,
			Version:     c.Version,
			Name:        localizedFromV2005(c.Name),
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

func convertV2005Product(p *bmecat2005.Product) *Product {
	if p == nil {
		return nil
	}
	out := &Product{
		ID:              p.SupplierPID,
		Mode:            p.Mode,
		CatalogGroupIDs: p.CatalogGroupIDs,
	}
	if d := p.Details; d != nil {
		out.GTIN = gtinFromV2005(d)
		for _, pid := range d.InternationalPIDs {
			if pid != nil {
				out.PIDs = append(out.PIDs, &TypedValue{Type: pid.Type, Value: pid.Value})
			}
		}
		out.DescriptionShort = localizedFromV2005(d.DescriptionShort)
		out.DescriptionLong = localizedFromV2005(d.DescriptionLong)
		out.SupplierAltID = d.SupplierAltPID
		out.ManufacturerID = d.ManufacturerPID
		out.ManufacturerName = d.ManufacturerName
		out.ManufacturerTypeDescr = localizedFromV2005(d.ManufacturerTypeDescr)
		out.ERPGroupBuyer = d.ERPGroupBuyer
		out.ERPGroupSupplier = d.ERPGroupSupplier
		out.DeliveryTime = d.DeliveryTime
		out.Keywords = localizedFromV2005(d.Keywords)
		out.Remarks = localizedFromV2005(d.Remarks)
		out.Segments = localizedFromV2005(d.Segments)
		for _, b := range d.BuyerPIDs {
			if b != nil {
				out.BuyerIDs = append(out.BuyerIDs, &TypedValue{Type: b.Type, Value: b.Value})
			}
		}
		for _, s := range d.SpecialTreatmentClasses {
			if s != nil {
				out.SpecialTreatmentClasses = append(out.SpecialTreatmentClasses, &TypedValue{Type: s.Type, Value: s.Value})
			}
		}
		for _, s := range d.ProductStatus {
			if s != nil {
				out.Status = append(out.Status, &TypedValue{Type: s.Type, Value: s.Value})
			}
		}
	}
	if od := p.OrderDetails; od != nil {
		out.OrderUnit = od.OrderUnit
		out.ContentUnit = od.ContentUnit
		out.NoCuPerOu = od.NoCuPerOu
		out.PriceQuantity = od.PriceQuantity
		out.QuantityMin = od.QuantityMin
		out.QuantityInterval = od.QuantityInterval
		out.QuantityMax = od.QuantityMax
	}
	out.UDX = convertV2005UDX(p.UDX)
	for _, f := range p.Features {
		out.Features = append(out.Features, convertV2005Features(f))
	}
	for _, pd := range p.PriceDetails {
		npd := convertV2005PriceDetails(pd)
		if npd == nil {
			continue
		}
		out.PriceDetails = append(out.PriceDetails, npd)
		out.Prices = append(out.Prices, npd.Prices...)
	}
	if mi := p.MimeInfo; mi != nil {
		for _, m := range mi.Mimes {
			out.Mimes = append(out.Mimes, &Mime{
				Type:    m.Type,
				Source:  localizedFromV2005(m.Source),
				Descr:   localizedFromV2005(m.Descr),
				Purpose: m.Purpose,
				Order:   m.Order,
			})
		}
	}
	return out
}

// localizedFromV2005 converts a bmecat2005 LocalizedStrings into the neutral
// one, preserving variant order and language.
func localizedFromV2005(in bmecat2005.LocalizedStrings) LocalizedStrings {
	if len(in) == 0 {
		return nil
	}
	out := make(LocalizedStrings, len(in))
	for i, ls := range in {
		out[i] = LocalizedString{Lang: ls.Lang, Value: ls.Value}
	}
	return out
}

func convertV2005PriceDetails(pd *bmecat2005.ProductPriceDetails) *PriceDetails {
	if pd == nil {
		return nil
	}
	out := &PriceDetails{
		ValidStart:   v2005ValidDate(pd.Dates, bmecat2005.DateTimeValidStartDate),
		ValidEnd:     v2005ValidDate(pd.Dates, bmecat2005.DateTimeValidEndDate),
		IsDailyPrice: pd.IsDailyPrice(),
	}
	for _, pr := range pd.Prices {
		if pr == nil {
			continue
		}
		out.Prices = append(out.Prices, &Price{
			Type:       pr.Type,
			Amount:     pr.Amount,
			Currency:   pr.Currency,
			Tax:        taxFromV2005Price(pr),
			Factor:     pr.Factor,
			LowerBound: pr.LowerBound,
			Territory:  pr.Territory,
		})
	}
	return out
}

// v2005ValidDate returns the parsed value of the first DATETIME of the given
// type, or nil when it is absent or unparseable. Unlike ProductPriceDetails'
// ValidStartDate/ValidEndDate, it does not substitute a sentinel default, so the
// neutral model can distinguish "no date" from a real bound.
func v2005ValidDate(dates []*bmecat2005.DateTime, typ string) *time.Time {
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

// taxFromV2005Price returns the tax rate for a 2005 price: the bare TAX
// element, falling back to the first TAX_DETAILS/TAX rate. New 2005 documents
// carry tax under TAX_DETAILS, so without this fallback the neutral Tax would
// be lost for them.
func taxFromV2005Price(pr *bmecat2005.ProductPrice) *float64 {
	if pr.Tax != nil {
		return pr.Tax
	}
	for _, td := range pr.TaxDetails {
		if td != nil && td.Tax != nil {
			return td.Tax
		}
	}
	return nil
}

// gtinFromV2005 returns the canonical GTIN/EAN for a 2005 product. It prefers
// an INTERNATIONAL_PID explicitly typed gtin or ean, then falls back to the
// first non-empty PID of any type (an untyped PID is almost always the GTIN),
// and finally to the legacy EAN element. This stops a leading non-GTIN PID
// (e.g. type="supplier_specific") from shadowing a correctly-typed one.
func gtinFromV2005(d *bmecat2005.ProductDetails) string {
	var first string
	for _, pid := range d.InternationalPIDs {
		if pid == nil || pid.Value == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(pid.Type)) {
		case "gtin", "ean":
			return pid.Value
		}
		if first == "" {
			first = pid.Value
		}
	}
	if first != "" {
		return first
	}
	return d.EAN
}

func convertV2005Features(f *bmecat2005.ProductFeatures) *Features {
	if f == nil {
		return nil
	}
	out := &Features{
		SystemName: f.FeatureSystemName,
		GroupID:    f.FeatureGroupID,
		GroupName:  localizedFromV2005(f.FeatureGroupName),
	}
	for _, ft := range f.Features {
		out.Features = append(out.Features, &Feature{
			Name:   localizedFromV2005(ft.Name),
			Values: localizedFromV2005(ft.Values),
			Unit:   ft.Unit,
		})
	}
	return out
}

func convertV2005UDX(udx *bmecat2005.UserDefinedExtensions) []*UDXField {
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
