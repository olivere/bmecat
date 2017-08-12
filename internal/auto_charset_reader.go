package internal

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// AutoCharsetReader returns a charset reader for XML decoding.
func AutoCharsetReader(encoding string, r io.Reader) (io.Reader, error) {
	enc := strings.ToLower(encoding)

	if enc == "" || enc == "utf-8" || enc == "utf8" {
		return r, nil
	}

	switch enc {
	case "ibm code page 437", "cp437", "cp-437":
		return transform.NewReader(r, charmap.CodePage437.NewDecoder()), nil
	case "ibm code page 866", "cp866", "cp-866":
		return transform.NewReader(r, charmap.CodePage866.NewDecoder()), nil
	case "iso88591", "iso 8859-1", "iso8859-1", "iso-8859-1":
		return transform.NewReader(r, charmap.Windows1252.NewDecoder()), nil
	case "iso88592", "iso 8859-2", "iso8859-2", "iso-8859-2":
		return transform.NewReader(r, charmap.ISO8859_2.NewDecoder()), nil
	case "iso88593", "iso 8859-3", "iso8859-3", "iso-8859-3":
		return transform.NewReader(r, charmap.ISO8859_3.NewDecoder()), nil
	case "iso88594", "iso 8859-4", "iso8859-4", "iso-8859-4":
		return transform.NewReader(r, charmap.ISO8859_4.NewDecoder()), nil
	case "iso88595", "iso 8859-5", "iso8859-5", "iso-8859-5":
		return transform.NewReader(r, charmap.ISO8859_5.NewDecoder()), nil
	case "iso88596", "iso 8859-6", "iso8859-6", "iso-8859-6":
		return transform.NewReader(r, charmap.ISO8859_6.NewDecoder()), nil
	case "iso88597", "iso 8859-7", "iso8859-7", "iso-8859-7":
		return transform.NewReader(r, charmap.ISO8859_7.NewDecoder()), nil
	case "iso88598", "iso 8859-8", "iso8859-8", "iso-8859-8":
		return transform.NewReader(r, charmap.ISO8859_8.NewDecoder()), nil
	case "iso885910", "iso 8859-10", "iso8859-10", "iso-8859-10":
		return transform.NewReader(r, charmap.ISO8859_10.NewDecoder()), nil
	case "iso885913", "iso 8859-13", "iso8859-13", "iso-8859-13":
		return transform.NewReader(r, charmap.ISO8859_13.NewDecoder()), nil
	case "iso885914", "iso 8859-14", "iso8859-14", "iso-8859-14":
		return transform.NewReader(r, charmap.ISO8859_14.NewDecoder()), nil
	case "iso885915", "iso 8859-15", "iso8859-15", "iso-8859-15":
		return transform.NewReader(r, charmap.ISO8859_15.NewDecoder()), nil
	case "iso885916", "iso 8859-16", "iso8859-16", "iso-8859-16":
		return transform.NewReader(r, charmap.ISO8859_16.NewDecoder()), nil
	case "windows1252", "windows-1252":
		return transform.NewReader(r, charmap.Windows1252.NewDecoder()), nil
	}

	return nil, fmt.Errorf("bmecat: unknown encoding: %s", encoding)
}
