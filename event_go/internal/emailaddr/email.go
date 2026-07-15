package emailaddr

import (
	"net/mail"
	"strings"
)

// Normalize accepts only a bare mailbox address and returns its lower-cased form.
// Display names are rejected so stored recovery identities remain deterministic.
func Normalize(value string) (string, bool) {
	value = strings.TrimSpace(value)
	parsed, err := mail.ParseAddress(value)
	if err != nil || !strings.EqualFold(parsed.Address, value) {
		return "", false
	}
	return strings.ToLower(parsed.Address), true
}
