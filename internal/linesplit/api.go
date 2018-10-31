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
	currentTerminator []byte
}

// CurrentTerminator returns the terminator that was most recently identified
// by the splitter. This value will be nil if no data was returned in the
// most recent Split. This value will be an empty slice if data was returned,
// but contained no terminator. Otherwise, if a terminator was identified within
// the slice, that terminator is returned.
func (l *Splitter) CurrentTerminator() []byte {
	return l.currentTerminator
}

// Split performs the line splitting operations.
func (l *Splitter) Split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	const (
		nl     = "\n"
		cr     = "\r"
		dos    = "\r\n"
		invdos = "\n\r"
	)
	l.currentTerminator = nil
	str := string(data)
	DOSIndex := util.IndexNonQuoted(str, dos)
	invertedDOSIndex := util.IndexNonQuoted(str, invdos)
	newlineIndex := util.IndexNonQuoted(str, nl)
	carriageReturnIndex := util.IndexNonQuoted(str, cr)

	nearestTerminator := -1

	if invertedDOSIndex != -1 &&
		newlineIndex == invertedDOSIndex &&
		carriageReturnIndex > newlineIndex {
		l.currentTerminator = []byte(invdos)
		nearestTerminator = invertedDOSIndex
	}

	if DOSIndex != -1 &&
		carriageReturnIndex == DOSIndex &&
		newlineIndex > carriageReturnIndex {
		if nearestTerminator == -1 {
			l.currentTerminator = []byte(dos)
			nearestTerminator = DOSIndex
		} else if DOSIndex < nearestTerminator {
			l.currentTerminator = []byte(dos)
			nearestTerminator = DOSIndex
		}
	}

	if nearestTerminator != -1 {
		if nearestTerminator == len(data)-2 {
			l.currentTerminator = nil
			advance = 0
			token = nil
		} else {
			advance = nearestTerminator + 2
			token = data[:advance]
		}
		return
	}

	if newlineIndex != -1 {
		l.currentTerminator = []byte(nl)
		nearestTerminator = newlineIndex
	}

	if carriageReturnIndex != -1 {
		if nearestTerminator == -1 {
			l.currentTerminator = []byte(cr)
			nearestTerminator = carriageReturnIndex
		}
	}

	if nearestTerminator != -1 {
		if nearestTerminator == len(data)-1 {
			// The nearest terminator is either '\n' or '\r' at the end of the
			// current search space. We need to expand the search space to
			// ensure we are observing the full terminator sequence.
			advance = 0
			token = nil
			l.currentTerminator = nil
		} else {
			advance = nearestTerminator + 1
			token = data[:advance]
		}
		return
	}

	if !atEOF {
		// requesting a larger search space
		return
	}

	token = data
	err = bufio.ErrFinalToken
	if data != nil {
		l.currentTerminator = []byte{}
	}
	return
}
