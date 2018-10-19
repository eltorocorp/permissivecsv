package permissivecsv

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

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

// Scanner provides an interface for permissively reading CSV input. Successive
// calls to the Scan method will step through the records of a file, skipping
// terminator bytes between each record.
//
// Terminators (line endings) can be any (or a mix) of DOS or unix endings
// (\r\n or \n, respectively). When scanning, the scanner looks for the next
// occurence of both a '\r\n` and a `\n`. Whichever token is encountered first
// is presumed to be the current record terminator. This process is repeated
// for each record that is scanned.
//
// Once a record is identified, it is split into fields using standard CSV
// encoding rules. A mixture of quoted and unquoted field values is permitted,
// and fields are presumed to be separated by commas. The first record scanned
// is always presumed to have the correct number of fields. For each subsequent
// record, if the record has fewer fields than expected, the scanner will pad
// the record with blank fields to accomodate the missing data. If the record
// has more fields than expected, the scanner will truncate the record so its
// length matches the desired value. Information about padded or truncated
// records is made available via the Summary method once scanning is complete.
type Scanner struct {
	headerCheck        HeaderCheck
	currentRecord      []string
	eof                bool
	reader             io.Reader
	scanner            *bufio.Scanner
	expectedFieldCount int
	recordsScanned     int64
	scanSummary        *ScanSummary
}

// HeaderCheck is a function that evaluates whether or not the currentrecord is
// a header. HeaderCheck is called by the RecordIsHeader method, and is supplied
// values according to the current position of the scanner.
//
// i is the current record index.
//
// currentRecord is the current record in the scanner (this is the same value
// that would be returned if the Record method is called.
//
// nextRecord is the record that immediately follows the current record.
//
// If the scanner is at the end of the file, nextRecord will be nil.
//
// If the file is empty, recordi and recordj will both be nil.
//
// In general, if i != 0, HeaderCheck should return false (since
// headers are typically the first record in a file).
type HeaderCheck func(i int, currentRecord, nextRecord *[]string) bool

// HeaderCheckAssumeNoHeader is a HeaderCheck that instructs the RecordIsHeader
// method to report that no header exists for the file being scanned.
var HeaderCheckAssumeNoHeader HeaderCheck = func(i int, currentRecord, nextRecord *[]string) bool {
	return false
}

// HeaderCheckAssumeHeaderExists is a HeaderCheck that instructs the
// RecordIsHeader method to always report that the first record of a file
// is a header.
var HeaderCheckAssumeHeaderExists HeaderCheck = func(i int, currentRecord, nextRecord *[]string) bool {
	return i == 0
}

// NewScanner returns a new Scanner to read from r.
func NewScanner(r io.Reader, headerCheck HeaderCheck) *Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Split(recordSplitter)
	return &Scanner{
		headerCheck: headerCheck,
		reader:      r,
		scanner:     scanner,
	}
}

func recordSplitter(data []byte, atEOF bool) (advance int, token []byte, err error) {
	const (
		nl     = "\n"
		cr     = "\r"
		dos    = "\r\n"
		invdos = "\n\r"
	)
	str := string(data)
	DOSIndex := util.IndexNonQuoted(str, dos)
	invertedDOSIndex := util.IndexNonQuoted(str, invdos)
	newlineIndex := util.IndexNonQuoted(str, nl)
	carriageReturnIndex := util.IndexNonQuoted(str, cr)

	nearestTerminator := -1

	if invertedDOSIndex != -1 &&
		newlineIndex == invertedDOSIndex &&
		carriageReturnIndex > newlineIndex {
		nearestTerminator = invertedDOSIndex
	}

	if DOSIndex != -1 &&
		carriageReturnIndex == DOSIndex &&
		newlineIndex > carriageReturnIndex {
		if nearestTerminator == -1 {
			nearestTerminator = DOSIndex
		} else if DOSIndex < nearestTerminator {
			nearestTerminator = DOSIndex
		}
	}

	if nearestTerminator != -1 {
		advance = nearestTerminator + 2
		token = data[:nearestTerminator]
		return
	}

	if newlineIndex != -1 {
		nearestTerminator = newlineIndex
	}

	if carriageReturnIndex != -1 {
		if nearestTerminator == -1 || carriageReturnIndex < nearestTerminator {
			nearestTerminator = carriageReturnIndex
		}
	}

	if nearestTerminator != -1 {
		advance = nearestTerminator + 1
		token = data[:nearestTerminator]
		return
	}

	if !atEOF {
		return
	}

	token = data
	err = bufio.ErrFinalToken
	return
}

