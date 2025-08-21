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
		{"valid date", "2023年1月15日", true},
		{"invalid date", "2023年2月30日", false},
		{"valid date with space", " 2024年12月1日 ", true},
		{"invalid format", "2023-01-15", false},
		{"era date", "平成30年2月1日", false}, // Known limitation
		{"empty", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidateDate(tc.date); got != tc.expected {
				t.Errorf("expected %v, but got %v for date %s", tc.expected, got, tc.date)
			}
		})
	}
}
