package bmecat2005

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"sync/atomic"
)

type Transaction byte

const (
	NewCatalog Transaction = iota
	UpdateProducts
	UpdatePrices
)

func (t Transaction) String() string {
	switch t {
	default:
		return "T_NEW_CATALOG"
	case UpdateProducts:
		return "T_UPDATE_PRODUCTS"
	case UpdatePrices:
		return "T_UPDATE_PRICES"
	}
}

// xmlNamespace is the namespace used for all BMEcat 2005 documents. Unlike
// 1.2, which uses a different namespace per transaction, the 2005 DTDs fix a
// single namespace regardless of the transaction.
const xmlNamespace = "http://www.bmecat.org/bmecat/2005"

// dtd returns the SYSTEM identifier of the DTD for the given transaction, as
// shipped in the official BMEcat 2005 DTD package.
func (t Transaction) dtd() string {
	switch t {
	case UpdateProducts:
		return "bmecat_2005_tupro.dtd"
	case UpdatePrices:
		return "bmecat_2005_tupri.dtd"
	default:
		return "bmecat_2005_tnc.dtd"
	}
}

// CatalogWriter specifies the contract that users of Writer have to
// implement to allow writing a BMEcat file.
//
// Implement Products with the StreamProducts helper to stream products from a
// pull-style producer and avoid hand-rolling the channel bookkeeping.
type CatalogWriter interface {
	Transaction() Transaction
	Language() string
	PreviousVersion() int
	Header() *Header
	ClassificationSystem() *ClassificationSystem
	Products(context.Context) (<-chan *Product, <-chan error)
}

// Writer allows writing BMEcat 2005 catalog files.
type Writer struct {
	w        io.Writer
	progress WriteProgress
	enc      *xml.Encoder

	// indent setting for the writer.
	indent string
	// Transaction specifies the mode of the catalog, e.g. "T_NEW_CATALOG" (default),
	// "T_UPDATE_PRODUCTS", or "T_UPDATE_PRICES".
	transaction Transaction
}

// NewWriter creates a new Writer. It expects an underlying io.Writer
// which essentially gets the XML content. You can also pass additional
// options like WithProgress.
func NewWriter(w io.Writer, options ...WriterOption) *Writer {
	writer := &Writer{w: w, indent: "  ", transaction: NewCatalog}
	for _, o := range options {
		o(writer)
	}
	return writer
}

// WriterOption is the signature of options to pass into a NewWriter.
type WriterOption func(*Writer)

// WithIndent sets the indentation for writing the XML file. It is set to two spaces by default.
func WithIndent(indent string) WriterOption {
	return func(w *Writer) {
		w.indent = indent
	}
}

// WithProgress reports the current number of products as they are written.
func WithProgress(f WriteProgress) WriterOption {
	return func(w *Writer) {
		w.progress = f
	}
}

// WriteProgress is the signature of the progress callback while writing.
// You can tell the Writer to report progress with the WithProgress option.
type WriteProgress func(written int)

// txStartElement returns the XML StartElement for the BMEcat transaction,
// e.g. "T_NEW_CATALOG".
func (w *Writer) txStartElement(writer CatalogWriter) xml.StartElement {
	tx := writer.Transaction().String()
	attr := []xml.Attr{}
	// Both update transactions carry prev_version; a new catalog does not.
	switch writer.Transaction() {
	case UpdateProducts, UpdatePrices:
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "prev_version"}, Value: fmt.Sprint(writer.PreviousVersion())})
	}
	return xml.StartElement{Name: xml.Name{Local: tx}, Attr: attr}
}

// txEndElement returns the XML EndElement for the BMEcat transaction,
// e.g. "T_NEW_CATALOG".
func (w *Writer) txEndElement(writer CatalogWriter) xml.EndElement {
	return xml.EndElement{Name: xml.Name{Local: writer.Transaction().String()}}
}

// Do writes the BMEcat file.
//
// You must pass a context, which can be canceled to stop writing.
//
// The products to write are streamed from the CatalogWriter's Products
// method; once that channel is closed, Do writes the rest of the BMEcat
// file and returns.
func (w *Writer) Do(ctx context.Context, writer CatalogWriter) error {
	w.enc = xml.NewEncoder(w.w)
	if w.indent != "" {
		w.enc.Indent("", w.indent)
	}
	if err := w.writeLeadIn(writer); err != nil {
		return fmt.Errorf("bmecat/2005: unable to write lead in: %w", err)
	}
	header := writer.Header()
	if header != nil {
		if err := w.enc.Encode(header); err != nil {
			return fmt.Errorf("bmecat/2005: unable to write Header: %w", err)
		}
	}
	tx := writer.Transaction().String()
	if err := w.enc.EncodeToken(w.txStartElement(writer)); err != nil {
		return fmt.Errorf("bmecat/2005: unable to write opening %s: %w", tx, err)
	}

	if writer.Transaction() == NewCatalog {
		// FEATURE_SYSTEM

		// CLASSIFICATION_SYSTEM
		if system := writer.ClassificationSystem(); system != nil {
			if !system.IsBlank() {
				if err := w.enc.Encode(system); err != nil {
					return fmt.Errorf("bmecat/2005: unable to write CLASSIFICATION_SYSTEM: %w", err)
				}
			}
		}

		// CATALOG_GROUP_SYSTEM
	}

	// PRODUCT
	if err := w.writeProducts(ctx, writer); err != nil {
		return fmt.Errorf("bmecat/2005: unable to write PRODUCT: %w", err)
	}

	if writer.Transaction() != UpdatePrices {
		// PRODUCT_TO_CATALOGGROUP_MAP
	}

	if err := w.enc.EncodeToken(w.txEndElement(writer)); err != nil {
		return fmt.Errorf("bmecat/2005: unable to write closing %s: %w", tx, err)
	}
	if err := w.writeLeadOut(); err != nil {
		return fmt.Errorf("bmecat/2005: unable to write lead out: %w", err)
	}
	return w.enc.Flush()
}

func (w *Writer) writeLeadIn(writer CatalogWriter) error {
	_, err := fmt.Fprint(w.w, xml.Header)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w.w, "<!DOCTYPE BMECAT SYSTEM %q>\n", writer.Transaction().dtd())
	if err != nil {
		return err
	}
	// <BMECAT version="2005" xmlns="http://www.bmecat.org/bmecat/2005">
	attr := []xml.Attr{
		{Name: xml.Name{Local: "xmlns"}, Value: xmlNamespace},
		{Name: xml.Name{Local: "version"}, Value: "2005"},
	}
	t := xml.StartElement{
		Name: xml.Name{Local: "BMECAT"},
		Attr: attr,
	}
	return w.enc.EncodeToken(t)
}

func (w *Writer) writeLeadOut() error {
	return w.enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "BMECAT"}})
}

func (w *Writer) writeProducts(ctx context.Context, writer CatalogWriter) error {
	productsCh, errCh := writer.Products(ctx)
	if productsCh == nil {
		return nil
	}

	var stop bool
	var written atomic.Uint32
	for !stop {
		select {
		case p, ok := <-productsCh:
			if !ok {
				stop = true
				break
			}
			if err := w.writeProduct(p); err != nil {
				return fmt.Errorf("unable to write SUPPLIER_PID %q: %w", p.SupplierPID, err)
			}
			current := written.Add(1)
			if w.progress != nil {
				w.progress(int(current))
			}
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func (w *Writer) writeProduct(p *Product) error {
	// TODO(oe) Only serialize the part of the product that is required by w.Transaction
	err := w.enc.Encode(p)
	if err != nil {
		return err
	}
	return nil
}
