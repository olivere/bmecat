package bmecat2005

import (
	"encoding/xml"
	"testing"
)

func TestMarshalUDX(t *testing.T) {
	udx := &UserDefinedExtensions{}
	udx.Fields.Add("SYSTEM.CUSTOM_FIELD1", "A")
	udx.Fields.Add("SYSTEM.CUSTOM_FIELD5", "E")
	if want, have := 2, len(udx.Fields); want != have {
		t.Fatalf("want len = %d, have: %d", want, have)
	}
	out, err := xml.Marshal(udx)
	if err != nil {
		t.Fatal(err)
	}
	expected := `<USER_DEFINED_EXTENSIONS><UDX.SYSTEM.CUSTOM_FIELD1>A</UDX.SYSTEM.CUSTOM_FIELD1><UDX.SYSTEM.CUSTOM_FIELD5>E</UDX.SYSTEM.CUSTOM_FIELD5></USER_DEFINED_EXTENSIONS>`
	if want, have := expected, string(out); want != have {
		t.Fatalf("want:\n%v\nhave:\n%v", want, have)
	}
}

func TestMarshalRawUDX(t *testing.T) {
	udx := &UserDefinedExtensions{}
	udx.Fields.AddRaw("SYSTEM.CUSTOM_FIELD1", "<ID>A</ID><VALUE>Value</VALUE>")
	udx.Fields.AddRaw("SYSTEM.CUSTOM_FIELD5", "E")
	if want, have := 2, len(udx.Fields); want != have {
		t.Fatalf("want len = %d, have: %d", want, have)
	}
	out, err := xml.Marshal(udx)
	if err != nil {
		t.Fatal(err)
	}
	expected := `<USER_DEFINED_EXTENSIONS><UDX.SYSTEM.CUSTOM_FIELD1><ID>A</ID><VALUE>Value</VALUE></UDX.SYSTEM.CUSTOM_FIELD1><UDX.SYSTEM.CUSTOM_FIELD5>E</UDX.SYSTEM.CUSTOM_FIELD5></USER_DEFINED_EXTENSIONS>`
	if want, have := expected, string(out); want != have {
		t.Fatalf("want:\n%v\nhave:\n%v", want, have)
	}
}

func TestUnmarshalUDX(t *testing.T) {
	input := `<USER_DEFINED_EXTENSIONS><UDX.SYSTEM.CUSTOM_FIELD1>A</UDX.SYSTEM.CUSTOM_FIELD1><UDX.SYSTEM.CUSTOM_FIELD5>E</UDX.SYSTEM.CUSTOM_FIELD5><UDX.WALLMEDIEN.PROPERTIES><UDX.WALLMEDIEN.PROPERTY><UDX.WALLMEDIEN.PROPERTY.NAME>EXTCONFIGFORM</UDX.WALLMEDIEN.PROPERTY.NAME><UDX.WALLMEDIEN.PROPERTY.VALUE>ADV_Relevanz</UDX.WALLMEDIEN.PROPERTY.VALUE></UDX.WALLMEDIEN.PROPERTY></UDX.WALLMEDIEN.PROPERTIES></USER_DEFINED_EXTENSIONS>`
	udx := &UserDefinedExtensions{}
	err := xml.Unmarshal([]byte(input), udx)
	if err != nil {
		t.Fatal(err)
	}

	s, ok := udx.Fields.Get("SYSTEM.CUSTOM_FIELD1")
	if !ok {
		t.Fatalf("expected to find UDX field %q", "SYSTEM.CUSTOM_FIELD1")
	}
	if want, have := "A", s; want != have {
		t.Fatalf("want %q, have %q", want, have)
	}

	s, ok = udx.Fields.Get("SYSTEM.CUSTOM_FIELD5")
	if !ok {
		t.Fatalf("expected to find UDX field %q", "SYSTEM.CUSTOM_FIELD5")
	}
	if want, have := "E", s; want != have {
		t.Fatalf("want %q, have %q", want, have)
	}

	s, ok = udx.Fields.GetInnerXML("WALLMEDIEN.PROPERTIES")
	if !ok {
		t.Fatalf("expected to find UDX field %q", "WALLMEDIEN.PROPERTIES")
	}
	if want, have := `<UDX.WALLMEDIEN.PROPERTY><UDX.WALLMEDIEN.PROPERTY.NAME>EXTCONFIGFORM</UDX.WALLMEDIEN.PROPERTY.NAME><UDX.WALLMEDIEN.PROPERTY.VALUE>ADV_Relevanz</UDX.WALLMEDIEN.PROPERTY.VALUE></UDX.WALLMEDIEN.PROPERTY>`, s; want != have {
		t.Fatalf("want %q, have %q", want, have)
	}
}
