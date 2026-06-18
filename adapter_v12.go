package bmecat

import (
	"io"

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

	// headerErr retains a non-EOF error returned by the neutral HeaderHandler.
	// The bmecat12 reader currently swallows such errors (see issue #16), so
	// the facade surfaces it after Do returns to honor the HeaderHandler
	// contract.
	headerErr error
}

func newV12Adapter(handler any) *v12Adapter {
	a := &v12Adapter{}
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
	err := a.header.HandleHeader(convertV12Header(h))
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
		Name:        cg.Name,
		Description: cg.Description,
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
		Name:        cg.Name,
		Description: cg.Description,
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
			Name:        c.Name,
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
		out.DescriptionShort = d.DescriptionShort
		out.DescriptionLong = d.DescriptionLong
		out.SupplierAltID = d.SupplierAltAID
		out.ManufacturerID = d.ManufacturerAID
		out.ManufacturerName = d.ManufacturerName
		out.Keywords = d.Keywords
	}
	if od := a.OrderDetails; od != nil {
		out.OrderUnit = od.OrderUnit
	}
	for _, f := range a.Features {
		out.Features = append(out.Features, convertV12Features(f))
	}
	for _, pd := range a.PriceDetails {
		for _, p := range pd.Prices {
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
	}
	if mi := a.MimeInfo; mi != nil {
		for _, m := range mi.Mimes {
			out.Mimes = append(out.Mimes, &Mime{
				Type:    m.Type,
				Source:  m.Source,
				Descr:   m.Descr,
				Purpose: m.Purpose,
				Order:   m.Order,
			})
		}
	}
	return out
}

func convertV12Features(f *bmecat12.ArticleFeatures) *Features {
	if f == nil {
		return nil
	}
	out := &Features{
		SystemName: f.FeatureSystemName,
		GroupID:    f.FeatureGroupID,
		GroupName:  f.FeatureGroupName,
	}
	for _, ft := range f.Features {
		out.Features = append(out.Features, &Feature{
			Name:   ft.Name,
			Values: ft.Values,
			Unit:   ft.Unit,
		})
	}
	return out
}
