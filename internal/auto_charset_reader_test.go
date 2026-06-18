package internal

import (
	"io"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

func TestAutoCharsetReaderPassthrough(t *testing.T) {
	// UTF-8 and empty encodings must be returned unchanged.
	for _, enc := range []string{"", "utf-8", "UTF-8", "utf8"} {
		r, err := AutoCharsetReader(enc, strings.NewReader("Müller"))
		if err != nil {
			t.Fatalf("encoding %q: unexpected error: %v", enc, err)
		}
		data, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("encoding %q: read failed: %v", enc, err)
		}
		if got, want := string(data), "Müller"; got != want {
			t.Errorf("encoding %q: got %q, want %q", enc, got, want)
		}
	}
}

func TestAutoCharsetReaderISO8859_1(t *testing.T) {
	// 0xE4 = ä, 0xFC = ü, 0xDF = ß in ISO-8859-1 / Windows-1252.
	input := []byte{0x4d, 0xfc, 0x6c, 0x6c, 0x65, 0x72} // "Müller" in latin1
	for _, enc := range []string{"iso-8859-1", "iso8859-1", "latin1", "windows-1252"} {
		r, err := AutoCharsetReader(enc, strings.NewReader(string(input)))
		if err != nil {
			t.Fatalf("encoding %q: unexpected error: %v", enc, err)
		}
		data, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("encoding %q: read failed: %v", enc, err)
		}
		if got, want := string(data), "Müller"; got != want {
			t.Errorf("encoding %q: got %q, want %q", enc, got, want)
		}
	}
}

func TestAutoCharsetReaderChinese(t *testing.T) {
	tests := []struct {
		enc     string
		text    string
		encoder transform.Transformer
	}{
		{"gbk", "电子产品", simplifiedchinese.GBK.NewEncoder()},
		{"gb2312", "电子产品", simplifiedchinese.GBK.NewEncoder()},
		{"gb18030", "电子产品", simplifiedchinese.GB18030.NewEncoder()},
		{"big5", "電子產品", traditionalchinese.Big5.NewEncoder()},
	}
	for _, tt := range tests {
		// Encode the UTF-8 text into the legacy encoding, then make sure the
		// charset reader decodes it back to the original UTF-8.
		encoded, _, err := transform.String(tt.encoder, tt.text)
		if err != nil {
			t.Fatalf("encoding %q: encode failed: %v", tt.enc, err)
		}
		r, err := AutoCharsetReader(tt.enc, strings.NewReader(encoded))
		if err != nil {
			t.Fatalf("encoding %q: unexpected error: %v", tt.enc, err)
		}
		data, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("encoding %q: read failed: %v", tt.enc, err)
		}
		if got, want := string(data), tt.text; got != want {
			t.Errorf("encoding %q: got %q, want %q", tt.enc, got, want)
		}
	}
}

func TestAutoCharsetReaderUnknown(t *testing.T) {
	r, err := AutoCharsetReader("no-such-charset", strings.NewReader("x"))
	if err == nil {
		t.Fatal("expected an error for an unknown encoding, got nil")
	}
	if r != nil {
		t.Errorf("expected a nil reader on error, got %v", r)
	}
}
