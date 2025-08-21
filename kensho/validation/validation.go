package validation

import (
	"fmt"
	"regexp"
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

var eraStartYears = map[string]int{
	"令和": 2019,
	"平成": 1989,
	"昭和": 1926,
	"大正": 1912,
	"明治": 1868,
}

var warekiRegex = regexp.MustCompile(fmt.Sprintf(`(%s)(\d+|元)年(\d+)月(\d+)日`, strings.Join(getEraKeys(), "|")))

func getEraKeys() []string {
	keys := make([]string, 0, len(eraStartYears))
	for k := range eraStartYears {
		keys = append(keys, k)
	}
	return keys
}

// warekiToTime converts a Japanese era date string to a time.Time object.
func warekiToTime(warekiStr string) (time.Time, error) {
	warekiStr = strings.TrimSpace(warekiStr)
	// Replace "元年" (gan-nen) with "1年" for easier parsing.
	warekiStr = strings.Replace(warekiStr, "元年", "1年", 1)

	matches := warekiRegex.FindStringSubmatch(warekiStr)
	if len(matches) != 5 {
		return time.Time{}, fmt.Errorf("invalid wareki format: %s", warekiStr)
	}

	era := matches[1]
	yearStr := matches[2]
	monthStr := matches[3]
	dayStr := matches[4]

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid year: %s", yearStr)
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid month: %s", monthStr)
	}

	day, err := strconv.Atoi(dayStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day: %s", dayStr)
	}

	startYear, ok := eraStartYears[era]
	if !ok {
		return time.Time{}, fmt.Errorf("unknown era: %s", era)
	}

	gregorianYear := startYear + year - 1

	// Let time.Parse validate the date's existence (e.g., Feb 30)
	dateStr := fmt.Sprintf("%d-%02d-%02d", gregorianYear, month, day)
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("date validation failed: %w", err)
	}

	return t, nil
}

// ValidateDate checks if a given date string is a real calendar date.
// It handles Japanese eras (e.g., 令和, 平成) and standard YYYY-MM-DD formats.
func ValidateDate(dateStr string) bool {
	dateStr = strings.TrimSpace(dateStr)

	// Try parsing as a Japanese era date first.
	if _, err := warekiToTime(dateStr); err == nil {
		return true
	}

	// Fallback for non-era dates or other formats
	// This part handles formats like YYYY年MM月DD日 (without era) or YYYY-MM-DD
	dateStr = strings.ReplaceAll(dateStr, "年", "-")
	dateStr = strings.ReplaceAll(dateStr, "月", "-")
	dateStr = strings.ReplaceAll(dateStr, "日", "")

	// Try a few common layouts
	layouts := []string{"2006-1-2", "2006-01-02"}
	for _, layout := range layouts {
		if _, err := time.Parse(layout, dateStr); err == nil {
			return true
		}
	}

	return false
}
