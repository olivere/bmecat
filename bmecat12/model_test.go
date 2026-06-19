package bmecat12

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"
)

func TestArticleFeaturesClassification(t *testing.T) {
	tests := []struct {
		name     string
		system   string
		isEclass bool
		isUnspsc bool
		version  string
	}{
		{"eclass lower", "eclass-5.1", true, false, "5.1"},
		{"eclass upper", "ECLASS-7.0", true, false, "7.0"},
		{"unspsc", "unspsc-13.2", false, true, "13.2"},
		{"unspsc no version", "UNSPSC", false, true, ""},
		{"unknown", "custom", false, false, ""},
		{"empty", "", false, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			af := ArticleFeatures{FeatureSystemName: tt.system}
			if got := af.IsEclass(); got != tt.isEclass {
				t.Errorf("IsEclass() = %v, want %v", got, tt.isEclass)
			}
			if got := af.IsUnspsc(); got != tt.isUnspsc {
				t.Errorf("IsUnspsc() = %v, want %v", got, tt.isUnspsc)
			}
			if got := af.Version(); got != tt.version {
				t.Errorf("Version() = %q, want %q", got, tt.version)
			}
		})
	}
}

func TestArticlePriceDetailsValidDates(t *testing.T) {
	apd := &ArticlePriceDetails{
		Dates: []*DateTime{
			{Type: DateTimeValidStartDate, DateString: "2020-01-02"},
			{Type: DateTimeValidEndDate, DateString: "2021-03-04"},
		},
	}
	if got, want := apd.ValidStartDate(), time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("ValidStartDate() = %v, want %v", got, want)
	}
	if got, want := apd.ValidEndDate(), time.Date(2021, 3, 4, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("ValidEndDate() = %v, want %v", got, want)
	}
}

func TestArticlePriceDetailsValidDatesDefaults(t *testing.T) {
	apd := &ArticlePriceDetails{}
	if got := apd.ValidStartDate(); !got.Equal(DefaultStartDate) {
		t.Errorf("ValidStartDate() = %v, want default %v", got, DefaultStartDate)
	}
	if got := apd.ValidEndDate(); !got.Equal(DefaultEndDate) {
		t.Errorf("ValidEndDate() = %v, want default %v", got, DefaultEndDate)
	}
}

func TestArticlePriceDetailsInvalidDatesFallBackToDefault(t *testing.T) {
	apd := &ArticlePriceDetails{
		Dates: []*DateTime{
			{Type: DateTimeValidStartDate, DateString: "not-a-date"},
			{Type: DateTimeValidEndDate, DateString: "also-bad"},
		},
	}
	if got := apd.ValidStartDate(); !got.Equal(DefaultStartDate) {
		t.Errorf("ValidStartDate() = %v, want default %v", got, DefaultStartDate)
	}
	if got := apd.ValidEndDate(); !got.Equal(DefaultEndDate) {
		t.Errorf("ValidEndDate() = %v, want default %v", got, DefaultEndDate)
	}
}

