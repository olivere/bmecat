package bmecat2005

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/olivere/bmecat/internal"
)

// HeaderHandler specifies the interface for a handler that wants to
// get notified when the BMEcat HEADER data were read.
type HeaderHandler interface {
	// HandleHeader, when implemented by a handler, is called when the
	// Reader passed the BMEcat HEADER element.
	//
	// HandleHeader may return io.EOF to stop the Reader from continueing
	// to read. Any other error will also stop the Reader, and the error
	// is passed to the caller of the Reader's Do method.
	HandleHeader(*Header) error
}

// CatalogGroupHandler, if implemented by a handler, is called whenever
// the Reader passed a CATALOG_STRUCTURE element with a category.
type CatalogGroupHandler interface {
	HandleCatalogGroup(*CatalogGroup) error
}

// ClassificationGroupHandler, if implemented by a handler, is called whenever
// the Reader passed a CLASSIFICATION_GROUP element with a category.
type ClassificationGroupHandler interface {
	HandleClassificationGroup(*ClassificationGroup) error
}

// ProductHandler, if implemented by a handler, is called whenever
// the Reader passed a PRODUCT element with a product.
type ProductHandler interface {
	HandleProduct(*Product) error
}

// CompletionHandler, if implemented by a handler, is called once when
// the Reader is done parsing the BMEcat document.
type CompletionHandler interface {
	HandleComplete()
}

// CharsetReaderFunc typedef's the CharsetReader from the Decoder in encoding/xml.
type CharsetReaderFunc func(charset string, input io.Reader) (io.Reader, error)

// Reader reads a BMEcat 2005 file.
type Reader struct {
	r             io.ReadSeeker
	charsetReader CharsetReaderFunc
	progress      ReaderProgress

	prodToCatalogGroupMu sync.Mutex
	prodToCatalogGroup   map[string][]string
}

// NewReader creates a new Reader. It expects an underlying io.ReadSeeker
// which essentially gets the XML content. You can also pass additional
// options like WithProgress.
func NewReader(r io.ReadSeeker, options ...ReaderOption) *Reader {
	reader := &Reader{
		r:                  r,
		charsetReader:      internal.AutoCharsetReader,
		prodToCatalogGroup: make(map[string][]string),
	}
	for _, o := range options {
		o(reader)
	}
	return reader
}

// ReaderOption is the signature of options to pass into a NewReader.
type ReaderOption func(*Reader)

// WithCharsetReader specifies the charset reader to decode XML data.
func WithCharsetReader(f CharsetReaderFunc) ReaderOption {
	return func(r *Reader) {
		r.charsetReader = f
	}
}

// ReaderProgress is the signature for reporting progress.
// When set via WithReaderProgress, it returns the current pass of the
// parser (currently 1 or 2) and the current byte offset into the XML file.
type ReaderProgress func(pass int, offset int64)

// WithReaderProgress specifies a callback that is invoked periodically to
// report progress as the BMEcat file is read.
func WithReaderProgress(f ReaderProgress) ReaderOption {
	return func(r *Reader) {
		r.progress = f
	}
}

