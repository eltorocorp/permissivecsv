package permissivecsv_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/eltorocorp/permissivecsv"
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
			name:  "unix records",
			input: "a,a,a\nb,b,b\nc,c,c",
			result: [][]string{
				[]string{"a", "a", "a"},
				[]string{"b", "b", "b"},
				[]string{"c", "c", "c"},
			},
		},
		{
			name:  "dos records",
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
			name:  "inverted dos terminator",
			input: "a,a,a\n\rb,b,b\n\rc,c,c",
			result: [][]string{
				[]string{"a", "a", "a"},
				[]string{"b", "b", "b"},
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
			assert.ElementsMatch(t, test.result, result)
		}
		t.Run(test.name, testFn)
	}
}
