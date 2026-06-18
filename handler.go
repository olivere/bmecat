package bmecat

// The handler interfaces mirror those of the bmecat12 and bmecat2005 packages,
// but receive the version-neutral types. A handler passed to Reader.Do may
// implement any combination of them; only the implemented callbacks are
// invoked.

// HeaderHandler is called when the Reader passed the BMEcat HEADER element.
//
// HandleHeader may return io.EOF to stop the Reader from continuing to read.
// Any other error also stops the Reader and is returned from Reader.Do.
type HeaderHandler interface {
	HandleHeader(*Header) error
}

// CatalogGroupHandler is called for each CATALOG_STRUCTURE element.
type CatalogGroupHandler interface {
	HandleCatalogGroup(*CatalogGroup) error
}

// ClassificationGroupHandler is called for each CLASSIFICATION_GROUP element.
type ClassificationGroupHandler interface {
	HandleClassificationGroup(*ClassificationGroup) error
}

// ProductHandler is called for each product (ARTICLE in 1.2, PRODUCT in 2005).
type ProductHandler interface {
	HandleProduct(*Product) error
}

// CompletionHandler is called once when the Reader is done parsing.
type CompletionHandler interface {
	HandleComplete()
}
