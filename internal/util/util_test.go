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
			name:          "found if no quoted",
			s:             "abc",
			substr:        "bc",
			expectedIndex: 1,
		},
		{
			name:          "found if quoted first",
			s:             "a\"bc\"bc",
			substr:        "bc",
			expectedIndex: 5,
		},
		{
			name:          "found if quoted second",
			s:             "abc\"bc\"",
			substr:        "bc",
			expectedIndex: 1,
		},
		{
			name:          "not found if only quoted",
			s:             "a\"bc\"",
			substr:        "bc",
			expectedIndex: -1,
		},
		{
			name:          "deep substr",
			s:             "\"bcbcbc\"bc",
			substr:        "bc",
			expectedIndex: 8,
		},
		{
			name:          "special characters are handled",
			s:             "\"*\"*",
			substr:        "*",
			expectedIndex: 3,
		},
		{
			name:          "newlines ok",
			s:             "\"\n\"b,b,b\nc,c,c",
			substr:        "\n",
			expectedIndex: 8,
		},
		{
			name:          "unix terminator between quoted fields",
			s:             "\"AAA\"\n\"AAA\"",
			substr:        "\n",
			expectedIndex: 5,
		},
		{
			name:          "dos terminator between quoted fields",
			s:             "\"AAA\"\r\n\"AAA\"",
			substr:        "\r\n",
			expectedIndex: 5,
		},
		{
			name:          "dos terminator at end",
			s:             "\"AAA\"\r\n",
			substr:        "\r\n",
			expectedIndex: 5,
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
