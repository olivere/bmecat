# BMEcat for Go

[![Test](https://github.com/olivere/bmecat/actions/workflows/test.yaml/badge.svg)](https://github.com/olivere/bmecat/actions/workflows/test.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/olivere/bmecat.svg)](https://pkg.go.dev/github.com/olivere/bmecat)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A small, dependency-light library for reading and writing
[BMEcat](https://www.bmecat.org/) electronic product catalogs in Go.
It currently supports BMEcat version 1.2 via the `bmecat12` package.

Reading is streaming and handler-based, so large catalogs can be processed
without loading the whole document into memory.

## Requirements

Go 1.25 or later.

## Installation

```sh
go get github.com/olivere/bmecat
```

## Reading a catalog

Implement only the handler interfaces you care about — `HeaderHandler`,
`ArticleHandler`, `CatalogGroupHandler`, `ClassificationGroupHandler` and/or
`CompletionHandler` — and pass your handler to `Reader.Do`:

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
	fmt.Printf("Article %s: %s\n", a.SupplierAID, a.Details.DescriptionShort)
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

Implement the `bmecat12.CatalogWriter` interface and hand it to `Writer.Do`.
See the [package documentation](https://pkg.go.dev/github.com/olivere/bmecat/bmecat12)
and the writer tests for complete, runnable examples.

## Command-line tool

The `cmd/bmecat` package contains a small CLI that demonstrates the library,
e.g. printing the header of a catalog:

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

This library implements the subset of **BMEcat 1.2** needed to read and write
the common `T_NEW_CATALOG`, `T_UPDATE_PRODUCTS` and `T_UPDATE_PRICES`
transactions.

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
