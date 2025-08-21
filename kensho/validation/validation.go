package validation

import (
	"strconv"
	"strings"
	"time"
)

// ValidateDriverLicenseNumber validates the check digit of a Japanese driver's license number.
func ValidateDriverLicenseNumber(number string) bool {
	number = strings.ReplaceAll(number, " ", "")
	number = strings.ReplaceAll(number, "第", "")
	number = strings.ReplaceAll(number, "号", "")
	if len(number) != 12 {
		return false
	}

	// The check digit calculation is a weighted sum modulo 11.
	// This is a common algorithm, but the specific weights for Japanese driver's licenses
	// are not publicly documented. For this example, we'll use a simplified checksum logic.
	// A real implementation would need the official algorithm.
	sum := 0
	for i := 0; i < 11; i++ {
		digit, err := strconv.Atoi(string(number[i]))
		if err != nil {
			return false
		}
		// Example weighting (not the real one)
		weights := []int{5, 4, 3, 2, 7, 6, 5, 4, 3, 2, 1}
		if i < len(weights) { // defensive check
			sum += digit * weights[i]
		}
	}

	checkDigit, err := strconv.Atoi(string(number[11]))
	if err != nil {
		return false
	}

	// Simplified check digit logic
	return (11-(sum%11))%10 == checkDigit
}

// ValidateMyNumber validates the check digit of a Japanese Individual Number (My Number).
func ValidateMyNumber(number string) bool {
	number = strings.ReplaceAll(number, "-", "")
	if len(number) != 12 {
		return false
	}

	n, err := strconv.Atoi(number[:11])
	if err != nil {
		return false
	}

	// Calculation logic based on public specification
	sum := 0
	for i := 0; i < 11; i++ {
		digit := n % 10
		n /= 10
		var weight int
		if i < 6 {
			weight = i + 2
		} else {
			weight = i - 4
		}
		sum += digit * weight
	}

	remainder := sum % 11
	checkDigit := 0
	if remainder > 1 {
		checkDigit = 11 - remainder
	}

	lastDigit, err := strconv.Atoi(string(number[11]))
	if err != nil {
		return false
	}

	return checkDigit == lastDigit
}

// ValidateDate checks if a given date string is a real calendar date.
// It expects the format YYYY年MM月DD日 or similar, and does not handle Japanese eras.
func ValidateDate(dateStr string) bool {
	dateStr = strings.TrimSpace(dateStr)
	if !strings.Contains(dateStr, "年") || !strings.Contains(dateStr, "月") || !strings.Contains(dateStr, "日") {
		return false
	}

	// This doesn't handle era names yet.
	// A more robust solution would involve a proper date parsing library that understands Japanese eras.
	dateStr = strings.ReplaceAll(dateStr, "年", "-")
	dateStr = strings.ReplaceAll(dateStr, "月", "-")
	dateStr = strings.ReplaceAll(dateStr, "日", "")

	// This is a simplified check and won't handle Japanese eras correctly.
	// For example "昭和60年1月1日" would fail.
	// A real implementation would require a library to convert eras to Gregorian years first.
	_, err := time.Parse("2006-1-2", dateStr)
	return err == nil
}
