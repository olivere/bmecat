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

func (dt DateTime) Time() (time.Time, error) {
	ts := dt.TimeString
	if ts == "" {
		ts = "00:00:00"
	}
	// TODO time zone support
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
