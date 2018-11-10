package permissivecsv_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/eltorocorp/permissivecsv"
	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"
)

var ErrReader = errors.New("arbitrary reader error")

// BadReader returns ErrReader on the first Read call.
func BadReader(r io.ReadSeeker) io.ReadSeeker { return &badReader{r} }

type badReader struct {
	r io.ReadSeeker
}

func (r *badReader) Read(p []byte) (int, error) {
	return 0, ErrReader
}

func (r *badReader) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

func Test_Reader(t *testing.T) {
	tests := []struct {
		name             string
		reader           io.ReadSeeker
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
			expScans:         0,
			expCurrentRecord: []string{},
		},
		{
			// If a reader reports an error, the scanner will stop after the
			// current record. If possible, whichever value is loaded as the
			// current record in the underlaying io.Scanner is returned, but
			// this is typically an empty string. Since no more reads are
			// possible, permissivecsv considers this the end of the file.
			name:             "reader returns an error",
			reader:           BadReader(strings.NewReader("a\nb\nc")),
			expScans:         0,
			expCurrentRecord: []string{},
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
			result: [][]string{},
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
			name:  "dangling terminator",
			input: "a,a,a\nb,b,b\nc,c,c\n\n",
			result: [][]string{
				[]string{"a", "a", "a"},
				[]string{"b", "b", "b"},
				[]string{"c", "c", "c"},
			},
		},
		{
			name:  "leading terminator",
			input: "\r\n\r\n\r\n\r\na,a,a\r\nb,b,b\r\nc,c,c",
			result: [][]string{
				[]string{"a", "a", "a"},
				[]string{"b", "b", "b"},
				[]string{"c", "c", "c"},
			},
		},
		{
			name:  "empty records",
			input: "a,a,a\nb,b,b\n\n\nc,c,c",
			result: [][]string{
				[]string{"a", "a", "a"},
				[]string{"b", "b", "b"},
				[]string{"c", "c", "c"},
			},
		},
		{
			// if the first thing permissivecsv encounters is an empty field
			// followed by a terminator, it assumes that to be correct. All
			// subsequent records are expected to also only have one field, and
			// are truncated as necessary.
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
			// permissivecsv doesn't care about mixing terminators.
			name:  "mixed terminators",
			input: "a,a\nb,b\nc,c\r\nd,d\ne,e\n\rf,f",
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
			input: "a,a,a\nb\"\"b,b,b\nc,c,c",
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
		data io.ReadSeeker
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
		{
			name:      "truncated record",
			data:      strings.NewReader("a,b,c\nd,e,f,g"),
			scanLimit: -1,
			expSummary: &permissivecsv.ScanSummary{
				RecordCount:     2,
				AlterationCount: 1,
				EOF:             true,
				Err:             nil,
				Alterations: []*permissivecsv.Alteration{
					&permissivecsv.Alteration{
						RecordOrdinal:         2,
						OriginalData:          "d,e,f,g",
						ResultingRecord:       []string{"d", "e", "f"},
						AlterationDescription: permissivecsv.AltTruncatedRecord,
					},
				},
			},
		},
		{
			name:      "padded record",
			data:      strings.NewReader("a,b,c\nd,e"),
			scanLimit: -1,
			expSummary: &permissivecsv.ScanSummary{
				RecordCount:     2,
				AlterationCount: 1,
				EOF:             true,
				Err:             nil,
				Alterations: []*permissivecsv.Alteration{
					&permissivecsv.Alteration{
						RecordOrdinal:         2,
						OriginalData:          "d,e",
						ResultingRecord:       []string{"d", "e", ""},
						AlterationDescription: permissivecsv.AltPaddedRecord,
					},
				},
			},
		},
		{
			name:      "EOF false before end of file",
			data:      strings.NewReader("a\n\b\nc"),
			scanLimit: 1,
			expSummary: &permissivecsv.ScanSummary{
				RecordCount:     1,
				AlterationCount: 0,
				EOF:             false,
				Err:             nil,
				Alterations:     []*permissivecsv.Alteration{},
			},
		},
	}

	for _, test := range tests {
		testFn := func(t *testing.T) {
			s := permissivecsv.NewScanner(test.data, permissivecsv.HeaderCheckAssumeNoHeader)
			for n := 0; ; n++ {
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

func Test_HeaderCheckCallback(t *testing.T) {
	tests := []struct {
		name            string
		data            string
		scanLimit       int
		expFirstRecord  []string
		expSecondRecord []string
	}{
		{
			name:            "nils before Scan",
			data:            "a,b,c\nd,e,f\ng,h,i",
			scanLimit:       0,
			expFirstRecord:  nil,
			expSecondRecord: nil,
		},
		{
			name:            "1st and 2nd correct on first Scan",
			data:            "a,b,c\nd,e,f\ng,h,i",
			scanLimit:       1,
			expFirstRecord:  []string{"a", "b", "c"},
			expSecondRecord: []string{"d", "e", "f"},
		},
		{
			name:            "scan advanced beyond first record",
			data:            "a,b,c\nd,e,f\ng,h,i",
			scanLimit:       -1,
			expFirstRecord:  nil,
			expSecondRecord: nil,
		},
		{
			name:            "2nd nil if no second record",
			data:            "x,y,z",
			scanLimit:       1,
			expFirstRecord:  []string{"x", "y", "z"},
			expSecondRecord: nil,
		},
	}

	for _, test := range tests {
		testFn := func(t *testing.T) {
			var actualFirstRecord []string
			var actualSecondRecord []string
			headerCheck := func(firstRecord, secondRecord []string) bool {
				actualFirstRecord = firstRecord
				actualSecondRecord = secondRecord
				return false
			}
			r := strings.NewReader(test.data)
			s := permissivecsv.NewScanner(r, headerCheck)
			for n := 0; ; n++ {
				if test.scanLimit >= 0 && n >= test.scanLimit {
					break
				}
				more := s.Scan()
				// actual result of RecordIsHeader isn't pertinant to these test
				// cases
				_ = s.RecordIsHeader()
				if !more {
					break
				}
			}

			if test.expFirstRecord == nil {
				assert.Nil(t, actualFirstRecord, "expected first record to be nil")
			} else {
				assert.Equal(t, test.expFirstRecord, actualFirstRecord)
			}

			if test.expSecondRecord == nil {
				assert.Nil(t, actualSecondRecord, "expected second record to be nil")
			} else {
				assert.Equal(t, test.expSecondRecord, actualSecondRecord)
			}
		}
		t.Run(test.name, testFn)
	}
}

func Test_Partition(t *testing.T) {
	// The partition tests specifically target segment generation capabilities,
	// and presume that the underlaying record splitter is properly identifying
	// terminators and returning raw records to Split as intended.
	tests := []struct {
		name                string
		data                io.ReadSeeker
		recordsPerPartition int
		excludeHeader       bool
		expPartitions       []*permissivecsv.Segment
	}{
		{
			name:                "nil reader",
			data:                nil,
			recordsPerPartition: 10,
			excludeHeader:       false,
			expPartitions:       []*permissivecsv.Segment{},
		},
		{
			name:                "empty file",
			data:                strings.NewReader(""),
			recordsPerPartition: 10,
			excludeHeader:       false,
			expPartitions:       []*permissivecsv.Segment{},
		},
		{
			name:                "one byte long terminator",
			data:                strings.NewReader("a,b\nc,d\ne,f\ng,h\ni,j\nk,l"),
			recordsPerPartition: 2,
			excludeHeader:       false,
			expPartitions: []*permissivecsv.Segment{
				&permissivecsv.Segment{
					Ordinal:     1,
					LowerOffset: 0,
					Length:      8,
				},
				&permissivecsv.Segment{
					Ordinal:     2,
					LowerOffset: 8,
					Length:      8,
				},
				&permissivecsv.Segment{
					Ordinal:     3,
					LowerOffset: 16,
					Length:      7,
				},
			},
		},
		{
			name:                "two byte long terminator",
			data:                strings.NewReader("a,b\r\nc,d\r\ne,f\r\ng,h\r\ni,j\r\nk,l"),
			recordsPerPartition: 2,
			excludeHeader:       false,
			expPartitions: []*permissivecsv.Segment{
				&permissivecsv.Segment{
					Ordinal:     1,
					LowerOffset: 0,
					Length:      10,
				},
				&permissivecsv.Segment{
					Ordinal:     2,
					LowerOffset: 10,
					Length:      10,
				},
				&permissivecsv.Segment{
					Ordinal:     3,
					LowerOffset: 20,
					Length:      8,
				},
			},
		},
		{
			name:                "one byte term with partial final segment",
			data:                strings.NewReader("a,b\nc,d\ne,f\ng,h\ni,j\nk,l\nm,n"),
			recordsPerPartition: 2,
			excludeHeader:       false,
			expPartitions: []*permissivecsv.Segment{
				&permissivecsv.Segment{
					Ordinal:     1,
					LowerOffset: 0,
					Length:      8,
				},
				&permissivecsv.Segment{
					Ordinal:     2,
					LowerOffset: 8,
					Length:      8,
				},
				&permissivecsv.Segment{
					Ordinal:     3,
					LowerOffset: 16,
					Length:      8,
				},
				&permissivecsv.Segment{
					Ordinal:     4,
					LowerOffset: 24,
					Length:      3,
				},
			},
		},
		{
			name:                "two byte term with partial final segment",
			data:                strings.NewReader("a,b\r\nc,d\r\ne,f\r\ng,h\r\ni,j\r\nk,l\r\nm,n"),
			recordsPerPartition: 2,
			excludeHeader:       false,
			expPartitions: []*permissivecsv.Segment{
				&permissivecsv.Segment{
					Ordinal:     1,
					LowerOffset: 0,
					Length:      10,
				},
				&permissivecsv.Segment{
					Ordinal:     2,
					LowerOffset: 10,
					Length:      10,
				},
				&permissivecsv.Segment{
					Ordinal:     3,
					LowerOffset: 20,
					Length:      10,
				},
				&permissivecsv.Segment{
					Ordinal:     4,
					LowerOffset: 30,
					Length:      3,
				},
			},
		},
		{
			name:                "mixed terminators",
			data:                strings.NewReader("a,b\r\nc,d\ne,f\ng,h\ni,j\nk,l\nm,n"),
			recordsPerPartition: 2,
			excludeHeader:       false,
			expPartitions: []*permissivecsv.Segment{
				&permissivecsv.Segment{
					Ordinal:     1,
					LowerOffset: 0,
					Length:      9,
				},
				&permissivecsv.Segment{
					Ordinal:     2,
					LowerOffset: 9,
					Length:      8,
				},
				&permissivecsv.Segment{
					Ordinal:     3,
					LowerOffset: 17,
					Length:      8,
				},
				&permissivecsv.Segment{
					Ordinal:     4,
					LowerOffset: 25,
					Length:      3,
				},
			},
		},
		{
			name:                "variable record lengths",
			data:                strings.NewReader("a,b,c\ndd\nee,ff,gg,h\ni,j"),
			recordsPerPartition: 2,
			excludeHeader:       false,
			expPartitions: []*permissivecsv.Segment{
				&permissivecsv.Segment{
					Ordinal:     1,
					LowerOffset: 0,
					Length:      9,
				},
				&permissivecsv.Segment{
					Ordinal:     2,
					LowerOffset: 9,
					Length:      14,
				},
			},
		},
		{
			name:                "one byte term ignore header",
			data:                strings.NewReader("a,b\nc,d\ne,f\ng,h\ni,j\nk,l\nm,n"),
			recordsPerPartition: 2,
			excludeHeader:       true,
			expPartitions: []*permissivecsv.Segment{
				&permissivecsv.Segment{
					Ordinal:     1,
					LowerOffset: 4,
					Length:      8,
				},
				&permissivecsv.Segment{
					Ordinal:     2,
					LowerOffset: 12,
					Length:      8,
				},
				&permissivecsv.Segment{
					Ordinal:     3,
					LowerOffset: 20,
					Length:      7,
				},
			},
		},
		{
			name:                "two byte term ignore header",
			data:                strings.NewReader("a,b\n\rc,d\n\re,f\n\rg,h\n\ri,j\n\rk,l\n\rm,n"),
			recordsPerPartition: 2,
			excludeHeader:       true,
			expPartitions: []*permissivecsv.Segment{
				&permissivecsv.Segment{
					Ordinal:     1,
					LowerOffset: 5,
					Length:      10,
				},
				&permissivecsv.Segment{
					Ordinal:     2,
					LowerOffset: 15,
					Length:      10,
				},
				&permissivecsv.Segment{
					Ordinal:     3,
					LowerOffset: 25,
					Length:      8,
				},
			},
		},
		{
			name:                "leading terminators",
			data:                strings.NewReader("\n\n\na\nb\nc\nd"),
			recordsPerPartition: 2,
			excludeHeader:       false,
			expPartitions: []*permissivecsv.Segment{
				&permissivecsv.Segment{
					Ordinal:     1,
					LowerOffset: 0,
					Length:      7,
				},
				&permissivecsv.Segment{
					Ordinal:     2,
					LowerOffset: 7,
					Length:      3,
				},
			},
		},
		{
			name:                "dangling terminators",
			data:                strings.NewReader("a\nb\n\n\n"),
			recordsPerPartition: 2,
			excludeHeader:       false,
			expPartitions: []*permissivecsv.Segment{
				&permissivecsv.Segment{
					Ordinal:     1,
					LowerOffset: 0,
					Length:      6,
				},
			},
		},
		// New Cases:
		// trailing terminators are ignored
		// empty records are respected
	}
	for _, test := range tests {
		testFn := func(t *testing.T) {
			s := permissivecsv.NewScanner(test.data, permissivecsv.HeaderCheckAssumeHeaderExists)
			partitions := s.Partition(test.recordsPerPartition, test.excludeHeader)
			diff := deep.Equal(test.expPartitions, partitions)
			if diff != nil {
				for _, d := range diff {
					t.Log(d)
				}
				t.Fail()
			}
		}
		t.Run(test.name, testFn)
	}
}