// Scan advances the scanner to the next record, which will then be available
// via the CurrentRecord method. Scan returns false when it reaches the end
// of the file. Once scanning is complete, subsequent scans will continue to
// return false until the Reset method is called.
//
// If the underlaying Reader is nil, Scan will return false on the first call.
// In all other cases, Scan will return true on the first call. If the
func (s *Scanner) Scan() bool {
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
	originalText := s.scanner.Text()

	if originalText == "" {
		record = []string{""}
	} else {
		// we want to leverage csv.Reader for its field parsing logic, but
		// want to avoid its record parsing logic. So, we replace any instances
		// of \n or \r with tokens to override the Readers standard record
		// termination handling; then fix the tokens after the fact.
		text := util.TokenizeTerminators(originalText)
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
		s.appendAlteration(originalText, record, AltExtraneousQuote)
	} else if bareQuoteEncountered {
		s.appendAlteration(originalText, record, AltBareQuote)
	} else if recordTruncated {
		s.appendAlteration(originalText, record, AltTruncatedRecord)
	} else if recordPadded {
		s.appendAlteration(originalText, record, AltPaddedRecord)
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

// Summary returns a summary of information about the assumptions or alterations
// that were made during the most recent Scan. If the Scan method has not been
// called, or Reset was called after the last call to Scan, Summary will return
// nil. Summary will continue to collect data each time Scan is called, and will
// only reset after the Reset method has been called.
func (s *Scanner) Summary() *ScanSummary {
	return s.scanSummary
}

// RecordIsHeader returns true if the current record has been identified as a
// header. RecordIsHeader calls the HeaderCheck callback that was supplied
// to NewScanner when the Scanner was instantiated. If HeaderCheck determines
// that the current record is a header, RecordIsHeader returns true. If
// HeaderCheck determines that the current record is not a header,
// RecordIsHeader will return false. See docs on HeaderSeeker for more
// information.
func (s *Scanner) RecordIsHeader() bool {
	panic("not implemented")
}

// Segment represents a byte range within a file that contains a subset of
// records.
type Segment struct {
	Position    int64
	LowerOffset int64
	UpperOffset int64
	SegmentSize int64
}

// Partition reads the full file and divides it into a series of partitions,
// each of which contains n records. All partitions are guaranteed to contain at
// least n records, except for the final partition, which may contain a
// smaller number of records.
//
// Each partition is represented by a Segment, which contains a Position (a
// zero-based index representing the segment's placement relative to other
// segments), the lower byte offset where the partition starts, the upper byte
// offset where the partition ends, and the segment size, which is the
// partition length in bytes.
//
// If ignoreHeaderCheck is excluded or false (the default behavior), Partition
// calls the HeaderCheck callback when reading the file. If HeaderCheck returns
// true, the current record is considered a header, and it is excluded from its
// partition. If ignoreHeaderCheck is true, Partition will always include the
// first record in the first segment, regardless of if it is a header or not.
//
// Partition is designed to be used in conjunction with byte offset seekers
// such as os.File.Seek or bufio.Reader.Discard in situations where files are
// need to be accessed in an asyncronous manner.
//
// Partition implicitly calls Reset before reading the file, so using Scan
// and Partition in conjunction could have undesired results.
func (s *Scanner) Partition(n int, ignoreHeaderCheck ...bool) []Segment {
	panic("not implemented")
}
