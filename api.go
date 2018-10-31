// Package permissivecsv provides facilties for permissively reading
// non-standards-compliant csv files.
package permissivecsv

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"text/template"

	"github.com/eltorocorp/permissivecsv/internal/linesplit"
	"github.com/eltorocorp/permissivecsv/internal/util"
)

var (
	// ErrReaderIsNil is returned in the Summary if Scan is called but the
	// reader that the Scanner was initialized with is nil.
	ErrReaderIsNil = fmt.Errorf("reader is nil")
)

const (
	// AltBareQuote is the description for bare-quote record alterations.
	AltBareQuote = "bare quote"

	// AltExtraneousQuote is the description for extraneous-quote record alterations.
	AltExtraneousQuote = "extraneous quote"

	// AltTruncatedRecord is the description for truncated record alterations.
	AltTruncatedRecord = "truncated record"

	// AltPaddedRecord is the description for padded record alterations.
	AltPaddedRecord = "padded record"
)

// Scanner provides facility for permissively reading CSV input. Successive
// calls to the Scan method will step through the records of a file, skipping
// terminator bytes between each record.
//
// Terminators (line endings) can be any (or a mix) of DOS (\r\n), inverted DOS
// (\n\r), unix (\n), or carriage return (\r) tokens.  When scanning, the
// scanner looks for the next occurence of any known token within a search
// space.
//
// Any tokens that fall within a pair of double quotes are ignored.
//
// If no tokens are found within the current search space, the space is expanded
// until either a token or EOF is reached.
//
// If only one token is found in the current space, that token is
// presumed to be the terminator for the current record.
//
// If more than one potential token is identified in the current
// space, the Scanner will select the first, non-quoted, highest priority
// token. The Scanner first gives priority to token length. Longer tokens have
// higher priority than shorter tokens. This priority avoids lexographical
// confusion between shorter tokens and longer tokens that are actually
// composites of the shorter tokens. Thus, DOS and inverted DOS terminators have
// highest priority, as they are longer than unix or carriage return
// terminators. Between two or more tokens of the same length, the Scanner gives
// priority to tokens that are more common. Thus DOS has higher priority than
// inverted DOS because inverted DOS is a non-standard terminator. Similarly
// between unix and carriage return, unix has priority, as bare carriage returns
// are a non-standard terminator.
//
// The preceding terminator detection process is repeated for each record that
// is scanned.
//
// Once a record is identified, it is split into fields using standard CSV
// encoding rules. A mixture of quoted and unquoted field values is permitted,
// and fields are presumed to be separated by commas. The first record scanned
// is always presumed to have the correct number of fields. For each subsequent
// record, if the record has fewer fields than expected, the scanner will pad
// the record with blank fields to accommodate the missing data. If the record
// has more fields than expected, the scanner will truncate the record so its
// length matches the desired length. Information about padded or truncated
// records is made available via the Summary method once scanning is complete.
//
// When parsing the fields of a record, the Scanner might encounter ambiguous
// double quotes. Two common quote ambiguities are handled by the Scanner.
// 1) Bare-Quotes, where a field contains two quotes, but also appears to have
// data outside of the quotes. 2) Extraneous-Quotes, where a record appears to
// have an odd number of quotes, making it impossible to determine if a quote
// was left unclosed, or if the extraneous quote was supposed to be escaped.
// If the Scanner encounters quotes that are ambiguous, it will return empty
// fields in place of any data that might have been present, as the Scanner is
// unable to make any assumptions about the author's intentions. When such
// replacements are made, the type of replacement, record number, and original
// data are all immediately available via the Summary method.
type Scanner struct {
	headerCheck        HeaderCheck
	currentRecord      []string
	reader             io.ReadSeeker
	scanner            *bufio.Scanner
	expectedFieldCount int
	recordsScanned     int64
	scanSummary        *ScanSummary
	checkedForHeader   bool
	splitter           *linesplit.Splitter

	// these values can only be non-nil the first time Scan is called
	// and will be nil for all subsequent calls.
	firstRecord  []string
	secondRecord []string
}

// HeaderCheck is a function that evaluates whether or not firstRecord is
// a header. HeaderCheck is called by the RecordIsHeader method, and is supplied
// values according to the current state of the Scanner.
//
// firstRecord is the first record of the file.
// firstRecord will be nil in the following conditions:
//  - Scan has not been called.
//  - The file is empty.
//  - The Scanner has advanced beyond the first record.
//
// secondRecord is the second record of the file.
// secondRecord will be nil in the following conditions:
//  - Scan has not been called
//  - The file is empty.
//  - The Scanner has advanced beyond the first record.
//  - The file does not have a second record.
type HeaderCheck func(firstRecord, secondRecord []string) bool

