package bmecat2005

import "encoding/xml"

// LocalizedString is a single language variant of a character-data element that
// carries an xml-style lang attribute, such as DESCRIPTION_SHORT or
// DESCRIPTION_LONG. Lang is empty when the source element has no lang attribute.
type LocalizedString struct {
	Lang  string `xml:"lang,attr,omitempty"`
	Value string `xml:",chardata"`
}

// LocalizedStrings is an ordered list of language variants of the same element.
// BMEcat allows an element such as DESCRIPTION_SHORT to appear once per
// language; the variants are kept in document order.
type LocalizedStrings []LocalizedString

// Localized returns a LocalizedStrings holding one entry per value, each with no
// language attribute. It is the convenient constructor for the common
// single-language case (one value) and for a plain list of values.
func Localized(values ...string) LocalizedStrings {
	out := make(LocalizedStrings, len(values))
	for i, v := range values {
		out[i] = LocalizedString{Value: v}
	}
	return out
}

// Get returns the value for the given language. It returns the value of the
// exact-match variant if one exists, otherwise the first variant in document
// order, and the empty string when there are no variants.
func (s LocalizedStrings) Get(lang string) string {
	for _, ls := range s {
		if ls.Lang == lang {
			return ls.Value
		}
	}
	return s.Value()
}

// Value returns the first variant's value in document order, or the empty
// string when there are no variants.
func (s LocalizedStrings) Value() string {
	if len(s) > 0 {
		return s[0].Value
	}
	return ""
}

// All returns every value for the given language, in document order. It is the
// multi-valued counterpart of Get, for elements that legitimately repeat (such
// as KEYWORD): a flat list carries both several distinct values and their
// language variants, and All filters to one language. When no entry matches the
// language it falls back to every value (so single-language data tagged without
// a lang is still returned), and it returns nil when there are no entries.
func (s LocalizedStrings) All(lang string) []string {
	var out []string
	for _, ls := range s {
		if ls.Lang == lang {
			out = append(out, ls.Value)
		}
	}
	if out == nil && len(s) > 0 {
		for _, ls := range s {
			out = append(out, ls.Value)
		}
	}
	return out
}

// Set adds or replaces the variant for the given language in place.
func (s *LocalizedStrings) Set(lang, value string) {
	for i := range *s {
		if (*s)[i].Lang == lang {
			(*s)[i].Value = value
			return
		}
	}
	*s = append(*s, LocalizedString{Lang: lang, Value: value})
}

// MarshalXML emits one element per variant, carrying a lang attribute only for
// variants whose Lang is set. When there are no variants it still emits a single
// empty element, so a required element such as DESCRIPTION_SHORT is preserved;
// callers that want the element omitted when empty tag the field with
// ",omitempty", which suppresses this method for an empty slice.
func (s LocalizedStrings) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if len(s) == 0 {
		return e.EncodeElement("", start)
	}
	for _, ls := range s {
		if err := e.EncodeElement(ls, start); err != nil {
			return err
		}
	}
	return nil
}
