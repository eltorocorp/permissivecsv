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
			// permissivecsv accepts standard unix record terminators
			name:  "unix terminators",
			input: "a,a,a\nb,b,b\nc,c,c",
			result: [][]string{
				[]string{"a", "a", "a"},
				[]string{"b", "b", "b"},
				[]string{"c", "c", "c"},
			},
		},
		{
			// permissivecsv accepts standard DOS record terminators
			name:  "DOS terminators",
			input: "a,a,a\r\nb,b,b\r\nc,c,c",
			result: [][]string{
				[]string{"a", "a", "a"},
				[]string{"b", "b", "b"},
				[]string{"c", "c", "c"},
			},
		},
		{
			// permissivecsv accepts non-standard carriage return terminators
			name:  "carriage return as terminator",
			input: "a,a,a\rb,b,b\rc,c,c",
			result: [][]string{
				[]string{"a", "a", "a"},
				[]string{"b", "b", "b"},
				[]string{"c", "c", "c"},
			},
		},
		{
			// permissivecsv accepts non-standard "inverted DOS" terminators
			name:  "inverted DOS terminator",
			input: "a,a,a\n\rb,b,b\n\rc,c,c",
			result: [][]string{
				[]string{"a", "a", "a"},
				[]string{"b", "b", "b"},
				[]string{"c", "c", "c"},
			},
		},
		{
			// permissivecsv allows records to end with a dangline terminator,
			// and presumes that to mean the record contains an implicit nil
			// record.
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
			// concludes that the head record simply contains zero fields.
			// Since it assumes that is correct, it presumes each subsequent
			// record should also have zero fields, and truncates them. This is
			// the expected behavior of the system given the assumptions it
			// makes.
			name:  "loitering terminator",
			input: "\na,a,a\nb,b,b\nc,c,c",
			result: [][]string{
				[]string{},
				[]string{},
				[]string{},
				[]string{},
			},
		},
		{
			// similar to the "loitering terminator" case above, if the first
			// thing permissivecsv encounters is an empty field followed by a
			// terminator, it assumes that to be correct. All subsequent
			// records are expected to also only have one field, and are
			// truncated as necessary.
			name:  "loitering empty field",
			input: "\"\"\na,a,a\nb,b,b\nc,c,c",
			result: [][]string{
				[]string{""},
				[]string{"a"},
				[]string{"b"},
				[]string{"c"},
			},
		},
		{
			// permissivecsv doesn't care about mixing terminators. It always
			// just uses the first terminator that it encounters.
			// "composite" terminators, such as the DOS (/r/n) and the inverted
			// DOS (/n/r) terminator, take precedence over "atomic" terminators,
			// such as newline ("\n") or carriage return ("\r").
			name:  "mixed terminators",
			input: "a,a\rb,b\n\rc,c\r\nd,d\ne,e\n\rf,f",
			result: [][]string{
				[]string{"a", "a"},
				[]string{"b", "b"},
				[]string{"c", "c"},
				[]string{"d", "d"},
				[]string{"e", "e"},
				[]string{"f", "f"},
			},
		},
		{
			// permissivecsv igores terminators that are quoted
			name:  "ignore quoted",
			input: "a,a,a\n\"\n\"b,b,b\nc,c,c",
			result: [][]string{
				[]string{"a", "a", "a"},
				[]string{"\"\n\"b", "b", "b"},
				[]string{"c", "c", "c"},
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
			assert.Equal(t, test.result, result)
		}
		t.Run(test.name, testFn)
	}
}
