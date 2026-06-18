package bmecat2005

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// UserDefinedExtensions allows to specify additional data.
// It can be used at various places in the specification, e.g.
// with products.
type UserDefinedExtensions struct {
	XMLName xml.Name `xml:"USER_DEFINED_EXTENSIONS"`

	// Fields of the User-Defined-Extensions.
	//
	// Each field is a name/value pair. The name is the UDX field without the
	// "UDX." prefix. E.g. a UDX with the name "UDX.SYSTEM.CUSTOM_FIELD1" has
	// a field name of "SYSTEM.CUSTOM_FIELD1".
	Fields UserDefinedExtensionFields `xml:"-"`
}

// UserDefinedExtensionFields is a list of UDX fields.
type UserDefinedExtensionFields []*UserDefinedExtensionField

// UserDefinedExtensionField represents a single UDX field.
type UserDefinedExtensionField struct {
	Name     string `xml:"-"`
	Value    string `xml:",chardata"`
	InnerXML string `xml:",innerxml"`
	Raw      bool   `xml:"-"` // true to marshal Value as raw XML, i.e. not escape it
}

// MarshalXML encodes the contents of the UserDefinedExtensions struct.
func (x *UserDefinedExtensions) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	e.EncodeToken(xml.StartElement{Name: xml.Name{Local: "USER_DEFINED_EXTENSIONS"}})
	for _, field := range x.Fields {
		local := fmt.Sprintf("UDX.%s", field.Name)
		if field.Raw {
			// Directly inject the Raw field contents into the XML element
			raw := struct {
				Value string `xml:",innerxml"`
			}{
				Value: field.Value,
			}
			e.EncodeElement(raw, xml.StartElement{Name: xml.Name{Local: local}})

		} else {
			e.EncodeToken(xml.StartElement{Name: xml.Name{Local: local}})
			e.EncodeToken(xml.CharData([]byte(field.Value)))
			e.EncodeToken(xml.EndElement{Name: xml.Name{Local: local}})
		}
	}
	e.EncodeToken(xml.EndElement{Name: xml.Name{Local: "USER_DEFINED_EXTENSIONS"}})
	return nil
}

// UnmarshalXML decodes the contents of the UserDefinedExtensions struct.
func (x *UserDefinedExtensions) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var fields []*UserDefinedExtensionField

	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch se := t.(type) {
		case xml.StartElement:
			if strings.HasPrefix(se.Name.Local, "UDX.") {
				field := &UserDefinedExtensionField{Name: se.Name.Local[4:]}
				d.DecodeElement(&field, &se)
				fields = append(fields, field)
			}
		}
	}
	x.Fields = UserDefinedExtensionFields(fields)

	return nil
}

// Add an UDX field to the list.
func (x *UserDefinedExtensionFields) Add(name, value string) {
	if x != nil {
		*x = append(*x, &UserDefinedExtensionField{Name: name, Value: value, Raw: false})
	}
}

// AddRaw adds a UDX field with the raw value that is directly injected
// into the XML document. The caller is responsible for ensuring the raw
// value is valid XML.
func (x *UserDefinedExtensionFields) AddRaw(name, value string) {
	if x != nil {
		*x = append(*x, &UserDefinedExtensionField{Name: name, Value: value, Raw: true})
	}
}

// Get returns the UDX field by name. The second return value indicates
// whether a field with that name actually exists.
func (x UserDefinedExtensionFields) Get(name string) (string, bool) {
	for _, field := range x {
		if field.Name == name {
			return field.Value, true
		}
	}
	return "", false
}

// GetInnerXML returns the inner XML of the UDX field by name.
// The second return value indicates whether a field with that
// name actually exists.
func (x UserDefinedExtensionFields) GetInnerXML(name string) (string, bool) {
	for _, field := range x {
		if field.Name == name {
			return field.InnerXML, true
		}
	}
	return "", false
}
