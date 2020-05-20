package bmecat12

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/pkg/errors"
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

// CatalogWriter specifies the contract that users of Writer have to
// implement to allow writing a BMEcat file.
type CatalogWriter interface {
	Transaction() Transaction
	Language() string
	PreviousVersion() int
	Header() *Header
	ClassificationSystem() *ClassificationSystem
	Articles(context.Context) (<-chan *Article, <-chan error)
}

// Optionally implement this interface on your CatalogWriter
// if you want to write a CATALOG_GROUP_SYSTEM.
type CatalogGroupSystemWriter interface {
	GroupSystem() *GroupSystem
}

// Writer allows writing BMEcat 1.2 catalog files.
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

// WithProgress reports the current number of articles as they are written.
func WithProgress(f WriteProgress) WriterOption {
	return func(w *Writer) {
		w.progress = f
	}
}

// WriteProgress is the signature of the progress callback while writing.
// You can tell the Writer to report progress with the WithProgress option.
type WriteProgress func(written int)

// xmlNamespace returns the XML namespace to use for the output.
func (w *Writer) xmlNamespace(writer CatalogWriter) string {
	switch writer.Transaction() {
	case UpdateProducts:
		return "http://www.bmecat.org/bmecat/1.2/bmecat_update_products"
	case UpdatePrices:
		return "http://www.bmecat.org/bmecat/1.2/bmecat_update_prices"
	default:
		return "http://www.bmecat.org/bmecat/1.2/bmecat_new_catalog"
	}
}

// txStartElement returns the XML StartElement for the BMEcat transaction,
// e.g. "T_NEW_CATALOG".
func (w *Writer) txStartElement(writer CatalogWriter) xml.StartElement {
	tx := writer.Transaction().String()
	attr := []xml.Attr{}
	switch writer.Transaction() {
	case UpdateProducts:
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "prev_version"}, Value: fmt.Sprint(writer.PreviousVersion())})
	case UpdatePrices:
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
// You must also pass a channel of articles, which Do loops over.
// If the articles channel is closed, Do will write the rest of
// the BMEcat file, and then return.
func (w *Writer) Do(ctx context.Context, writer CatalogWriter) error {
	w.enc = xml.NewEncoder(w.w)
	if w.indent != "" {
		w.enc.Indent("", w.indent)
	}
	if err := w.writeLeadIn(writer); err != nil {
		return errors.Wrap(err, "bmecat/v12: unable to write lead in")
	}
	header := writer.Header()
	if header != nil {
		if err := w.enc.Encode(header); err != nil {
			return errors.Wrap(err, "bmecat/v12: unable to write Header")
		}
	}
	tx := writer.Transaction().String()
	if err := w.enc.EncodeToken(w.txStartElement(writer)); err != nil {
		return errors.Wrapf(err, "bmecat/v12: unable to write opening %s", tx)
	}

	if writer.Transaction() == NewCatalog {
		// FEATURE_SYSTEM

		// CLASSIFICATION_SYSTEM
		if system := writer.ClassificationSystem(); system != nil {
			if !system.IsBlank() {
				if err := w.enc.Encode(system); err != nil {
					return errors.Wrap(err, "bmecat/v12: unable to write CLASSIFICATION_SYSTEM")
				}
			}
		}

		// CATALOG_GROUP_SYSTEM
		if gw, ok := writer.(CatalogGroupSystemWriter); ok {
			if system := gw.GroupSystem(); system != nil {
				if !system.IsBlank() {
					if err := w.enc.Encode(system); err != nil {
						return errors.Wrap(err, "bmecat/v12: unable to write CATALOG_GROUP_SYSTEM")
					}
				}

			}
		}
	}

	// ARTICLE
	if err := w.writeArticles(ctx, writer); err != nil {
		return errors.Wrapf(err, "bmecat/v12: unable to write ARTICLE")
	}

	if writer.Transaction() != UpdatePrices {
		// ARTICLE_TO_CATALOGROUP_MAP
	}

	if err := w.enc.EncodeToken(w.txEndElement(writer)); err != nil {
		return errors.Wrapf(err, "bmecat/v12: unable to write closing %s", tx)
	}
	if err := w.writeLeadOut(); err != nil {
		return errors.Wrap(err, "bmecat/v12: unable to write lead out")
	}
	return w.enc.Flush()
}

func (w *Writer) writeLeadIn(writer CatalogWriter) error {
	_, err := fmt.Fprint(w.w, xml.Header)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w.w, `<!DOCTYPE BMECAT SYSTEM "bmecat_new_catalog.dtd">`)
	if err != nil {
		return err
	}
	// <BMECAT version="1.2" xmlns="http://www.bmecat.org/bmecat/1.2/bmecat_new_catalog">`, writer.Language())
	attr := []xml.Attr{
		xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: w.xmlNamespace(writer)},
		xml.Attr{Name: xml.Name{Local: "version"}, Value: "1.2"},
	}
	/*
		if language := writer.Language(); language != "" {
			attr = append(attr, xml.Attr{Name: xml.Name{Local: "xml:lang"}, Value: language})
		}
	*/
	t := xml.StartElement{
		Name: xml.Name{Local: "BMECAT"},
		Attr: attr,
	}
	return w.enc.EncodeToken(t)
}

func (w *Writer) writeLeadOut() error {
	return w.enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "BMECAT"}})
}

func (w *Writer) writeArticles(ctx context.Context, writer CatalogWriter) error {
	articlesCh, errCh := writer.Articles(ctx)
	if articlesCh == nil {
		return nil
	}

	var stop bool
	var written uint32
	for !stop {
		select {
		case a, ok := <-articlesCh:
			if !ok {
				stop = true
				break
			}
			if err := w.writeArticle(a); err != nil {
				return errors.Wrapf(err, "unable to write SUPPLIER_AID %q", a.SupplierAID)
			}
			current := atomic.AddUint32(&written, 1)
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

func (w *Writer) writeArticle(a *Article) error {
	// TODO(oe) Only serialize the part of the article that is required by w.Transaction
	err := w.enc.Encode(a)
	if err != nil {
		return err
	}
	return nil
}
