# BMEcat for Go

[![Test](https://github.com/olivere/bmecat/actions/workflows/test.yaml/badge.svg)](https://github.com/olivere/bmecat/actions/workflows/test.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/olivere/bmecat.svg)](https://pkg.go.dev/github.com/olivere/bmecat)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A small, dependency-light library for reading and writing
[BMEcat](https://www.bmecat.org/) electronic product catalogs in Go, for
BMEcat 1.2 and BMEcat 2005. Reading and writing both stream, so a catalog of
millions of products is never held in memory at once.

The module has three packages:

- `bmecat` reads both versions into one version-neutral model and writes that
  model as either version. Start here; your mapping code then handles 1.2 and
  2005 alike.
- `bmecat12` and `bmecat2005` expose each version's elements as they appear in
  the document, including fields the other version lacks (such as 2005's
  `PRODUCT_LOGISTIC_DETAILS`). Use them when you need that fidelity.

The [package documentation](https://pkg.go.dev/github.com/olivere/bmecat)
covers the model in detail: GTIN and typed identifiers, multi-language text,
price blocks and their validity dates, transactions, classification systems
and character encodings.

## Requirements

Go 1.27.3 or later.

## Installation

```sh
go get github.com/olivere/bmecat
```

## Reading a catalog

Implement the handler interfaces you care about (`HeaderHandler`,
`ProductHandler`, `CatalogGroupHandler`, `ClassificationGroupHandler`,
`CompletionHandler`):

```go
// catalogPrinter implements the neutral handler interfaces it cares about.
// The same handler works for BMEcat 1.2 and 2005 input.
type catalogPrinter struct{}

// These assertions turn a misspelled handler method into a compile error.
var (
	_ bmecat.HeaderHandler  = catalogPrinter{}
	_ bmecat.ProductHandler = catalogPrinter{}
)

func (catalogPrinter) HandleHeader(h *bmecat.Header) error {
	fmt.Printf("Catalog %q (BMEcat %s)\n", h.Catalog.Name.Value(), h.Version)
	return nil
}

func (catalogPrinter) HandleProduct(p *bmecat.Product) error {
	fmt.Printf("- %s: %s (GTIN %s)\n", p.ID, p.DescriptionShort.Value(), p.GTIN)
	return nil
}
```

Then pass it to `Reader.Do`. The reader detects the version from the document:

```go
r := bmecat.NewReader(strings.NewReader(doc))
if err := r.Do(context.Background(), catalogPrinter{}); err != nil {
	panic(err)
}
```

## Writing a catalog

`Writer.WriteFunc` takes a header and a function that streams products by
calling `yield`. `WithVersion` picks the output version:

```go
var buf strings.Builder
w := bmecat.NewWriter(&buf, bmecat.WithVersion(bmecat.Version2005))
err := w.WriteFunc(context.Background(), header, func(yield func(*bmecat.Product) error) error {
	for _, p := range products {
		if err := yield(p); err != nil {
			return err
		}
	}
	return nil
})
if err != nil {
	panic(err)
}
```

The neutral model carries the fields the two versions share. The writer does
not emit catalog groups yet ([#42](https://github.com/olivere/bmecat/issues/42)).

## Version-specific packages

`bmecat12` and `bmecat2005` read the same way as the neutral package, through
handler interfaces passed to `Reader.Do`, but with their own types. A 1.2
handler, for example, receives each `ARTICLE`:

```go
func (articlePrinter) HandleArticle(a *bmecat12.Article) error {
	fmt.Printf("- %s: %s (EAN %s)\n", a.SupplierAID, a.Details.DescriptionShort, a.Details.EAN)
	return nil
}
```

Both packages have runnable examples in their documentation
([bmecat12](https://pkg.go.dev/github.com/olivere/bmecat/bmecat12),
[bmecat2005](https://pkg.go.dev/github.com/olivere/bmecat/bmecat2005)).

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
`INTERNATIONAL_PID` type qualifiers and `PRODUCT_LOGISTIC_DETAILS`.

## Development

Common tasks are available through the `Makefile`. Formatting
([gofumpt](https://github.com/mvdan/gofumpt)) and vulnerability scanning
([govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)) are
managed as Go tool dependencies, so no separate installation is needed.

```sh
make build       # build the CLI
make test        # run tests with the race detector
make fmt         # format the code with gofumpt
make lint        # check formatting, run go vet and staticcheck
make vulncheck   # scan for known vulnerabilities
make modernize   # apply modern Go idioms
make bench       # run the benchmarks
make check       # lint + test + vulncheck
```

### Benchmarks

The package ships read and write benchmarks for both BMEcat versions, streaming
a content-rich catalog through the neutral facade so the per-product conversion
path is what is measured. Run them with:

```sh
make bench                       # all benchmarks, six samples each
make bench BENCH=Write           # only benchmarks matching "Write"
make bench BENCHCOUNT=10         # more samples for a tighter estimate
```

To measure a change against a baseline, save a run before and after, then
compare them with [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
(managed as a Go tool dependency, so no separate install is needed):

```sh
# 1. Baseline: stash your change and record the current numbers.
git stash
make bench-save BENCHFILE=base.txt
git stash pop

# 2. Your change: record the new numbers.
make bench-save BENCHFILE=new.txt

# 3. Compare. benchstat reports the delta and whether it is significant.
make benchstat OLD=base.txt NEW=new.txt
```

benchstat needs at least six samples per benchmark for a confidence interval,
which is why `BENCHCOUNT` defaults to six. A `~` in the output means the change
was not statistically significant.

## License

MIT. See the [LICENSE](LICENSE) file.
