package bmecat

import (
	"context"
	"io"

	"github.com/olivere/bmecat/bmecat12"
	"github.com/olivere/bmecat/bmecat2005"
)

// CatalogWriter is the neutral source of a catalog to write. It mirrors the
// CatalogWriter interfaces of the bmecat12 and bmecat2005 packages, but deals
// in the version-neutral types: a caller implements it once and Writer emits
// either supported version.
//
// Products streams the catalog's products and is the reason writing is
// memory-flat: Writer converts and encodes each product as it arrives, so even
// a catalog of millions of products is never held in memory at once.
//
// The implementation returns a products channel and an error channel. It must:
//
//   - close the products channel when all products have been sent;
//   - to report a failure, send a single non-nil error on the error channel
//     (which must be buffered, e.g. make(chan error, 1)) before closing the
//     products channel, then stop;
//   - select on ctx.Done() while sending, so a canceled context (Writer cancels
//     it when Do returns) unblocks the producer instead of leaking it.
//
// Sending more than one error, or sending the error only after closing the
// products channel, is a contract violation and may cause the error to be lost.
type CatalogWriter interface {
	// Header returns the catalog header, or nil to omit the HEADER element.
	Header() *Header
	// Products streams the catalog's products. See the type comment for the
	// channel contract.
	Products(ctx context.Context) (<-chan *Product, <-chan error)
}

// Writer writes a neutral catalog (CatalogWriter) as a BMEcat document of a
// chosen version. It is the streaming, write-path counterpart of Reader: a
// caller assembles the neutral types once, picks a target version with
// WithVersion, and Writer emits a valid BMEcat 1.2 or 2005 document, converting
// the neutral model to the version-specific one at the boundary.
//
// Because the neutral model exposes only the fields 1.2 and 2005 have in common,
// the output carries those common fields; version-specific fidelity (e.g. 2005
// PRODUCT_LOGISTIC_DETAILS) requires the bmecat12 / bmecat2005 packages
// directly. The writer also does not emit catalog-group mappings, mirroring the
// version writers, which do not write them either.
type Writer struct {
	w           io.Writer
	version     Version
	transaction Transaction
	prevVersion int
	indent      string
}

// NewWriter creates a Writer over w. By default it emits BMEcat 1.2 with a
// T_NEW_CATALOG transaction and two-space indentation; override with
// WithVersion, WithTransaction, WithPreviousVersion and WithIndent.
func NewWriter(w io.Writer, options ...WriterOption) *Writer {
	writer := &Writer{
		w:           w,
		version:     Version12,
		transaction: NewCatalog,
		indent:      "  ",
	}
	for _, o := range options {
		o(writer)
	}
	return writer
}

// WriterOption is the signature of options to pass into NewWriter.
type WriterOption func(*Writer)

// WithVersion selects the BMEcat version to emit. It defaults to Version12.
func WithVersion(v Version) WriterOption {
	return func(w *Writer) {
		w.version = v
	}
}

// WithTransaction selects the document-level transaction to emit. It defaults
// to NewCatalog.
func WithTransaction(t Transaction) WriterOption {
	return func(w *Writer) {
		w.transaction = t
	}
}

// WithPreviousVersion sets the prev_version attribute written on update
// transactions (T_UPDATE_PRODUCTS / T_UPDATE_PRICES). It is ignored for
// NewCatalog.
func WithPreviousVersion(v int) WriterOption {
	return func(w *Writer) {
		w.prevVersion = v
	}
}

// WithIndent sets the indentation of the emitted XML. It defaults to two
// spaces; pass the empty string to disable indentation.
func WithIndent(indent string) WriterOption {
	return func(w *Writer) {
		w.indent = indent
	}
}

// Do writes the neutral catalog as a BMEcat document of the configured version.
// It streams products from cw.Products and returns the first error encountered,
// from the producer or from encoding.
func (w *Writer) Do(ctx context.Context, cw CatalogWriter) error {
	// A cancelable context bounds the producer and the conversion bridge: if Do
	// returns early (e.g. an encoding error), canceling unblocks them so no
	// goroutine is left waiting on a send.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	switch w.version {
	case Version2005:
		return w.writeV2005(ctx, cw)
	default:
		return w.writeV12(ctx, cw)
	}
}

func (w *Writer) writeV12(ctx context.Context, cw CatalogWriter) error {
	adapter := &v12CatalogWriter{
		tx:          transactionToV12(w.transaction),
		prevVersion: w.prevVersion,
		neutral:     cw,
	}
	return bmecat12.NewWriter(w.w, bmecat12.WithIndent(w.indent)).Do(ctx, adapter)
}

func (w *Writer) writeV2005(ctx context.Context, cw CatalogWriter) error {
	adapter := &v2005CatalogWriter{
		tx:          transactionToV2005(w.transaction),
		prevVersion: w.prevVersion,
		neutral:     cw,
	}
	return bmecat2005.NewWriter(w.w, bmecat2005.WithIndent(w.indent)).Do(ctx, adapter)
}
