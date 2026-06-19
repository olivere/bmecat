package bmecat

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/olivere/bmecat/bmecat12"
	"github.com/olivere/bmecat/bmecat2005"
	"github.com/olivere/bmecat/internal"
)

// CharsetReaderFunc typedef's the CharsetReader from the Decoder in encoding/xml.
type CharsetReaderFunc func(charset string, input io.Reader) (io.Reader, error)

// Reader reads a BMEcat file of either supported version. It auto-detects the
// version from the root <BMECAT version="…"> element and dispatches to the
// bmecat12 or bmecat2005 reader, normalizing both into the version-neutral
// types in this package.
//
// Reader is the recommended entry point. Callers that need raw, version-
// specific fidelity can still use the bmecat12 and bmecat2005 packages
// directly.
type Reader struct {
	r             io.ReadSeeker
	charsetReader CharsetReaderFunc
	progress      ReaderProgress
}

// NewReader creates a new Reader over r. Options such as WithCharsetReader and
// WithReaderProgress may be passed.
func NewReader(r io.ReadSeeker, options ...ReaderOption) *Reader {
	reader := &Reader{
		r:             r,
		charsetReader: internal.AutoCharsetReader,
	}
	for _, o := range options {
		o(reader)
	}
	return reader
}

// ReaderOption is the signature of options to pass into NewReader.
type ReaderOption func(*Reader)

// WithCharsetReader specifies the charset reader used to decode XML data. It is
// applied both to version detection and to the underlying version reader.
func WithCharsetReader(f CharsetReaderFunc) ReaderOption {
	return func(r *Reader) {
		r.charsetReader = f
	}
}

// ReaderProgress is the signature for reporting progress. It reports the
// current pass of the underlying parser (1 or 2) and the byte offset.
type ReaderProgress func(pass int, offset int64)

// WithReaderProgress specifies a callback invoked periodically as the file is
// read.
func WithReaderProgress(f ReaderProgress) ReaderOption {
	return func(r *Reader) {
		r.progress = f
	}
}

// DetectVersion reports the BMEcat version of the document without consuming
// it for the caller: it seeks back to the start before returning.
func (r *Reader) DetectVersion() (Version, error) {
	if _, err := r.r.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}
	dec := xml.NewDecoder(r.r)
	dec.CharsetReader = r.charsetReader
	defer r.r.Seek(0, io.SeekStart)

	for {
		t, err := dec.Token()
		if err == io.EOF {
			return 0, fmt.Errorf("bmecat: no BMECAT element found")
		}
		if err != nil {
			return 0, err
		}
		se, ok := t.(xml.StartElement)
		if !ok || se.Name.Local != "BMECAT" {
			continue
		}
		// Prefer the explicit version attribute.
		for _, attr := range se.Attr {
			if attr.Name.Local == "version" {
				return versionFromString(attr.Value)
			}
		}
		// Fall back to the namespace, which is fixed per version.
		if v, ok := versionFromNamespace(se.Name.Space); ok {
			return v, nil
		}
		return 0, fmt.Errorf("bmecat: BMECAT element has no recognizable version")
	}
}

// DetectTransaction reports the document-level BMEcat transaction without
// consuming the document for the caller: it seeks back to the start before
// returning.
//
// It mirrors DetectVersion and is the cheap, phase-1 way to gate on the
// transaction — for example to reject incremental updates (Transaction.IsUpdate)
// up front, without the full parse that Do performs. The same value is also
// surfaced on Header.Transaction during Do.
func (r *Reader) DetectTransaction() (Transaction, error) {
	if _, err := r.r.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}
	dec := xml.NewDecoder(r.r)
	dec.CharsetReader = r.charsetReader
	defer r.r.Seek(0, io.SeekStart)

	var inBMECAT bool
	for {
		t, err := dec.Token()
		if err == io.EOF {
			return 0, fmt.Errorf("bmecat: no transaction element found")
		}
		if err != nil {
			return 0, err
		}
		se, ok := t.(xml.StartElement)
		if !ok {
			continue
		}
		if !inBMECAT {
			if se.Name.Local == "BMECAT" {
				inBMECAT = true
			}
			continue
		}
		// The transaction element is the first non-HEADER child of BMECAT.
		if tx, ok := transactionFromElement(se.Name.Local); ok {
			return tx, nil
		}
		// Skip other children (e.g. HEADER) without descending into them.
		if err := dec.Skip(); err != nil {
			return 0, err
		}
	}
}

func versionFromString(value string) (Version, error) {
	switch {
	case strings.HasPrefix(value, "1.2"):
		return Version12, nil
	case strings.HasPrefix(value, "2005"):
		return Version2005, nil
	default:
		return 0, fmt.Errorf("bmecat: unsupported BMECAT version %q", value)
	}
}

func versionFromNamespace(ns string) (Version, bool) {
	switch {
	case strings.Contains(ns, "/bmecat/1.2"):
		return Version12, true
	case strings.Contains(ns, "/bmecat/2005"):
		return Version2005, true
	default:
		return 0, false
	}
}

// Do reads the BMEcat file, detecting its version and dispatching to the
// matching version reader. The handler may implement any combination of
// HeaderHandler, CatalogGroupHandler, ClassificationGroupHandler,
// ProductHandler and CompletionHandler.
func (r *Reader) Do(ctx context.Context, handler any) error {
	version, err := r.DetectVersion()
	if err != nil {
		return err
	}
	transaction, err := r.DetectTransaction()
	if err != nil {
		return err
	}

	switch version {
	case Version2005:
		return r.do2005(ctx, handler, transaction)
	default:
		return r.do12(ctx, handler, transaction)
	}
}

func (r *Reader) do12(ctx context.Context, handler any, transaction Transaction) error {
	opts := []bmecat12.ReaderOption{
		bmecat12.WithCharsetReader(bmecat12.CharsetReaderFunc(r.charsetReader)),
	}
	if r.progress != nil {
		opts = append(opts, bmecat12.WithReaderProgress(bmecat12.ReaderProgress(r.progress)))
	}
	adapter := newV12Adapter(handler, transaction)
	err := bmecat12.NewReader(r.r, opts...).Do(ctx, adapter)
	if err == nil && adapter.headerErr != nil {
		// The bmecat12 reader swallows non-EOF HandleHeader errors (#16);
		// surface it so the neutral HeaderHandler contract holds.
		return adapter.headerErr
	}
	return err
}

func (r *Reader) do2005(ctx context.Context, handler any, transaction Transaction) error {
	opts := []bmecat2005.ReaderOption{
		bmecat2005.WithCharsetReader(bmecat2005.CharsetReaderFunc(r.charsetReader)),
	}
	if r.progress != nil {
		opts = append(opts, bmecat2005.WithReaderProgress(bmecat2005.ReaderProgress(r.progress)))
	}
	return bmecat2005.NewReader(r.r, opts...).Do(ctx, newV2005Adapter(handler, transaction))
}
