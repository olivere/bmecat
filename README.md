# BMEcat for Go

[![Test](https://github.com/olivere/bmecat/actions/workflows/test.yaml/badge.svg)](https://github.com/olivere/bmecat/actions/workflows/test.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/olivere/bmecat.svg)](https://pkg.go.dev/github.com/olivere/bmecat)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A small, dependency-light library for reading and writing
[BMEcat](https://www.bmecat.org/) electronic product catalogs in Go.
It supports BMEcat version 1.2 via the `bmecat12` package and BMEcat 2005
(2.0) via the `bmecat2005` package. The two packages share the same
streaming, handler-based API, so they are learnable as one.

Reading is streaming and handler-based, so large catalogs can be processed
without loading the whole document into memory.

## Requirements

Go 1.25 or later.

## Installation

```sh
go get github.com/olivere/bmecat
```

## Reading any version (recommended)

If you ingest catalogs from a mix of suppliers, use the top-level `bmecat`
package. `bmecat.NewReader` auto-detects the version from the root
`<BMECAT version="…">` element and normalises both 1.2 and 2005 into a single,
version-neutral model — so you write your mapping once. Implement only the
neutral handler interfaces you care about (`HeaderHandler`, `ProductHandler`,
`CatalogGroupHandler`, `ClassificationGroupHandler` and/or `CompletionHandler`):

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/olivere/bmecat"
)

// handler works unchanged for BMEcat 1.2 and 2005 input.
type handler struct{}

func (handler) HandleHeader(h *bmecat.Header) error {
	fmt.Printf("Catalog %q (BMEcat %s) with %d product(s)\n",
		h.Catalog.Name, h.Version, h.NumberOfProducts)
	return nil
}

func (handler) HandleProduct(p *bmecat.Product) error {
	fmt.Printf("Product %s: %s (GTIN %s)\n", p.ID, p.DescriptionShort.Value(), p.GTIN)
	return nil
}

func main() {
	f, err := os.Open("catalog.xml")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err := bmecat.NewReader(f).Do(context.Background(), handler{}); err != nil {
		log.Fatal(err)
	}
}
```

The neutral model exposes the fields 1.2 and 2005 share; in particular
`Product.GTIN` unifies the 1.2 `EAN` and 2005 `INTERNATIONAL_PID` elements.

BMEcat 2005 lets many text elements repeat once per language (the spec's
`dtMLSTRING` type). Those fields are exposed as `LocalizedStrings`, preserving
every variant in document order, so a consumer can pick the one matching a
configured language with `p.DescriptionShort.Get("eng")` (which falls back to the
first variant when the language is absent) or take the first with
`p.DescriptionShort.Value()`. For elements that legitimately repeat (`KEYWORD`,
`SEGMENT`, feature values), `p.Keywords.All("eng")` returns every value for a
language. Build the common single-language value with `bmecat.Localized("Widget")`
(it is variadic: `bmecat.Localized("a", "b")` builds a list). Every element the
2005 schema types as `dtMLSTRING` is covered — on the neutral model the catalog
name; product short/long description, manufacturer type description, keywords,
remarks and segments; feature group name, feature names and values; MIME source
and description — and, in the `bmecat2005` package, additionally address parts
(name, street, city, …), MIME alt, feature descriptions/value-details/variant
values, catalog-group keywords, classification-group synonyms and classification
system level names. Identifiers and codes the schema types as plain strings (e.g.
`MANUFACTURER_NAME`, `SUPPLIER_NAME`, `FUNIT`, the classification system name,
`EMAIL`/`URL`) stay scalar.

BMEcat 1.2 has no per-element `lang` attribute — it declares one catalog language
in the header — so the `bmecat12` structs keep plain `string`/`[]string` fields.
Reading a 1.2 document fills each `LocalizedStrings` with a single language-less
variant; writing the neutral model to 1.2 emits the variant matching the
catalog's language (falling back to the first), which is lossy for genuinely
multi-language data. The `lang` attribute is written (in 2005) only for variants
that set a language, so single-language catalogs round-trip unchanged.

This replaced the former scalar fields: update reads such as `p.DescriptionShort`
to `p.DescriptionShort.Value()` (or `.Get(lang)` / `.All(lang)`).
Prices come both flattened (`Product.Prices`) and grouped by their
`ARTICLE_PRICE_DETAILS` / `PRODUCT_PRICE_DETAILS` wrapper with validity dates
(`Product.PriceDetails`), so you can pick the currently-valid block or spot a
price calendar. Runnable examples are in the
[package documentation](https://pkg.go.dev/github.com/olivere/bmecat#pkg-examples).

To gate on the document-level transaction — for example to accept only full
catalogs and reject incremental updates — call `DetectTransaction` before `Do`
(it rewinds, like `DetectVersion`); the same value is also available as
`Header.Transaction` during a full parse:

```go
r := bmecat.NewReader(f)
if tx, err := r.DetectTransaction(); err != nil {
	return err
} else if tx.IsUpdate() {
	return fmt.Errorf("only full catalogs are supported, got %s", tx)
}
```

## Reading a specific version

To work with a single version directly — for raw fidelity, version-specific
fields (e.g. 2005's `PRODUCT_LOGISTIC_DETAILS`), or writing — use the
`bmecat12` or `bmecat2005` package. Both share the same shape: implement only
the handler interfaces you care about — `HeaderHandler`, `ArticleHandler`
(1.2) / `ProductHandler` (2005), `CatalogGroupHandler`,
`ClassificationGroupHandler` and/or `CompletionHandler` — and pass your handler
to `Reader.Do`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/olivere/bmecat/bmecat12"
)

// handler reacts to the parts of the catalog it implements a method for.
type handler struct{}

func (handler) HandleHeader(h *bmecat12.Header) error {
	fmt.Printf("Catalog %q with %d article(s)\n", h.Catalog.Name, h.NumberOfArticles)
	return nil
}

func (handler) HandleArticle(a *bmecat12.Article) error {
	fmt.Printf("Article %s: %s\n", a.SupplierAID, a.Details.DescriptionShort.Value())
	return nil
}

func main() {
	f, err := os.Open("catalog.xml")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	r := bmecat12.NewReader(f)
	if err := r.Do(context.Background(), handler{}); err != nil {
		log.Fatal(err)
	}
}
```

### Character encodings

The reader decodes any encoding declared in the XML prolog that is registered
with the IANA character set registry and implemented by `golang.org/x/text` —
UTF-8, the ISO-8859 family, Windows code pages, and CJK encodings such as GBK,
GB18030, Big5, Shift-JIS and EUC-KR. ISO-8859-1 is decoded leniently as
Windows-1252, which matches how most real-world catalogs are authored.

## Writing a catalog

### Version-neutral writing

`bmecat.NewWriter` is the streaming, write-path counterpart of
`bmecat.NewReader`: implement a neutral `CatalogWriter` (a header plus a channel
of products), pick a target version with `WithVersion`, and the writer emits a
valid BMEcat 1.2 or 2005 document, converting the neutral model to the
version-specific one for you. Like reading, writing is streaming — each product
is converted and encoded as it arrives, so even a catalog of millions of
products is never held in memory at once.

```go
// catalog implements bmecat.CatalogWriter.
type catalog struct{ /* e.g. a database handle */ }

func (catalog) Header() *bmecat.Header { return header }

func (c catalog) Products(ctx context.Context) (<-chan *bmecat.Product, <-chan error) {
	out := make(chan *bmecat.Product)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		for _, p := range c.stream() { // e.g. rows from a database cursor
			select {
			case out <- p:
			case <-ctx.Done():
				errc <- ctx.Err()
				return
			}
		}
	}()
	return out, errc
}

// ...
w := bmecat.NewWriter(out, bmecat.WithVersion(bmecat.Version2005))
if err := w.Do(context.Background(), catalog{}); err != nil {
	return err
}
```

The neutral model carries the fields 1.2 and 2005 have in common, so the output
covers those; version-specific fidelity needs the version packages below. The
neutral writer also does not emit catalog-group mappings (neither version writer
does).

### Version-specific writing

For full, version-specific fidelity, implement the `CatalogWriter` interface of
the target package (`bmecat12` or `bmecat2005`) and hand it to that package's
`Writer.Do`. See the package documentation
([bmecat12](https://pkg.go.dev/github.com/olivere/bmecat/bmecat12),
[bmecat2005](https://pkg.go.dev/github.com/olivere/bmecat/bmecat2005)) and the
writer tests for complete, runnable examples.

## Command-line tool

The `cmd/bmecat` package contains a small CLI that demonstrates the library,
e.g. printing the header of a catalog. It reads both BMEcat 1.2 and 2005 files,
auto-detecting the version:

```sh
go run ./cmd/bmecat info path/to/catalog.xml
```

## BMEcat

BMEcat is a standard for electronic product catalogs.

### Specifications

The specifications are published by the German
[BME](https://www.bme.de/) (Bundesverband Materialwirtschaft, Einkauf und
Logistik e.V.). They are **not redistributed in this repository** because they
are copyrighted by the BME; obtain them from the authoritative sources below:

- [bmecat.org](https://www.bmecat.org/) — overview of the format
- [BME BMEcat initiative](https://www.bme.de/initiativen/bmecat/) — the
  publisher's home for the standard
- [BMEcat downloads](https://www.bme.de/initiativen/bmecat/download/) — where to
  obtain the official specification documents and DTDs

This library implements the subset of **BMEcat 1.2** (`bmecat12`) needed to
read and write the common `T_NEW_CATALOG`, `T_UPDATE_PRODUCTS` and
`T_UPDATE_PRICES` transactions, and a mirror of that surface for **BMEcat
2005** (`bmecat2005`). BMEcat 2005 is mostly a set of element renames over
1.2 (`ARTICLE` becomes `PRODUCT`, `EAN` becomes `INTERNATIONAL_PID`,
`MANUFACTURER_AID` becomes `MANUFACTURER_PID`, and so on); the `bmecat2005`
package also models a few common 2005-only additions such as
`INTERNATIONAL_PID` type qualifiers and `PRODUCT_LOGISTIC_DETAILS`. Writing
currently targets `T_NEW_CATALOG`.

## Development

Common tasks are available through the `Makefile`. Formatting
([gofumpt](https://github.com/mvdan/gofumpt)) and vulnerability scanning
([govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)) are
managed as Go tool dependencies, so no separate installation is needed.

```sh
make build       # build the CLI
make test        # run tests with the race detector
make fmt         # format the code with gofumpt
make lint        # check formatting and run go vet
make vulncheck   # scan for known vulnerabilities
make modernize   # apply modern Go idioms
make check       # lint + test + vulncheck
```

## License

MIT. See the [LICENSE](LICENSE) file.
