package linesplit

import (
	"bufio"

	"github.com/eltorocorp/permissivecsv/internal/util"
)

// Splitter provides a lineSplit function that will split records on
// unix, DOS, inverted DOS (/n/r) or bare carriage return (/r) terminators.
// Splitter emits certain information about the status of the splitter,
// such as the most recently read record, terminator, terminator length, etc...
type Splitter struct {
	currentRawRecord        string
	currentTerminator       string
	currentTerminatorLength int
	atEOF                   bool
	currentRawUpperOffset   uint64
}

// CurrentRawRecord returns the record that was most recently idenfied by the
// splitter.
func (l *Splitter) CurrentRawRecord() string {
	return l.currentRawRecord
}

// CurrentTerminator returns the terminator that was most recently identified
// by the splitter.
func (l *Splitter) CurrentTerminator() string {
	return l.currentTerminator
}

// CurrentTerminatorLength returns the length of the terminator that was most
// recently identified by the splitter.
func (l *Splitter) CurrentTerminatorLength() int {
	return len(l.CurrentTerminator())
}

// EOF returns true if the splitter has reached the end of the file.
func (l *Splitter) EOF() bool {
	return l.atEOF
}

// Split performs the line splitting operations.
func (l *Splitter) Split(data []byte, atEOF bool) (advance int, token []byte, err error) {
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
		token = data[:advance]
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
		token = data[:advance]
		return
	}

	if !atEOF {
		return
	}

	token = data
	err = bufio.ErrFinalToken
	return
}
