package permissivecsv_test

import (
	"strings"
	"testing"

	"github.com/eltorocorp/permissivecsv"
	"github.com/stretchr/testify/assert"
)

func Test_ScanAndCurrentRecord(t *testing.T) {
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
			// concludes that the head record contains a single empty field.
			// Since it assumes that is correct, it presumes each subsequent
			// record should also have one field, and truncates them. This is
			// the expected behavior of the system given the assumptions it
			// makes.
			name:  "leading terminator",
			input: "\na,a,a\nb,b,b\nc,c,c",
			result: [][]string{
				[]string{""},
				[]string{"a"},
				[]string{"b"},
				[]string{"c"},
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
			// permissivecsv ignores terminators that are quoted
			name:  "ignore quoted",
			input: "a,a,a\n\"\n\",b,b\nc,c,c",
			result: [][]string{
				[]string{"a", "a", "a"},
				[]string{"\n", "b", "b"},
				[]string{"c", "c", "c"},
			},
		},
		{
			// permissivecsv will nullify the values for all of a record's
			// fields if it encounters a lazy quote. This is the most
			// consistent way to represent data that has been corrupted in this
			// manner. permissivecsv's handling of lazy quotes differs from
			// the stdlib's csv.Reader. csv.Reader will concatenate all of a
			// record's data into a single field if it encounters an unpaired
			// quote. This results in a variation of data output per record that
			// encounters this issue. permissivecsv instead blanks the data
			// for the bad record, and reports the issue via Summary.
			// This reduces the number of data variants output by the Scanner,
			// while allowing the caller to still handle issues as they see fit.
			name:  "lazy quotes",
			input: "a,a,a\n\"b\"b,b,b\nc,c,c",
			result: [][]string{
				[]string{"a", "a", "a"},
				[]string{"", "", ""},
				[]string{"c", "c", "c"},
			},
		},
		{
			// permissivecsv handles extraneous quotes the same way that it
			// handles lazy quotes, by nullifying the field values of the
			// affected record.
			name:  "extraneous quote",
			input: "a,a,a\nb\"\"\"b,b,b\nc,c,c",
			result: [][]string{
				[]string{"a", "a", "a"},
				[]string{"", "", ""},
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

func Test_CurrentRecordNextRecord(t *testing.T) {
	tests := []struct {
		name           string
		data           string
		numberOfScans  int
		expCurrRecords [][]string
		expNextRecords [][]string
		expEOFs        []bool
	}{
		{
			name:           "no records",
			data:           "",
			numberOfScans:  1,
			expCurrRecords: [][]string{[]string{""}},
			expNextRecords: [][]string{},
			expEOFs:        []bool{true},
		},
		{
			// Subsequent calls to current record will continue to return the
			// current record even when no further records are available.
			name:           "single record",
			data:           "a,b,c",
			numberOfScans:  2,
			expCurrRecords: [][]string{[]string{"a", "b", "c"}, []string{"a", "b", "c"}},
			expNextRecords: [][]string{},
			expEOFs:        []bool{true, true},
		},
		{
			name:           "multiple records initial scan",
			data:           "a,b\nc,d\ne,f",
			numberOfScans:  3,
			expCurrRecords: [][]string{[]string{"a", "b"}, []string{"c", "d"}, []string{"e", "f"}},
			expNextRecords: [][]string{[]string{"c", "d"}, []string{"e", "f"}, nil},
			expEOFs:        []bool{false, false, true},
		},
	}

	for _, test := range tests {
		testFn := func(t *testing.T) {
			r := strings.NewReader(test.data)
			s := permissivecsv.NewScanner(r, permissivecsv.HeaderCheckAssumeNoHeader)
			for n := 0; n < test.numberOfScans; n++ {
				s.Scan()
				nextRecord, EOF := s.NextRecord()
				currentRecord := s.CurrentRecord()
				if len(test.expCurrRecords) == 0 {
					assert.Nil(t, currentRecord, "expected currentRecord to be nil")
				} else {
					assert.ElementsMatch(t, test.expCurrRecords[n], currentRecord, "incorrect currentRecord")
				}
				if len(test.expNextRecords) == 0 {
					assert.Nil(t, nextRecord, "expected nextRecord to be nil")
				} else {
					assert.ElementsMatch(t, test.expNextRecords[n], nextRecord, "incorrect nextRecord")
				}
				assert.Equal(t, test.expEOFs[n], EOF, "incorrect EOF")
			}
		}
		t.Run(test.name, testFn)
	}
}