// HeaderCheckAssumeNoHeader is a HeaderCheck that instructs the RecordIsHeader
// method to report that no header exists for the file being scanned.
var HeaderCheckAssumeNoHeader HeaderCheck = func(firstRecord, secondRecod []string) bool {
	return false
}

// HeaderCheckAssumeHeaderExists returns true unless firstRecord is nil.
var HeaderCheckAssumeHeaderExists HeaderCheck = func(firstRecord, secondRecod []string) bool {
	return firstRecord != nil
}

// NewScanner returns a new Scanner to read from r.
func NewScanner(r io.ReadSeeker, headerCheck HeaderCheck) *Scanner {
	internalScanner := bufio.NewScanner(r)
	s := &Scanner{
		headerCheck: headerCheck,
		reader:      r,
		scanner:     internalScanner,
		splitter:    new(linesplit.Splitter),
	}
	internalScanner.Split(s.splitter.Split)
	return s
}

// Scan advances the scanner to the next record, which will then be available
// via the CurrentRecord method. Scan returns false when it reaches the end
// of the file. Once scanning is complete, subsequent scans will continue to
// return false until the Reset method is called.
//
// If the underlaying Reader is nil, Scan will return false on the first call.
// In all other cases, Scan will return true on the first call.
func (s *Scanner) Scan() bool {
	if !s.checkedForHeader {
		more := s.scan()
		s.firstRecord = make([]string, len(s.currentRecord))
		copy(s.firstRecord, s.currentRecord)
		if more {
			s.scan()
			if !s.Summary().EOF {
				s.secondRecord = make([]string, len(s.currentRecord))
				copy(s.secondRecord, s.currentRecord)
			}
		}
		s.recordsScanned = 0
		s.currentRecord = nil
		s.scanSummary = nil
		if s.reader != nil {
			s.reader.Seek(0, io.SeekStart)
		}
		s.scanner = bufio.NewScanner(s.reader)
		s.scanner.Split(s.splitter.Split)
		s.checkedForHeader = true
	} else {
		s.firstRecord = nil
		s.secondRecord = nil
	}
	return s.scan()
}

func (s *Scanner) scan() bool {
	var (
		extraneousQuoteEncountered = false
		bareQuoteEncountered       = false
		recordTruncated            = false
		recordPadded               = false
	)

	if s.scanSummary == nil {
		s.scanSummary = &ScanSummary{
			Alterations: []*Alteration{},
		}
	}

	if s.reader == nil {
		s.scanSummary.Err = ErrReaderIsNil
		s.scanSummary.RecordCount = -1
		s.scanSummary.AlterationCount = -1
		s.scanSummary.EOF = false
		return false
	}

	var record []string
	more := s.scanner.Scan()
	if !more {
		s.scanSummary.EOF = true
		return false
	}

	s.scanSummary.RecordCount++
	rawRecord := s.scanner.Text()
	var trimmedRawRecord string
	currentTerminator := s.splitter.CurrentTerminator()
	if len(currentTerminator) > 0 && strings.HasSuffix(rawRecord, string(currentTerminator)) {
		trimmedRawRecord = rawRecord[:len(rawRecord)-len(currentTerminator)]
	} else {
		trimmedRawRecord = rawRecord
	}

	if trimmedRawRecord == "" {
		record = []string{""}
	} else {
		// we want to leverage csv.Reader for its field parsing logic, but
		// want to avoid its record parsing logic. So, we replace any instances
		// of \n or \r with tokens to override the Readers standard record
		// termination handling; then fix the tokens after the fact.
		text := util.TokenizeTerminators(trimmedRawRecord)
		c := csv.NewReader(strings.NewReader(text))
		var err error
		record, err = c.Read()
		if err != nil {
			extraneousQuoteEncountered = util.IsExtraneousQuoteError(err)
			bareQuoteEncountered = util.IsBareQuoteError(err)
			record = []string{}
		}
		record = util.ResetTerminatorTokens(record)
	}

	s.recordsScanned++
	if s.recordsScanned == 1 {
		s.expectedFieldCount = len(record)
	}

	if len(record) > s.expectedFieldCount {
		record = record[:s.expectedFieldCount]
		recordTruncated = true
	} else if len(record) < s.expectedFieldCount {
		pad := make([]string, s.expectedFieldCount-len(record))
		record = append(record, pad...)
		recordPadded = true
	}

	// In cases where the record (for any reason) ends up with zero capacity
	// (nil), we return an empty slice with capacity 1 instead. This ensures the
	// scanner always returns an empty slice, rather than a nil slice if a
	// record contains no fields.
	if cap(record) == 0 {
		record = make([]string, 0, 1)
	}
	s.currentRecord = record

	if extraneousQuoteEncountered {
		s.appendAlteration(trimmedRawRecord, record, AltExtraneousQuote)
	} else if bareQuoteEncountered {
		s.appendAlteration(trimmedRawRecord, record, AltBareQuote)
	} else if recordTruncated {
		s.appendAlteration(trimmedRawRecord, record, AltTruncatedRecord)
	} else if recordPadded {
		s.appendAlteration(trimmedRawRecord, record, AltPaddedRecord)
	}

	return true
}