// Do reads the BMEcat file.
//
// You must pass a context, which can be canceled to stop reading.
//
// The handler may implement any combination of HeaderHandler,
// CatalogGroupHandler, ClassificationGroupHandler, ProductHandler and
// CompletionHandler; only the implemented callbacks are invoked.
func (r *Reader) Do(ctx context.Context, handler any) error {
	_, err := r.r.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	var h struct {
		Header       HeaderHandler
		CatalogGroup CatalogGroupHandler
		ClassifGroup ClassificationGroupHandler
		Product      ProductHandler
		Complete     CompletionHandler
	}
	if f, ok := handler.(HeaderHandler); ok {
		h.Header = f
	}
	if f, ok := handler.(CatalogGroupHandler); ok {
		h.CatalogGroup = f
	}
	if f, ok := handler.(ClassificationGroupHandler); ok {
		h.ClassifGroup = f
	}
	if f, ok := handler.(ProductHandler); ok {
		h.Product = f
	}
	if f, ok := handler.(CompletionHandler); ok {
		h.Complete = f
	}

	var numProducts int
	var numCatalogGroups int
	var numClassifGroups int
	var rl *rate.Limiter

	// 1st pass
	if r.progress != nil {
		r.progress(1, 0)
		// Specify a rate limiter to only report progress once a second
		rl = rate.NewLimiter(rate.Every(1*time.Second), 1)
	}
	dec := xml.NewDecoder(r.r)
	dec.CharsetReader = r.charsetReader
	var stop bool
	for !stop {
		t, err := dec.Token()
		if err == io.EOF {
			stop = true
			break
		}
		if err != nil {
			return err
		}
		switch se := t.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "PRODUCT":
				numProducts++
			case "CATALOG_STRUCTURE":
				numCatalogGroups++
			case "CLASSIFICATION_GROUP":
				numClassifGroups++
			case "PRODUCT_TO_CATALOGGROUP_MAP":
				var m ProductToCatalogGroupMap
				if err := dec.DecodeElement(&m, &se); err != nil {
					return fmt.Errorf("bmecat/reader: unable to decode PRODUCT_TO_CATALOGGROUP_MAP around byte offset %d: %w", dec.InputOffset(), err)
				}
				r.prodToCatalogGroupMu.Lock()
				if slice, ok := r.prodToCatalogGroup[m.ProductID]; ok {
					slice = append(slice, m.CatalogGroupID)
					r.prodToCatalogGroup[m.ProductID] = slice
				} else {
					r.prodToCatalogGroup[m.ProductID] = []string{m.CatalogGroupID}
				}
				r.prodToCatalogGroupMu.Unlock()
			}
		}
		if r.progress != nil && rl.Allow() {
			r.progress(1, dec.InputOffset())
		}
		select {
		default:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Seek back to start
	if _, err := r.r.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("bmecat/reader: unable to seek back to start: %w", err)
	}

	// 2nd pass
	if r.progress != nil {
		r.progress(2, 0)
	}
	var lastPID string
	dec = xml.NewDecoder(r.r)
	dec.CharsetReader = r.charsetReader
	stop = false
	for !stop {
		t, err := dec.Token()
		if err == io.EOF {
			stop = true
			break
		}
		if err != nil {
			return err
		}
		switch se := t.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "HEADER":
				var hdr Header
				if err := dec.DecodeElement(&hdr, &se); err != nil {
					return fmt.Errorf("bmecat/reader: unable to decode HEADER around byte offset %d: %w", dec.InputOffset(), err)
				}
				hdr.NumberOfProducts = numProducts
				hdr.NumberOfCatalogGroups = numCatalogGroups
				hdr.NumberOfClassificationGroups = numClassifGroups
				r.prodToCatalogGroupMu.Lock()
				hdr.NumberOfProductToCatalogGroupMaps = len(r.prodToCatalogGroup)
				r.prodToCatalogGroupMu.Unlock()
				if h.Header != nil {
					if err := h.Header.HandleHeader(&hdr); err != nil {
						if err == io.EOF {
							stop = true
							break
						}
						return fmt.Errorf("bmecat/reader: handler for HEADER returned an error around byte offset %d: %w", dec.InputOffset(), err)
					}
				}
			case "CATALOG_STRUCTURE":
				var cg CatalogGroup
				if err := dec.DecodeElement(&cg, &se); err != nil {
					return fmt.Errorf("bmecat/reader: unable to decode CATALOG_GROUP around byte offset %d: %w", dec.InputOffset(), err)
				}
				if h.CatalogGroup != nil {
					if err := h.CatalogGroup.HandleCatalogGroup(&cg); err != nil {
						return fmt.Errorf("bmecat/reader: handler for CATALOG_GROUP %q returned an error around byte offset %d: %w", cg.ID, dec.InputOffset(), err)
					}
				}
			case "CLASSIFICATION_GROUP":
				var cg ClassificationGroup
				if err := dec.DecodeElement(&cg, &se); err != nil {
					return fmt.Errorf("bmecat/reader: unable to decode CLASSIFICATION_GROUP around byte offset %d: %w", dec.InputOffset(), err)
				}
				if h.ClassifGroup != nil {
					if err := h.ClassifGroup.HandleClassificationGroup(&cg); err != nil {
						return fmt.Errorf("bmecat/reader: handler for CLASSIFICATION_GROUP %q returned an error around byte offset %d: %w", cg.ID, dec.InputOffset(), err)
					}
				}
			case "PRODUCT":
				var p Product
				if err := dec.DecodeElement(&p, &se); err != nil {
					return fmt.Errorf("bmecat/reader: unable to decode PRODUCT after SUPPLIER_PID %q around byte offset %d: %w", lastPID, dec.InputOffset(), err)
				}
				if h.Product != nil {
					// Inject catalog group mappings
					r.prodToCatalogGroupMu.Lock()
					if ids, ok := r.prodToCatalogGroup[p.SupplierPID]; ok {
						p.CatalogGroupIDs = ids
					}
					r.prodToCatalogGroupMu.Unlock()
					// Call handler
					if err := h.Product.HandleProduct(&p); err != nil {
						return fmt.Errorf("bmecat/reader: handler for PRODUCT %q returned an error around byte offset %d: %w", p.SupplierPID, dec.InputOffset(), err)
					}
				}
				lastPID = p.SupplierPID
			}
		}
		if r.progress != nil && rl.Allow() {
			r.progress(2, dec.InputOffset())
		}
		select {
		default:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if h.Complete != nil {
		h.Complete.HandleComplete()
	}

	return nil
}
