/*
Package bmecat supports reading BMEcat files of either supported version
through a single, version-neutral API.

BMEcat is a standard for electronic product catalogs.
See http://www.bmecat.org/ for details about the format.
The specifications are available at
http://www.bme.de/initiativen/bmecat/download/.

# Version-neutral reading

[Reader] auto-detects the version from the root <BMECAT version="…"> element
and dispatches to the bmecat12 or bmecat2005 reader, normalizing both into the
version-neutral types in this package. A caller implements the handler
interfaces it cares about ([HeaderHandler], [ProductHandler],
[CatalogGroupHandler], [ClassificationGroupHandler], [CompletionHandler]) once,
and ingests 1.2 and 2005 catalogs through the same code path:

	r := bmecat.NewReader(file)
	err := r.Do(ctx, handler)

The neutral model exposes the fields the two versions have in common. In
particular [Product.GTIN] unifies the 1.2 EAN element and the 2005
INTERNATIONAL_PID element behind a single accessor. For 2005 documents that
carry more than one INTERNATIONAL_PID, [Product.PIDs] preserves every typed
identifier so a read-modify-write keeps them all.

BMEcat 2005 lets many text elements repeat once per language (the spec's
dtMLSTRING type). Those neutral fields are [LocalizedStrings], keeping every
variant in document order: [LocalizedStrings.Get] picks one by language (falling
back to the first variant), [LocalizedStrings.Value] returns the first,
[LocalizedStrings.All] returns every value for a language (for elements that
repeat, such as KEYWORD), and [Localized] builds the common single-language
value. Every element the 2005 schema types as dtMLSTRING is covered: on the
neutral model the catalog name; product short/long description, manufacturer type
description, keywords, remarks and segments; feature group name, feature names
and values; MIME source and description — and, in the bmecat2005 package,
additionally address parts, MIME alt, feature descriptions/value-details/variant
values, catalog-group keywords, classification-group synonyms and classification
system level names. Identifiers and codes typed as plain strings (e.g.
MANUFACTURER_NAME, FUNIT, the classification system name) stay scalar.

BMEcat 1.2 has no per-element lang attribute, so the bmecat12 structs stay
scalar: reading 1.2 yields a single language-less variant, and writing the
neutral model to 1.2 emits the variant matching the catalog language (falling
back to the first), which is lossy for multi-language data. The lang attribute is
written (in 2005) only for variants that set a language, so single-language
catalogs round-trip unchanged.

Prices are exposed two ways. [Product.Prices] is a flat list of every price
block, convenient when the grouping does not matter. [Product.PriceDetails]
preserves the ARTICLE_PRICE_DETAILS / PRODUCT_PRICE_DETAILS wrapper grouping,
including each wrapper's validity dates ([PriceDetails.ValidStart] /
[PriceDetails.ValidEnd], nil when the source omits them) — so a consumer can
pick the currently-valid block or detect a price calendar (several dated
wrappers). [Product.CurrentPriceDetails] selects the wrapper valid at a given
time, [Product.ValidPriceDetails] returns every match, and
[PriceDetails.PriceFor] resolves the graduated/scale tier for an order
quantity.

A caller that needs to gate on the document-level transaction — for example to
reject incremental updates and only accept full catalogs — can detect it cheaply
in phase 1, mirroring [Reader.DetectVersion]:

	switch tx, err := r.DetectTransaction(); {
	case err != nil:
		return err
	case tx.IsUpdate():
		return fmt.Errorf("only full catalogs are supported, got %s", tx)
	}

The same value is also surfaced on [Header.Transaction] during Do, for callers
that read it as part of a full parse.

# Version-neutral writing

[Writer] is the streaming, write-path counterpart of [Reader]: a caller
implements a [CatalogWriter] (a header plus a channel of products), selects a
target version with [WithVersion], and Writer emits a valid BMEcat 1.2 or 2005
document, converting the neutral model to the version-specific one at the
boundary:

	w := bmecat.NewWriter(out, bmecat.WithVersion(bmecat.Version2005))
	err := w.Do(ctx, catalog) // catalog implements bmecat.CatalogWriter

For most callers, [Writer.WriteFunc] is the ergonomic default: it takes a header
and a pull-style producer that streams products by calling yield, keeping the
streaming property while removing the channel bookkeeping the [CatalogWriter]
interface requires:

	err := w.WriteFunc(ctx, header, func(yield func(*bmecat.Product) error) error {
		for rows.Next() {
			if err := yield(buildProduct(rows)); err != nil {
				return err
			}
		}
		return rows.Err()
	})

As with reading, writing is streaming — each product is converted and encoded as
it arrives, so even a very large catalog is never held in memory at once — and
the neutral model carries only the fields the two versions share, so the output
covers those common fields.

Callers that need raw, version-specific fidelity (including 2005-only fields
such as PRODUCT_LOGISTIC_DETAILS, or writing version-specific elements) can use
the github.com/olivere/bmecat/bmecat12 and github.com/olivere/bmecat/bmecat2005
packages directly.
*/
package bmecat