func (s *Scanner) appendAlteration(originalText string, record []string, description string) {
	s.scanSummary.AlterationCount++
	s.scanSummary.Alterations = append(s.scanSummary.Alterations, &Alteration{
		RecordOrdinal:         s.scanSummary.RecordCount,
		OriginalData:          originalText,
		ResultingRecord:       record,
		AlterationDescription: description,
	})
}

// Reset sets the Scanner back to the top of the file, and clears any summary
// data that any previous calls to Scan may have generated.
func (s *Scanner) Reset() {
	if s.reader != nil {
		s.reader.Seek(0, io.SeekStart)
	}
	s = NewScanner(s.reader, s.headerCheck)
}

// CurrentRecord returns the most recent record generated by a call to Scan.
func (s *Scanner) CurrentRecord() []string {
	return s.currentRecord
}

// Alteration describes a change that the Scanner made to a record because the
// record was in an unexpected format.
type Alteration struct {
	RecordOrdinal         int
	OriginalData          string
	ResultingRecord       []string
	AlterationDescription string
}

// ScanSummary contains information about assumptions or alterations that have
// been made via any calls to Scan.
type ScanSummary struct {
	RecordCount     int
	AlterationCount int
	Alterations     []*Alteration
	EOF             bool
	Err             error
}

// String returns a prettified representation of the summary.
func (s *ScanSummary) String() string {
	const templateText = `Scan Summary
---------------------------------------
  Records Scanned:    {{.RecordCount}}
  Alterations Made:   {{.AlterationCount}}
  EOF:                {{.EOF}}
  Err:                {{if .Err}}{{.Err}}{{else}}none{{end}}
  Alterations:{{range .Alterations}}
    Record Number:    {{.RecordOrdinal}}
    Alteration:       {{.AlterationDescription}}
    Original Data:    {{.OriginalData}}
    Resulting Record: {{json .ResultingRecord}}
{{else}}        none{{end}}`

	var recordToJSON = func(s []string) string {
		record, err := json.Marshal(s)
		util.Panic(err)
		return string(record)
	}
	funcMap := template.FuncMap{"json": recordToJSON}
	tmpl := template.Must(template.
		New("summary").
		Funcs(funcMap).
		Parse(templateText))
	buf := new(bytes.Buffer)
	util.Panic(tmpl.Execute(buf, s))
	result, err := ioutil.ReadAll(buf)
	util.Panic(err)
	return string(result)
}

// Summary returns a summary of information about the assumptions or alterations
// that were made during the most recent Scan. If the Scan method has not been
// called, or Reset was called after the last call to Scan, Summary will return
// nil. Summary will continue to collect data each time Scan is called, and will
// only reset after the Reset method has been called.
func (s *Scanner) Summary() *ScanSummary {
	return s.scanSummary
}

// RecordIsHeader returns true if the current record has been identified as a
// header. RecordIsHeader determines if the current record is a header by
// calling the HeaderCheck callback which was supplied to NewScanner when the
// Scanner was instantiated.
func (s *Scanner) RecordIsHeader() bool {
	return s.headerCheck(s.firstRecord, s.secondRecord)
}

// Segment represents a byte range within a file that contains a subset of
// records.
type Segment struct {
	Ordinal     int64
	LowerOffset int64
	UpperOffset int64
	SegmentSize int64
}

