package permissivecsv

import (
	"bufio"
	"encoding/csv"
	"io"
	"strings"

	"github.com/eltorocorp/permissivecsv/internal/util"
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
	scanner            *bufio.Scanner
	expectedFieldCount int
	recordsScanned     int64
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
		token = data[:nearestTerminator+1]
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
		token = data[:nearestTerminator+1]
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
func (s *Scanner) Scan() bool {
	var record []string
	scanResult := s.scanner.Scan()
	text := s.scanner.Text()
	if text == "" {
		record = []string{""}
	} else {
		c := csv.NewReader(strings.NewReader(text))
		// we disregard Read's error since we're behaving permissively.
		record, _ = c.Read()
	}

	s.recordsScanned++
	if s.recordsScanned == 1 {
		s.expectedFieldCount = len(record)
	}

	if len(record) > s.expectedFieldCount {
		record = record[:s.expectedFieldCount]
	} else if len(record) < s.expectedFieldCount {
		pad := make([]string, s.expectedFieldCount-len(record))
		record = append(record, pad...)
	}

	// See "loitering terminator" test. If the initial value in a file is a
	// terminator, the Splitter may return nil data. In cases where the record
	// (for any reason) ends up with zero capacity (nil), we return an empty
	// slice with capacity 1 instead. This ensures the scanner always returns
	// an empty slice, rather than a nil slice if a record contains no fields.
	if cap(record) == 0 {
		record = make([]string, 0, 1)
	}
	s.currentRecord = record
	return scanResult
}

// NextRecord returns the next record without advancing the Scanner. If the current
// record is the last record of the file EOF will be true, otherwise EOF will be
// false.
func (s *Scanner) NextRecord() (nextRecord []string, EOF bool) {
	panic("not implemented")
}

// Reset sets the Scanner back to the top of the file, and clears any summary
// data that any previous calls to Scan may have generated.
func (s *Scanner) Reset() {
	panic("not implemented")
}

// CurrentRecord returns the most recent record generated by a call to Scan.
func (s *Scanner) CurrentRecord() []string {
	return s.currentRecord
}

// ScanSummary contains information about assumptions or alterations that have
// been made via any calls to Scan.
type ScanSummary struct {
}

// Summary returns a summary of information about the assumptions or alterations
// that were made during the most recent Scan. If the Scan method has not been
// called, or Reset was called after the last call to Scan, Summary will return
// nil. Summary will continue to collect data each time Scan is called, and will
// only reset after the Reset method has been called.
func (s *Scanner) Summary() *ScanSummary {
	panic("not implemented")
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