func TestArticlePriceDetailsIsDailyPrice(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"TRUE", true},
		{"true", true},
		{"1", true},
		{"t", true},
		{"T", true},
		{"FALSE", false},
		{"0", false},
		{"", false},
	}
	for _, tt := range tests {
		apd := ArticlePriceDetails{DailyPriceString: tt.value}
		if got := apd.IsDailyPrice(); got != tt.want {
			t.Errorf("IsDailyPrice(%q) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

func TestCatalogGroupTypePredicates(t *testing.T) {
	tests := []struct {
		typ    string
		isRoot bool
		isNode bool
		isLeaf bool
	}{
		{"root", true, false, false},
		{"node", false, true, false},
		{"leaf", false, false, true},
		{"", false, false, false},
	}
	for _, tt := range tests {
		cg := &CatalogGroup{Type: tt.typ}
		if got := cg.IsRoot(); got != tt.isRoot {
			t.Errorf("IsRoot(%q) = %v, want %v", tt.typ, got, tt.isRoot)
		}
		if got := cg.IsNode(); got != tt.isNode {
			t.Errorf("IsNode(%q) = %v, want %v", tt.typ, got, tt.isNode)
		}
		if got := cg.IsLeaf(); got != tt.isLeaf {
			t.Errorf("IsLeaf(%q) = %v, want %v", tt.typ, got, tt.isLeaf)
		}
	}
}

func TestClassificationGroupTypePredicates(t *testing.T) {
	if !(&ClassificationGroup{Type: "node"}).IsNode() {
		t.Error("IsNode() = false for node, want true")
	}
	if (&ClassificationGroup{Type: "leaf"}).IsNode() {
		t.Error("IsNode() = true for leaf, want false")
	}
	if !(&ClassificationGroup{Type: "leaf"}).IsLeaf() {
		t.Error("IsLeaf() = false for leaf, want true")
	}
	if (&ClassificationGroup{Type: "node"}).IsLeaf() {
		t.Error("IsLeaf() = true for node, want false")
	}
}

func TestDateTimeTime(t *testing.T) {
	// With explicit time.
	dt := DateTime{DateString: "2020-06-15", TimeString: "13:45:30"}
	got, err := dt.Time()
	if err != nil {
		t.Fatalf("Time() error: %v", err)
	}
	if want := time.Date(2020, 6, 15, 13, 45, 30, 0, time.UTC); !got.Equal(want) {
		t.Errorf("Time() = %v, want %v", got, want)
	}

	// Missing time defaults to midnight.
	dt = DateTime{DateString: "2020-06-15"}
	got, err = dt.Time()
	if err != nil {
		t.Fatalf("Time() error: %v", err)
	}
	if want := time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("Time() = %v, want %v", got, want)
	}

	// Invalid date returns an error.
	if _, err := (DateTime{DateString: "bad"}).Time(); err == nil {
		t.Error("Time() error = nil, want non-nil for invalid date")
	}

	// Explicit UTC timezone ("Z").
	dt = DateTime{DateString: "2020-06-15", TimeString: "13:45:30", TimeZoneString: "Z"}
	got, err = dt.Time()
	if err != nil {
		t.Fatalf("Time() error: %v", err)
	}
	if want := time.Date(2020, 6, 15, 13, 45, 30, 0, time.UTC); !got.Equal(want) {
		t.Errorf("Time() = %v, want %v", got, want)
	}

	// Non-UTC offset is applied rather than ignored.
	dt = DateTime{DateString: "2020-06-15", TimeString: "13:45:30", TimeZoneString: "-04:00"}
	got, err = dt.Time()
	if err != nil {
		t.Fatalf("Time() error: %v", err)
	}
	if want := time.Date(2020, 6, 15, 17, 45, 30, 0, time.UTC); !got.Equal(want) {
		t.Errorf("Time() = %v, want %v", got, want)
	}
	if got.Format(time.RFC3339) != "2020-06-15T13:45:30-04:00" {
		t.Errorf("Time() = %q, want offset preserved", got.Format(time.RFC3339))
	}

	// A malformed timezone is ignored, not treated as an error: the date still
	// parses as UTC rather than degrading to an error/default.
	dt = DateTime{DateString: "2020-06-15", TimeString: "13:45:30", TimeZoneString: "+0200"}
	got, err = dt.Time()
	if err != nil {
		t.Fatalf("Time() error: %v", err)
	}
	if want := time.Date(2020, 6, 15, 13, 45, 30, 0, time.UTC); !got.Equal(want) {
		t.Errorf("Time() = %v, want %v", got, want)
	}
}

func TestAgreementStartEndDate(t *testing.T) {
	a := &Agreement{
		Dates: []*DateTime{
			{Type: DateTimeAgreementStartDate, DateString: "2019-05-06"},
			{Type: DateTimeAgreementEndDate, DateString: "2022-07-08"},
		},
	}
	if got, want := a.StartDate(), time.Date(2019, 5, 6, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("StartDate() = %v, want %v", got, want)
	}
	if got, want := a.EndDate(), time.Date(2022, 7, 8, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("EndDate() = %v, want %v", got, want)
	}

	// No dates -> defaults.
	empty := &Agreement{}
	if got := empty.StartDate(); !got.Equal(DefaultStartDate) {
		t.Errorf("StartDate() = %v, want default %v", got, DefaultStartDate)
	}
	if got := empty.EndDate(); !got.Equal(DefaultEndDate) {
		t.Errorf("EndDate() = %v, want default %v", got, DefaultEndDate)
	}

	// Invalid dates -> defaults.
	bad := &Agreement{
		Dates: []*DateTime{
			{Type: DateTimeAgreementStartDate, DateString: "bad"},
			{Type: DateTimeAgreementEndDate, DateString: "bad"},
		},
	}
	if got := bad.StartDate(); !got.Equal(DefaultStartDate) {
		t.Errorf("StartDate() = %v, want default %v", got, DefaultStartDate)
	}
	if got := bad.EndDate(); !got.Equal(DefaultEndDate) {
		t.Errorf("EndDate() = %v, want default %v", got, DefaultEndDate)
	}
}

func TestMimeInfoSources(t *testing.T) {
	mi := &MimeInfo{
		Mimes: []*Mime{
			{Purpose: MimePurposeThumbnail, Source: "thumb.jpg"},
			{Purpose: MimePurposeNormal, Source: "normal.jpg"},
			{Purpose: MimePurposeDetail, Source: "detail.jpg"},
			{Purpose: MimePurposeDataSheet, Source: "sheet.pdf"},
			{Purpose: MimePurposeLogo, Source: "logo.png"},
		},
	}
	if got, want := mi.ThumbnailSource(), "thumb.jpg"; got != want {
		t.Errorf("ThumbnailSource() = %q, want %q", got, want)
	}
	if got, want := mi.NormalSource(), "normal.jpg"; got != want {
		t.Errorf("NormalSource() = %q, want %q", got, want)
	}
	if got, want := mi.DetailSource(), "detail.jpg"; got != want {
		t.Errorf("DetailSource() = %q, want %q", got, want)
	}
	if got, want := mi.DataSheetSource(), "sheet.pdf"; got != want {
		t.Errorf("DataSheetSource() = %q, want %q", got, want)
	}
	if got, want := mi.LogoSource(), "logo.png"; got != want {
		t.Errorf("LogoSource() = %q, want %q", got, want)
	}

	// Empty MimeInfo returns empty strings for every source.
	empty := &MimeInfo{}
	if got := empty.ThumbnailSource(); got != "" {
		t.Errorf("ThumbnailSource() = %q, want empty", got)
	}
	if got := empty.NormalSource(); got != "" {
		t.Errorf("NormalSource() = %q, want empty", got)
	}
	if got := empty.DetailSource(); got != "" {
		t.Errorf("DetailSource() = %q, want empty", got)
	}
	if got := empty.DataSheetSource(); got != "" {
		t.Errorf("DataSheetSource() = %q, want empty", got)
	}
	if got := empty.LogoSource(); got != "" {
		t.Errorf("LogoSource() = %q, want empty", got)
	}
}

func TestUDXGetNotFound(t *testing.T) {
	var fields UserDefinedExtensionFields
	fields.Add("A", "1")
	if _, ok := fields.Get("MISSING"); ok {
		t.Error("Get(MISSING) ok = true, want false")
	}
	if _, ok := fields.GetInnerXML("MISSING"); ok {
		t.Error("GetInnerXML(MISSING) ok = true, want false")
	}
}

func TestClassificationSystemLevelNameSpecialCharsRoundTrip(t *testing.T) {
	const value = "Tools & Co <Main> \"group\""
	in := ClassificationSystemLevelName{Level: 1, Value: value}

	buf, err := xml.Marshal(&in)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if strings.Contains(string(buf), "Tools & Co") {
		t.Errorf("Marshal() produced unescaped output: %s", buf)
	}

	var out ClassificationSystemLevelName
	if err := xml.Unmarshal(buf, &out); err != nil {
		t.Fatalf("Unmarshal() error = %v, xml = %s", err, buf)
	}
	if out.Value != value {
		t.Errorf("Value = %q, want %q", out.Value, value)
	}
}

func TestPriceFlagSpecialCharsRoundTrip(t *testing.T) {
	const value = "true & valid <x>"
	in := PriceFlag{Type: PriceFlagInclFreight, Value: value}

	buf, err := xml.Marshal(&in)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if strings.Contains(string(buf), "true & valid") {
		t.Errorf("Marshal() produced unescaped output: %s", buf)
	}

	var out PriceFlag
	if err := xml.Unmarshal(buf, &out); err != nil {
		t.Fatalf("Unmarshal() error = %v, xml = %s", err, buf)
	}
	if out.Value != value {
		t.Errorf("Value = %q, want %q", out.Value, value)
	}
}
