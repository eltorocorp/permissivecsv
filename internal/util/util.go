package util

import (
	"regexp"
)

// IndexNonQuoted returns the index of the first non-quoted occurence of
// substr in s.
func IndexNonQuoted(s, substr string) int {
	substr = regexp.QuoteMeta(substr)

	re := regexp.MustCompile(substr)
	matches := re.FindAllStringIndex(s, -1)

	if len(matches) == 0 {
		return -1
	}

	reQuoted := regexp.MustCompile("\"" + substr + "\"")
	matchesQuoted := reQuoted.FindAllStringIndex(s, -1)

	if len(matchesQuoted) == 0 {
		return matches[0][0]
	}

	if len(matchesQuoted) == len(matches) {
		return -1
	}

	for i := 0; i < len(matchesQuoted); i++ {
		matchesQuoted[i][0]++
		matchesQuoted[i][1]--
	}
	lastQuotedIndex := matchesQuoted[len(matchesQuoted)-1][0]
	return matches[lastQuotedIndex-1][0]
}
