package permissivecsv_test

import (
	"strings"
	"testing"

	"github.com/eltorocorp/permissivecsv"
	"github.com/stretchr/testify/assert"
)

func Test_Scan(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		result [][]string
	}{
		{
			name:   "single empty record",
			input:  "",
			result: [][]string{[]string{""}},
		},
		{
			name:   "single record",
			input:  "1,2,3",
			result: [][]string{[]string{"1", "2", "3"}},
		},
		{
			name:  "unix terminators",
			input: "a,a,a\nb,b,b\nc,c,c",
			result: [][]string{
				[]string{"a", "a", "a"},
				[]string{"b", "b", "b"},
				[]string{"c", "c", "c"},
			},
		},
		{
			name:  "DOS terminators",
			input: "a,a,a\r\nb,b,b\r\nc,c,c",
			result: [][]string{
				[]string{"a", "a", "a"},
				[]string{"b", "b", "b"},
				[]string{"c", "c", "c"},
			},
		},
		{
			name:  "carriage return as terminator",
			input: "a,a,a\rb,b,b\rc,c,c",
			result: [][]string{
				[]string{"a", "a", "a"},
				[]string{"b", "b", "b"},
				[]string{"c", "c", "c"},
			},
		},
		{
			name:  "inverted DOS terminator",
			input: "a,a,a\n\rb,b,b\n\rc,c,c",
			result: [][]string{
				[]string{"a", "a", "a"},
				[]string{"b", "b", "b"},
				[]string{"c", "c", "c"},
			},
		},
		{
			name:  "dangling terminator",
			input: "a,a,a\nb,b,b\nc,c,c\n",
			result: [][]string{
				[]string{"a", "a", "a"},
				[]string{"b", "b", "b"},
				[]string{"c", "c", "c"},
				[]string{"", "", ""},
			},
		},
		{
			// permissivecsv presumes that the first record of a file is correct
			// thus, if the first thing it encounters is a terminator, it
			// concludes that the head record is just a single empty field.
			// Since it assumes that is correct, it presumes each subsequent
			// record has too many fields, and truncates them. This is the
			// expected behavior of the system given the assumptions it makes.
			name:  "loitering terminator",
			input: "\na,a,a\nb,b,b\nc,c,c",
			result: [][]string{
				[]string{""},
				[]string{"a"},
				[]string{"b"},
				[]string{"c"},
			},
		},
	}

	for _, test := range tests {
		testFn := func(t *testing.T) {
			r := strings.NewReader(test.input)
			s := permissivecsv.NewScanner(r, permissivecsv.HeaderCheckAssumeNoHeader)
			result := [][]string{}
			for s.Scan() {
				result = append(result, s.CurrentRecord())
			}
			assert.ElementsMatch(t, test.result, result)
		}
		t.Run(test.name, testFn)
	}
}
