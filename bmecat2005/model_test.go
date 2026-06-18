package bmecat2005

import (
	"testing"
	"time"
)

func TestProductFeaturesClassification(t *testing.T) {
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
			pf := ProductFeatures{FeatureSystemName: tt.system}
			if got := pf.IsEclass(); got != tt.isEclass {
				t.Errorf("IsEclass() = %v, want %v", got, tt.isEclass)
			}
			if got := pf.IsUnspsc(); got != tt.isUnspsc {
				t.Errorf("IsUnspsc() = %v, want %v", got, tt.isUnspsc)
			}
			if got := pf.Version(); got != tt.version {
				t.Errorf("Version() = %q, want %q", got, tt.version)
			}
		})
	}
}

func TestProductPriceDetailsValidDates(t *testing.T) {
	ppd := &ProductPriceDetails{
		Dates: []*DateTime{
			{Type: DateTimeValidStartDate, DateString: "2020-01-02"},
			{Type: DateTimeValidEndDate, DateString: "2021-03-04"},
		},
	}
	if got, want := ppd.ValidStartDate(), time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("ValidStartDate() = %v, want %v", got, want)
	}
	if got, want := ppd.ValidEndDate(), time.Date(2021, 3, 4, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("ValidEndDate() = %v, want %v", got, want)
	}
}

func TestProductPriceDetailsValidDatesDefaults(t *testing.T) {
	ppd := &ProductPriceDetails{}
	if got := ppd.ValidStartDate(); !got.Equal(DefaultStartDate) {
		t.Errorf("ValidStartDate() = %v, want default %v", got, DefaultStartDate)
	}
	if got := ppd.ValidEndDate(); !got.Equal(DefaultEndDate) {
		t.Errorf("ValidEndDate() = %v, want default %v", got, DefaultEndDate)
	}
}

func TestProductPriceDetailsInvalidDatesFallBackToDefault(t *testing.T) {
	ppd := &ProductPriceDetails{
		Dates: []*DateTime{
			{Type: DateTimeValidStartDate, DateString: "not-a-date"},
			{Type: DateTimeValidEndDate, DateString: "also-bad"},
		},
	}
	if got := ppd.ValidStartDate(); !got.Equal(DefaultStartDate) {
		t.Errorf("ValidStartDate() = %v, want default %v", got, DefaultStartDate)
	}
	if got := ppd.ValidEndDate(); !got.Equal(DefaultEndDate) {
		t.Errorf("ValidEndDate() = %v, want default %v", got, DefaultEndDate)
	}
}

func TestProductPriceDetailsIsDailyPrice(t *testing.T) {
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
		ppd := ProductPriceDetails{DailyPriceString: tt.value}
		if got := ppd.IsDailyPrice(); got != tt.want {
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
			{Purpose: MimePurposeIcon, Source: "icon.png"},
			{Purpose: MimePurposeSafetyDataSheet, Source: "safety.pdf"},
		},
	}
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"Thumbnail", mi.ThumbnailSource(), "thumb.jpg"},
		{"Normal", mi.NormalSource(), "normal.jpg"},
		{"Detail", mi.DetailSource(), "detail.jpg"},
		{"DataSheet", mi.DataSheetSource(), "sheet.pdf"},
		{"Logo", mi.LogoSource(), "logo.png"},
		{"Icon", mi.IconSource(), "icon.png"},
		{"SafetyDataSheet", mi.SafetyDataSheetSource(), "safety.pdf"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%sSource() = %q, want %q", c.name, c.got, c.want)
		}
	}

	// Empty MimeInfo returns empty strings for every source.
	empty := &MimeInfo{}
	emptyCases := []struct {
		name string
		got  string
	}{
		{"Thumbnail", empty.ThumbnailSource()},
		{"Normal", empty.NormalSource()},
		{"Detail", empty.DetailSource()},
		{"DataSheet", empty.DataSheetSource()},
		{"Logo", empty.LogoSource()},
		{"Icon", empty.IconSource()},
		{"SafetyDataSheet", empty.SafetyDataSheetSource()},
	}
	for _, c := range emptyCases {
		if c.got != "" {
			t.Errorf("%sSource() = %q, want empty", c.name, c.got)
		}
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
