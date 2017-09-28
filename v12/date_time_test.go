package v12

import (
	"testing"
	"time"
)

func TestNewDateTime(t *testing.T) {
	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		Typ      string
		Date     time.Time
		Expected *DateTime
	}{
		// #0
		{
			// Date is zero
			Expected: nil,
		},
		// #1
		{
			Typ:  DateTimeGenerationDate,
			Date: time.Date(2000, 10, 24, 20, 38, 00, 999, time.UTC),
			Expected: &DateTime{
				Type:           DateTimeGenerationDate,
				DateString:     "2000-10-24",
				TimeString:     "20:38:00",
				TimeZoneString: "Z",
			},
		},
		// #2
		{
			Typ:  DateTimeValidStartDate,
			Date: time.Date(2017, 8, 1, 9, 12, 59, 999, nyc),
			Expected: &DateTime{
				Type:           DateTimeValidStartDate,
				DateString:     "2017-08-01",
				TimeString:     "09:12:59",
				TimeZoneString: "-04:00",
			},
		},
	}

	for i, tt := range tests {
		have := NewDateTime(tt.Typ, tt.Date)
		want := tt.Expected
		if have == nil && want != nil {
			t.Fatalf("#%d: want %#v, have nil", i, want)
		}
		if have != nil && want == nil {
			t.Fatalf("#%d: want nil, have %#v", i, have)
		}
		if have != nil && want != nil {
			if a, b := want.Type, have.Type; a != b {
				t.Fatalf("#%d: want Type = %q, have %q", i, a, b)
			}
			if a, b := want.DateString, have.DateString; a != b {
				t.Fatalf("#%d: want DateString = %q, have %q", i, a, b)
			}
			if a, b := want.TimeString, have.TimeString; a != b {
				t.Fatalf("#%d: want TimeString = %q, have %q", i, a, b)
			}
			if a, b := want.TimeZoneString, have.TimeZoneString; a != b {
				t.Fatalf("#%d: want TimeZoneString = %q, have %q", i, a, b)
			}
		}
	}
}
