package internal

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/transform"
)

// AutoCharsetReader returns a charset reader for XML decoding. It resolves the
// declared encoding through golang.org/x/text/encoding/htmlindex, which accepts
// every label in the WHATWG Encoding Standard — UTF-8, the ISO-8859 family,
// Windows code pages, and CJK encodings such as GBK, GB18030, Big5, Shift-JIS
// and EUC-KR — so encodings need not be enumerated here.
//
// Following that standard, ISO-8859-1 (and latin1) is decoded as its
// Windows-1252 superset and GB2312 as GBK, which matches how real-world
// catalogs are authored.
func AutoCharsetReader(encoding string, r io.Reader) (io.Reader, error) {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "", "utf-8", "utf8":
		// Already UTF-8: no decoding necessary.
		return r, nil
	}

	enc, err := htmlindex.Get(encoding)
	if err != nil || enc == nil {
		return nil, fmt.Errorf("bmecat: unknown encoding: %q", encoding)
	}
	return transform.NewReader(r, enc.NewDecoder()), nil
}
