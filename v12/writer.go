package v12

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

// Writer allows writing BMEcat 1.2 catalog files.
type Writer struct {
	w        io.Writer
	progress WriteProgress
	enc      *xml.Encoder

	// Indent setting for the writer.
	Indent string
	// Language that is used for the xml:lang attribute.
	Language string
	// Transaction specifies the mode of the catalog, e.g. "T_NEW_CATALOG" (default),
	// "T_UPDATE_PRODUCTS", or "T_UPDATE_PRICES".
	Transaction Transaction
	// Header of the catalog.
	Header *Header
	// PreviousVersion of the catalog. It is required for BMEcat "T_UPDATE_PRODUCTS"
	// and "T_UPDATE_PRICES".
	PreviousVersion int
}

// NewWriter creates a new Writer. It expects an underlying io.Writer
// which essentially gets the XML content. You can also pass additional
// options like WithProgress.
func NewWriter(w io.Writer, options ...WriterOption) *Writer {
	writer := &Writer{w: w, Transaction: NewCatalog}
	for _, o := range options {
		o(writer)
	}
	return writer
}

// WriterOption is the signature of options to pass into a NewWriter.
type WriterOption func(*Writer)

// WithProgress reports the current number of articles as they are written.
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
func (w *Writer) txStartElement() xml.StartElement {
	tx := w.Transaction.String()
	attr := []xml.Attr{}
	switch w.Transaction {
	case UpdateProducts:
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "prev_version"}, Value: fmt.Sprint(w.PreviousVersion)})
	case UpdatePrices:
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "prev_version"}, Value: fmt.Sprint(w.PreviousVersion)})
	}
	return xml.StartElement{Name: xml.Name{Local: tx}, Attr: attr}
}

// txEndElement returns the XML EndElement for the BMEcat transaction,
// e.g. "T_NEW_CATALOG".
func (w *Writer) txEndElement() xml.EndElement {
	return xml.EndElement{Name: xml.Name{Local: w.Transaction.String()}}
}

// Do writes the BMEcat file.
//
// You must pass a context, which can be canceled to stop writing.
//
// You must also pass a channel of articles, which Do loops over.
// If the articles channel is closed, Do will write the rest of
// the BMEcat file, and then return.
func (w *Writer) Do(ctx context.Context, articles <-chan *Article) error {
	w.enc = xml.NewEncoder(w.w)
	if w.Indent != "" {
		w.enc.Indent("", w.Indent)
	}
	if err := w.writeLeadIn(); err != nil {
		return errors.Wrap(err, "bmecat/v12: unable to write lead in")
	}
	if w.Header != nil {
		if err := w.enc.Encode(w.Header); err != nil {
			return errors.Wrap(err, "bmecat/v12: unable to write Header")
		}
	}
	tx := w.Transaction.String()
	if err := w.enc.EncodeToken(w.txStartElement()); err != nil {
		return errors.Wrapf(err, "bmecat/v12: unable to write opening %s", tx)
	}

	// FEATURE_SYSTEM
	// CLASSIFICATION_SYSTEM
	// CATALOG_GROUP_SYSTEM
	// ARTICLE
	var stop bool
	var written uint32
	for !stop {
		select {
		case a, ok := <-articles:
			if !ok {
				stop = true
				break
			}
			if err := w.writeArticle(a); err != nil {
				return errors.Wrapf(err, "bmecat/v12: unable to write article %q: %v", a.SupplierAID)
			}
			current := atomic.AddUint32(&written, 1)
			if w.progress != nil {
				w.progress(int(current))
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	// ARTICLE_TO_CATALOGROUP_MAP

	if err := w.enc.EncodeToken(w.txEndElement()); err != nil {
		return errors.Wrapf(err, "bmecat/v12: unable to write closing %s", tx)
	}
	if err := w.writeLeadOut(); err != nil {
		return errors.Wrap(err, "bmecat/v12: unable to write lead out")
	}
	return w.enc.Flush()
}

func (w *Writer) writeLeadIn() error {
	_, err := fmt.Fprint(w.w, xml.Header)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w.w, `<!DOCTYPE BMECAT SYSTEM "bmecat_new_catalog.dtd">`)
	if err != nil {
		return err
	}
	// <BMECAT version="1.2" xml:lang="de" xmlns="http://www.bmecat.org/bmecat/1.2/bmecat_new_catalog">`, w.Language)
	attr := []xml.Attr{
		xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: "http://www.bmecat.org/bmecat/1.2/bmecat_new_catalog"},
		xml.Attr{Name: xml.Name{Local: "version"}, Value: "1.2"},
	}
	if w.Language != "" {
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "xml:lang"}, Value: w.Language})
	}
	t := xml.StartElement{
		Name: xml.Name{Local: "BMECAT"},
		Attr: attr,
	}
	return w.enc.EncodeToken(t)
}

func (w *Writer) writeLeadOut() error {
	return w.enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "BMECAT"}})
	/*
		_, err := fmt.Fprintln(w.w, `</BMECAT>`)
		if err != nil {
			return err
		}
		return nil
	*/
}

func (w *Writer) writeArticle(a *Article) error {
	// TODO(oe) Only serialize the part of the article that is required by w.Transaction
	err := w.enc.Encode(a)
	if err != nil {
		return err
	}
	return nil
}
