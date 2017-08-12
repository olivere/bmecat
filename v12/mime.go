package v12

import "encoding/xml"

const (
	MimeTypeURL   = "url"
	MimeTypePDF   = "application/pdf"
	MimeTypeJPEG  = "image/jpeg"
	MimeTypeGIF   = "image/gif"
	MimeTypeHTML  = "text/html"
	MimeTypePlain = "text/plain"

	MimePurposeThumbnail       = "thumbnail"
	MimePurposeNormal          = "normal"
	MimePurposeDetail          = "detail"
	MimePurposeDataSheet       = "data_sheet"
	MimePurposeLogo            = "logo"
	MimePurposeOthers          = "others"
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

	Type    string `xml:"MIME_TYPE,omitempty"`
	Source  string `xml:"MIME_SOURCE"`
	Descr   string `xml:"MIME_DESCR,omitempty"`
	Alt     string `xml:"MIME_ALT,omitempty"`
	Purpose string `xml:"MIME_PURPOSE,omitempty"`
	Order   int    `xml:"MIME_ORDER,omitempty"`
}

// ThumbnailSource returns the URL of the thumbnail image.
// If no such image can be found, an empty string is returned.
func (m *MimeInfo) ThumbnailSource() string {
	for _, mime := range m.Mimes {
		if mime.Purpose == MimePurposeThumbnail {
			return mime.Source
		}
	}
	return ""
}

// NormalSource returns the URL of the normal image.
// If no such image can be found, an empty string is returned.
func (m *MimeInfo) NormalSource() string {
	for _, mime := range m.Mimes {
		if mime.Purpose == MimePurposeNormal {
			return mime.Source
		}
	}
	return ""
}

// DetailSource returns the URL of the detail image.
// If no such image can be found, an empty string is returned.
func (m *MimeInfo) DetailSource() string {
	for _, mime := range m.Mimes {
		if mime.Purpose == MimePurposeDetail {
			return mime.Source
		}
	}
	return ""
}

// DataSheetSource returns the URL of the data sheet.
// If no data sheet is found, an empty string is returned.
func (m *MimeInfo) DataSheetSource() string {
	for _, mime := range m.Mimes {
		if mime.Purpose == MimePurposeDataSheet {
			return mime.Source
		}
	}
	return ""
}

// LogoSource returns the URL of the logo.
// If no logo is found, an empty string is returned.
func (m *MimeInfo) LogoSource() string {
	for _, mime := range m.Mimes {
		if mime.Purpose == MimePurposeLogo {
			return mime.Source
		}
	}
	return ""
}

// IconSource returns the URL of the icon.
// If no icon can be found, an empty string is returned.
func (m *MimeInfo) IconSource() string {
	for _, mime := range m.Mimes {
		if mime.Purpose == MimePurposeIcon {
			return mime.Source
		}
	}
	return ""
}

// SafetyDataSheetSource returns the URL of the safety data sheet.
// If no such sheet is found, an empty string is returned.
func (m *MimeInfo) SafetyDataSheetSource() string {
	for _, mime := range m.Mimes {
		if mime.Purpose == MimePurposeSafetyDataSheet {
			return mime.Source
		}
	}
	return ""
}
