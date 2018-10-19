package permissivecsv_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/go-test/deep"

	"github.com/eltorocorp/permissivecsv"
	"github.com/stretchr/testify/assert"
)

var ErrReader = errors.New("arbitrary reader error")

// BadReader returns ErrReader on the first Read call.
func BadReader(r io.Reader) io.Reader { return &badReader{r} }

type badReader struct {
	r io.Reader
}

func (r *badReader) Read(p []byte) (int, error) {
	return 0, ErrReader
}

func Test_Reader(t *testing.T) {
	tests := []struct {
		name             string
		reader           io.Reader
		expScans         int
		expCurrentRecord []string
	}{
		{
			name:             "reader is nil",
			reader:           nil,
			expScans:         0,
			expCurrentRecord: nil,
		},
		{
			name:             "reader is not nil",
			reader:           strings.NewReader(""),
			expScans:         1,
			expCurrentRecord: []string{""},
		},
		{
			// If a reader reports an error, the scanner will stop after the
			// current record. If possible, whichever value is loaded as the
			// current record in the underlaying io.Scanner is returned, but
			// this is typically an empty string. Since no more reads are
			// possible, permissivecsv considers this the end of the file.
			name:             "reader returns an error",
			reader:           BadReader(strings.NewReader("a\nb\nc")),
			expScans:         1,
			expCurrentRecord: []string{""},
		},
	}

	for _, test := range tests {
		testFn := func(t *testing.T) {
			s := permissivecsv.NewScanner(test.reader, permissivecsv.HeaderCheckAssumeNoHeader)
			n := 0
			for s.Scan() {
				n++
			}
			currentRecord := s.CurrentRecord()
			assert.Equal(t, test.expScans, n, "expected scans is incorrect")
			assert.ElementsMatch(t, test.expCurrentRecord, currentRecord, "current record is incorrect")
		}
		t.Run(test.name, testFn)
	}

}

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
			// fields if it encounters a bare quote. This is the most
			// consistent way to represent data that has been corrupted in this
			// manner. permissivecsv's handling of bare quotes differs from
			// the stdlib's csv.Reader. csv.Reader will concatenate all of a
			// record's data into a single field if it encounters an unpaired
			// quote. This results in a variation of data output per record that
			// encounters this issue. permissivecsv instead blanks the data
			// for the bad record, and reports the issue via Summary.
			// This reduces the number of data variants output by the Scanner,
			// while allowing the caller to still handle issues as they see fit.
			name:  "bare quotes",
			input: "a,a,a\n\"b\"b,b,b\nc,c,c",
			result: [][]string{
				[]string{"a", "a", "a"},
				[]string{"", "", ""},
				[]string{"c", "c", "c"},
			},
		},
		{
			// permissivecsv handles extraneous quotes the same way that it
			// handles bare quotes, by nullifying the field values of the
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

func Test_Summary(t *testing.T) {
	tests := []struct {
		name string
		data io.Reader
		// scanLimit caps the number of times the test fixture will
		// call Scan. -1 will call Scan until it returns false.
		scanLimit  int
		expSummary *permissivecsv.ScanSummary
	}{
		{
			name:       "summary nil before Scan called",
			data:       strings.NewReader("a,b,c"),
			scanLimit:  0,
			expSummary: nil,
		},
		{
			name:      "nil reader",
			data:      nil,
			scanLimit: -1,
			expSummary: &permissivecsv.ScanSummary{
				RecordCount:     -1,
				AlterationCount: -1,
				EOF:             false,
				Err:             permissivecsv.ErrReaderIsNil,
				Alterations:     []*permissivecsv.Alteration{},
			},
		},
		{
			name:      "extraneous quotes",
			data:      strings.NewReader("\""),
			scanLimit: -1,
			expSummary: &permissivecsv.ScanSummary{
				RecordCount:     1,
				AlterationCount: 1,
				EOF:             true,
				Err:             nil,
				Alterations: []*permissivecsv.Alteration{
					&permissivecsv.Alteration{
						RecordOrdinal:         1,
						OriginalData:          "\"",
						ResultingRecord:       []string{},
						AlterationDescription: permissivecsv.AltExtraneousQuote,
					},
				},
			},
		},
		{
			name:      "bare quote",
			data:      strings.NewReader("a\nb\""),
			scanLimit: -1,
			expSummary: &permissivecsv.ScanSummary{
				RecordCount:     2,
				AlterationCount: 1,
				EOF:             true,
				Err:             nil,
				Alterations: []*permissivecsv.Alteration{
					&permissivecsv.Alteration{
						RecordOrdinal:         2,
						OriginalData:          "b\"",
						ResultingRecord:       []string{""},
						AlterationDescription: permissivecsv.AltBareQuote,
					},
				},
			},
		},
	}

	for _, test := range tests {
		testFn := func(t *testing.T) {
			s := permissivecsv.NewScanner(test.data, permissivecsv.HeaderCheckAssumeNoHeader)
			for n := 1; ; n++ {
				if test.scanLimit >= 0 && n >= test.scanLimit {
					break
				}
				more := s.Scan()
				if !more {
					break
				}
			}
			summary := s.Summary()
			if test.expSummary == nil {
				assert.Nil(t, summary)
			} else {
				diff := deep.Equal(summary, test.expSummary)
				if diff != nil {
					t.Error(diff)
				}
			}
		}
		t.Run(test.name, testFn)
	}
}
