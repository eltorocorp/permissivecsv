package util_test

import (
	"testing"

	"github.com/eltorocorp/permissivecsv/internal/util"
	"github.com/stretchr/testify/assert"
)

func Test_IndexNonQuoted(t *testing.T) {
	tests := []struct {
		name          string
		s             string
		substr        string
		expectedIndex int
	}{
		{
			name:          "negative one if not found",
			s:             "abc",
			substr:        "def",
			expectedIndex: -1,
		},
		{
			name:          "non-quoted found",
			s:             "abc",
			substr:        "bc",
			expectedIndex: 1,
		},
		{
			name:          "non-quote found with quoted",
			s:             "a\"bc\"bc",
			substr:        "bc",
			expectedIndex: 5,
		},
		{
			name:          "non-quote not found if only quoted",
			s:             "a\"bc\"",
			substr:        "bc",
			expectedIndex: -1,
		},
	}

	for _, test := range tests {
		testFn := func(t *testing.T) {
			i := util.IndexNonQuoted(test.s, test.substr)
			assert.Equal(t, test.expectedIndex, i)
		}
		t.Run(test.name, testFn)
	}
}
