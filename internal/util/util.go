package util

import (
	"strings"
)

// IndexNonQuoted returns the index of the first non-quoted occurrence of
// substr in s.
func IndexNonQuoted(s, substr string) int {
	quoteCount := 0
	for i, c := range s {
		if i+len(substr) > len(s) {
			break
		}

		if c == 34 {
			quoteCount++
		}

		if quoteCount%2 == 0 && s[i:i+len(substr)] == substr {
			return i
		}
	}

	return -1
}

const (
	tokenNL = "LINEFEED7540c64c"
	tokenCR = "CARRIAGERETURNa1cde9f4"
)

// TokenizeTerminators replaces newline and carriage return runes with tokens.
// This can be used as a text preprocessor to override default csv.Reader
// record termination handling.
func TokenizeTerminators(s string) string {
	s = strings.Replace(s, "\n", tokenNL, -1)
	return strings.Replace(s, "\r", tokenCR, -1)
}

// ResetTerminatorTokens replaces newline and carriage return tokens with
// their original runes. This is the inverse of TokenizeTerminators.
func ResetTerminatorTokens(ss []string) []string {
	for i, s := range ss {
		s = strings.Replace(s, tokenNL, "\n", -1)
		ss[i] = strings.Replace(s, tokenCR, "\r", -1)
	}
	return ss
}

// IsExtraneousQuoteError returns true if err is a csv.ErrQuote
func IsExtraneousQuoteError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "extraneous or missing \" in quoted-field")
}

// IsBareQuoteError returns true if err is a csv.ErrBareQuote
func IsBareQuoteError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "bare \" in non-quoted-field")
}

// Panic will panic if err is not nil.
func Panic(err error) {
	if err != nil {
		panic(err)
	}
}
