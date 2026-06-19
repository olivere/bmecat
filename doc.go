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
INTERNATIONAL_PID element behind a single accessor.

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

Callers that need raw, version-specific fidelity (including 2005-only fields
such as PRODUCT_LOGISTIC_DETAILS, or writing catalogs) can use the
github.com/olivere/bmecat/bmecat12 and github.com/olivere/bmecat/bmecat2005
packages directly.
*/
package bmecat
