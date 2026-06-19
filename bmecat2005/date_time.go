package bmecat2005

import "time"

var (
	DefaultStartDate time.Time
	DefaultEndDate   time.Time
)

func init() {
	DefaultStartDate, _ = time.Parse("2006-01-02", "1970-01-01")
	DefaultEndDate, _ = time.Parse("2006-01-02", "2038-01-19")
}

type DateTime struct {
	Type           string `xml:"type,attr"`
	DateString     string `xml:"DATE"`
	TimeString     string `xml:"TIME,omitempty"`
	TimeZoneString string `xml:"TIMEZONE,omitempty"`
}

// Time converts the DateTime to a time.Time. When TIMEZONE is set it applies
// the offset, accepting both the "Z" (UTC) and "±HH:MM" forms that NewDateTime
// emits; the returned time then carries that offset. A TIMEZONE that does not
// match either form is ignored and the wall-clock is interpreted as UTC, which
// is also the behavior when TIMEZONE is empty.
func (dt DateTime) Time() (time.Time, error) {
	ts := dt.TimeString
	if ts == "" {
		ts = "00:00:00"
	}
	if dt.TimeZoneString != "" {
		if t, err := time.Parse("2006-01-02 15:04:05Z07:00", dt.DateString+" "+ts+dt.TimeZoneString); err == nil {
			return t, nil
		}
		// Fall through: a malformed TIMEZONE is ignored rather than failing the
		// whole parse, so a valid date never degrades to an error.
	}
	return time.Parse("2006-01-02 15:04:05", dt.DateString+" "+ts)
}

func NewDateTime(typ string, dt time.Time) *DateTime {
	if dt.IsZero() {
		return nil
	}
	out := &DateTime{
		Type:       typ,
		DateString: dt.Format("2006-01-02"),
		TimeString: dt.Format("15:04:05"),
	}
	if dt.Location() == time.UTC {
		out.TimeZoneString = "Z"
	} else {
		out.TimeZoneString = dt.Format("-07:00")
	}
	return out
}