// Partition reads the full file and divides it into a series of partitions,
// each of which contains n records. All partitions are guaranteed to contain at
// least n records, except for the final partition, which may contain a
// smaller number of records.
//
// Each partition is represented by a Segment, which contains an Ordinal (an
// integer value representing the segment's placement relative to other
// segments), the lower byte offset where the partition starts, the upper byte
// offset where the partition ends, and the segment size, which is the
// partition length in bytes. If the file being read is empty (0 bytes),
// Partition will return a single empty segment with a length of zero, and
// both offsets set to -1.
//
// To maintain record consistency across segments, the byte offsets
// for a segment typically exclude its trailing terminator. Stripping the
// trailing terminator from the segment ensures that each segment can be properly
// interpreted as an independent file without having to make potentially
// erronious assumptions about implied empty records. In cases where a
// leading or trailing terminator implies that an empty record exists, the
// terminator will be retained. The following examples demonstrate how these
// rules are applied:
//
//	Terminators between segments are excluded from each segments byte-range.
//	----------------------------------------------------------------------------
//	Example 1: 6 records, 2 records per segment
//	Raw Data : 1\n2\n3\n4\n5\n6
//	Segment 1: 1\n2 <-- \n following record 2 is ommitted
//	Segment 2: 3\n4 <-- \n following record 4 is ommitted
//	Segment 3: 5\n6
//
//	Leading terminators are retained, as permissivecsv assumes they imply the
//	file starts with a record with a single empty field.
//	----------------------------------------------------------------------------
//	Example 2: 6 records, 2 records per segment
//	Raw Data :  \n2\n3\n4\n5\n6
//	Segment 1:  \n2 <-- record 1 is implied to be an empty record.
//	Segment 2: 3\n4
//	Segment 3: 5\n6
//
//	Trailing terminators are included if an empty record is implied.
//	----------------------------------------------------------------------------
//	Example 3: 6 records, 2 records per segment
//	Raw Data : 1\n2\n\n4\n5\n
//	Segment 1: 1\n2
//	Segment 2:  \n4 <-- record 3 is implied as an empty record
//	Segment 3: 5\n  <-- record 6 is implied as an empty record
//
//  Terminators are retained for empty records.
//	----------------------------------------------------------------------------
//	Example 4: 6 records, 1 records per segment
//	Raw Data : 1\n\n3\n\n5\n6
//	Segment 1: 1
//	Segment 2: \n  <-- record 2 is implied as an empty record
//	Segment 3: 3
//	Segment 4: \n  <-- record 4 is implied as an empty record
//	Segment 5: 5
//	Segment 6: 6
//
// If excludeHeader is true, Partition will check if a header exists. If a
// header is detected, the first Segment will ignore the header, and the
// LowerOffset value will be the first byte position after the header record.
//
// If excludeHeader is false, the LowerOffset of the first segment will always
// be 0 (regardless of whether the first record is a header or not).
//
// Partition is designed to be used in conjunction with byte offset seekers
// such as os.File.Seek or bufio.ReadSeeker.Discard in situations where files
// need to be accessed in a concurrent manner.
//
// Before processing, Partition explicitly resets the underlaying reader to the
// top of the file. Thus, using Partition in conjunction with Scan could have
// undesired results.
func (s *Scanner) Partition(n int, excludeHeader bool) []*Segment {
	var (
		ordinal     int64
		lowerOffset int64
		upperOffset int64
	)
	s.Reset()
	segments := []*Segment{}
	currentRawRecord := ""
	recordsInCurrentSegment := 0
	for s.Scan() {
		if recordsInCurrentSegment == n {
			ordinal++
			segments = append(segments, &Segment{
				Ordinal:     ordinal,
				LowerOffset: lowerOffset,
				UpperOffset: upperOffset - int64(len(s.splitter.CurrentTerminator())),
				SegmentSize: upperOffset - lowerOffset,
			})
			recordsInCurrentSegment = 0
			currentRawRecord = ""
			lowerOffset = upperOffset + int64(len(s.splitter.CurrentTerminator()))
		}
		currentRawRecord += s.scanner.Text()
		upperOffset = lowerOffset + int64(len(currentRawRecord)-len(s.splitter.CurrentTerminator()))
		recordsInCurrentSegment++
	}

	if recordsInCurrentSegment > 0 {
		ordinal++
		segments = append(segments,
			&Segment{
				Ordinal:     ordinal,
				LowerOffset: lowerOffset,
				UpperOffset: upperOffset - int64(len(s.splitter.CurrentTerminator())) - 1,
				SegmentSize: upperOffset - lowerOffset,
			})
	}

	if len(segments) == 1 && s.scanner.Text() == "" {
		segments[0].UpperOffset = 0
	}

	summary := s.Summary()
	if summary.Err == ErrReaderIsNil {
		segments = append(segments, &Segment{
			Ordinal:     -1,
			LowerOffset: -1,
			UpperOffset: -1,
			SegmentSize: -1,
		})
	}
	return segments
}
