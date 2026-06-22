package bmecat2005

import (
	"encoding/xml"
)

const (
	MimeTypeURL   = "url"
	MimeTypePDF   = "application/pdf"
	MimeTypeJPEG  = "image/jpeg"
	MimeTypeGIF   = "image/gif"
	MimeTypeHTML  = "text/html"
	MimeTypePlain = "text/plain"

	MimePurposeThumbnail = "thumbnail"
	MimePurposeNormal    = "normal"
	MimePurposeDetail    = "detail"
	MimePurposeDataSheet = "data_sheet"
	MimePurposeLogo      = "logo"
	MimePurposeOthers    = "others"
	// MimePurposeIcon and MimePurposeSafetyDataSheet were introduced in
	// BMEcat 2005 and are therefore available in this package (unlike in
	// bmecat12).
	MimePurposeIcon            = "icon"
	MimePurposeSafetyDataSheet = "safety_data_sheet"
)

// MimeInfo represents the MIME_INFO element from the BMEcat specification.
type MimeInfo struct {
	XMLName xml.Name `xml:"MIME_INFO"`

	Mimes []*Mime `xml:"MIME"`
}

// Mime represents the MIME element from the BMEcat specification.
type Mime struct {
	XMLName xml.Name `xml:"MIME"`

	Type    string           `xml:"MIME_TYPE,omitempty"`
	Source  LocalizedStrings `xml:"MIME_SOURCE"`
	Descr   LocalizedStrings `xml:"MIME_DESCR,omitempty"`
	Alt     LocalizedStrings `xml:"MIME_ALT,omitempty"`
	Purpose string           `xml:"MIME_PURPOSE,omitempty"`
	Order   int              `xml:"MIME_ORDER,omitempty"`
}

// sourceByPurpose returns the MIME_SOURCE of the first MIME with the given
// purpose, or an empty string when none matches.
func (m *MimeInfo) sourceByPurpose(purpose string) string {
	for _, mime := range m.Mimes {
		if mime.Purpose == purpose {
			return mime.Source.Value()
		}
	}
	return ""
}

// ThumbnailSource returns the URL of the thumbnail image.
// If no such image can be found, an empty string is returned.
func (m *MimeInfo) ThumbnailSource() string { return m.sourceByPurpose(MimePurposeThumbnail) }

// NormalSource returns the URL of the normal image.
// If no such image can be found, an empty string is returned.
func (m *MimeInfo) NormalSource() string { return m.sourceByPurpose(MimePurposeNormal) }

// DetailSource returns the URL of the detail image.
// If no such image can be found, an empty string is returned.
func (m *MimeInfo) DetailSource() string { return m.sourceByPurpose(MimePurposeDetail) }

// DataSheetSource returns the URL of the data sheet.
// If no data sheet is found, an empty string is returned.
func (m *MimeInfo) DataSheetSource() string { return m.sourceByPurpose(MimePurposeDataSheet) }

// LogoSource returns the URL of the logo.
// If no logo is found, an empty string is returned.
func (m *MimeInfo) LogoSource() string { return m.sourceByPurpose(MimePurposeLogo) }

// IconSource returns the URL of the icon.
// If no icon can be found, an empty string is returned.
func (m *MimeInfo) IconSource() string { return m.sourceByPurpose(MimePurposeIcon) }

// SafetyDataSheetSource returns the URL of the safety data sheet.
// If no such sheet is found, an empty string is returned.
func (m *MimeInfo) SafetyDataSheetSource() string {
	return m.sourceByPurpose(MimePurposeSafetyDataSheet)
}
