package utils

import (
	"strings"
)

// SanitizeString trims the spaces from the string and makes all the characters lowercase and ensures that there is only one space between words.
func SanitizeString(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ToLower(text)
	text = strings.Join(strings.Fields(text), " ")

	return text
}
