package privacy

import (
	"strings"
	"unicode"
)

func MaskName(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) == 0 {
		return ""
	}
	if len(runes) == 1 {
		return "*"
	}
	masked := len(runes) - 1
	if masked > 3 {
		masked = 3
	}
	return string(runes[0]) + strings.Repeat("*", masked)
}

func MaskContact(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if at := strings.LastIndex(value, "@"); at > 0 && at < len(value)-1 {
		local := []rune(value[:at])
		return string(local[0]) + "***@" + value[at+1:]
	}
	digits := make([]rune, 0, len(value))
	for _, char := range value {
		if unicode.IsDigit(char) {
			digits = append(digits, char)
		}
	}
	if len(digits) >= 7 {
		return string(digits[:3]) + "****" + string(digits[len(digits)-4:])
	}
	runes := []rune(value)
	return string(runes[0]) + "***"
}
