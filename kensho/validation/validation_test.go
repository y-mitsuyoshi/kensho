package validation

import "testing"

func TestValidateDriverLicenseNumber(t *testing.T) {
	testCases := []struct {
		name     string
		number   string
		expected bool
	}{
		// Based on the dummy weights in the function, for "12345678901", the check digit is 2.
		{"valid", "123456789012", true},
		{"invalid check digit", "123456789013", false},
		{"invalid length", "12345", false},
		{"invalid char", "12345678901a", false},
		{"empty", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidateDriverLicenseNumber(tc.number); got != tc.expected {
				t.Errorf("expected %v, but got %v for number %s", tc.expected, got, tc.number)
			}
		})
	}
}

func TestValidateMyNumber(t *testing.T) {
	testCases := []struct {
		name     string
		number   string
		expected bool
	}{
		// A known valid number and check digit
		{"valid", "123456789018", true},
		{"invalid check digit", "123456789012", false},
		{"invalid length", "12345", false},
		{"invalid char", "12345678901a", false},
		{"empty", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidateMyNumber(tc.number); got != tc.expected {
				t.Errorf("expected %v, but got %v for number %s", tc.expected, got, tc.number)
			}
		})
	}
}

func TestValidateDate(t *testing.T) {
	testCases := []struct {
		name     string
		date     string
		expected bool
	}{
		// Valid Gregorian Dates
		{"valid date with kanji", "2023年1月15日", true},
		{"valid date with space", " 2024年12月1日 ", true},
		{"valid YYYY-MM-DD", "2023-01-15", true},

		// Invalid Gregorian Dates
		{"invalid date", "2023年2月30日", false},
		{"invalid format", "2023/01/15", false},
		{"empty", "", false},

		// Valid Japanese Era Dates
		{"era date heisei", "平成30年2月1日", true},
		{"era date reiwa", "令和3年9月22日", true},
		{"era date showa", "昭和50年10月8日", true},
		{"era date gannen", "令和元年5月1日", true},
		{"user birth_date", "平成2年10月8日", true},
		{"user issue_date", "令和03年09月22日", true},
		{"user expiry_date", "令和08年11月08日", true},
		{"leading zero in day", "平成元年1月01日", true},

		// Invalid Japanese Era Dates
		{"invalid era day", "令和10年2月30日", false},
		{"invalid era month", "平成10年13月1日", false},
		{"invalid era name", "試験10年1月1日", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidateDate(tc.date); got != tc.expected {
				t.Errorf("expected %v, but got %v for date %s", tc.expected, got, tc.date)
			}
		})
	}
}
