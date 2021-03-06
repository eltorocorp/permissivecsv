package linesplit_test

import (
	"bufio"
	"testing"

	"github.com/eltorocorp/permissivecsv/internal/linesplit"
	"github.com/stretchr/testify/assert"
)

func Test_Split(t *testing.T) {
	tests := []struct {
		name                 string
		data                 []byte
		atEOF                bool
		expAdvance           int
		expToken             []byte
		expErr               error
		expCurrentTerminator []byte
	}{
		{
			name:                 "no data",
			data:                 nil,
			atEOF:                true,
			expAdvance:           0,
			expToken:             nil,
			expErr:               bufio.ErrFinalToken,
			expCurrentTerminator: nil,
		},
		{
			name:                 "empty data at EOF",
			data:                 []byte{},
			atEOF:                true,
			expAdvance:           0,
			expToken:             []byte{},
			expErr:               bufio.ErrFinalToken,
			expCurrentTerminator: []byte{},
		},
		{
			// In the initial read, Split should return 0, nil, nil, requesting
			// that the search space be increased.
			name:                 "no terminator and not EOF",
			data:                 []byte("a,b,c"),
			atEOF:                false,
			expAdvance:           0,
			expToken:             nil,
			expErr:               nil,
			expCurrentTerminator: nil,
		},
		{
			name:                 "no terminator, at EOF",
			data:                 []byte("a,b,c"),
			atEOF:                true,
			expAdvance:           0,
			expToken:             []byte("a,b,c"),
			expErr:               bufio.ErrFinalToken,
			expCurrentTerminator: []byte{},
		},
		// The trailing terminator should be included with the record it
		// terminates.
		{
			name:                 "unix",
			data:                 []byte("a,b,c\nd,e,f"),
			atEOF:                false,
			expAdvance:           6,
			expToken:             []byte("a,b,c\n"),
			expErr:               nil,
			expCurrentTerminator: []byte{10},
		},
		{
			name:                 "dos",
			data:                 []byte("a,b,c\r\nd,e,f"),
			atEOF:                false,
			expAdvance:           7,
			expToken:             []byte("a,b,c\r\n"),
			expErr:               nil,
			expCurrentTerminator: []byte{13, 10},
		},
		{
			name:                 "carriage return",
			data:                 []byte("a,b,c\rd,e,f"),
			atEOF:                false,
			expAdvance:           6,
			expToken:             []byte("a,b,c\r"),
			expErr:               nil,
			expCurrentTerminator: []byte{13},
		},
		{
			name:                 "inverted dos",
			data:                 []byte("a,b,c\n\rd,e,f"),
			atEOF:                false,
			expAdvance:           7,
			expToken:             []byte("a,b,c\n\r"),
			expErr:               nil,
			expCurrentTerminator: []byte{10, 13},
		},
		// If the current search space ends in a newline or carriage return,
		// and no other non-quoted terminators are present at an earlier index,
		// the search space should be increased to ensure that the correct
		// terminator is chosen.
		{
			name:                 "partial dos terminator closing search space",
			data:                 []byte("a,b,c\r"),
			atEOF:                false,
			expAdvance:           0,
			expToken:             nil,
			expErr:               nil,
			expCurrentTerminator: nil,
		},
		{
			name:                 "partial invdos terminator closing search space",
			data:                 []byte("a,b,c\n"),
			atEOF:                false,
			expAdvance:           0,
			expToken:             nil,
			expErr:               nil,
			expCurrentTerminator: nil,
		},
		// Since bare carriage returns are quite rare to be used as terminators,
		// we only want to select a carriage return as the terminator if no
		// other more likely terminator exists within the current search space.
		{
			name:                 "prefer newline over carriage return",
			data:                 []byte("a,b\rc,d\ne,f,g,h"),
			atEOF:                false,
			expAdvance:           8,
			expToken:             []byte("a,b\rc,d\n"),
			expErr:               nil,
			expCurrentTerminator: []byte{10},
		},
		// A terminator at the end of the search space (but not EOF) should
		// always trigger a search space extension.
		// Note that these tests use \r\n as the test case to avoid
		// collision with the partial terminator search space extension
		// requirement.
		{
			name:                 "terminator at end of search space",
			data:                 []byte("a,b,c\r\n"),
			atEOF:                false,
			expAdvance:           0,
			expToken:             nil,
			expErr:               nil,
			expCurrentTerminator: nil,
		},
		{
			name:                 "terminator at end of file",
			data:                 []byte("a,b,c\r\n"),
			atEOF:                true,
			expAdvance:           7,
			expToken:             []byte("a,b,c\r\n"),
			expErr:               nil,
			expCurrentTerminator: []byte{13, 10},
		},
		// If there are an even number of extraneous quotes before any terminator
		// they will be identified as such, and the terminator will be found.
		{
			name:                 "extraneous quotes (even)",
			data:                 []byte("b\"\"b,b,b\nc,c,c"),
			atEOF:                true,
			expAdvance:           9,
			expToken:             []byte("b\"\"b,b,b\n"),
			expErr:               nil,
			expCurrentTerminator: []byte{10},
		},
		// If there are an odd number of extraneous quotes before any terminator,
		// and we are at the end of the file, linesplit can't trust any
		// terminator it finds after the last quote, as it doesn't know if it
		// is "quoted" or not. Instead, the remaineder of the text is returned
		// in full.
		{
			name:                 "extraneous quotes (odd at EOF)",
			data:                 []byte("b\"\"\"b,b,b\nc,c,c"),
			atEOF:                true,
			expAdvance:           0,
			expToken:             []byte("b\"\"\"b,b,b\nc,c,c"),
			expErr:               bufio.ErrFinalToken,
			expCurrentTerminator: []byte{},
		},
		// If there are an odd number of extraneous quotes before any terminator
		// and we are not at the end of the file, linesplit will request to have
		// the search space increased, in an effort to idenfity a missing quote.
		{
			name:                 "extraneous quotes (odd not EOF)",
			data:                 []byte("b\"\"\"b,b,b\nc,c,c"),
			atEOF:                false,
			expAdvance:           0,
			expToken:             nil,
			expErr:               nil,
			expCurrentTerminator: nil,
		},
	}

	for _, test := range tests {
		testFn := func(t *testing.T) {
			splitter := new(linesplit.Splitter)
			actAdvance, actToken, actErr := splitter.Split(test.data, test.atEOF)
			actCurrentTerminator := splitter.CurrentTerminator()
			assert.Equal(t, test.expAdvance, actAdvance, "advance")
			assert.Equal(t, test.expToken, actToken, "token")
			assert.Equal(t, test.expErr, actErr, "err")
			if test.expCurrentTerminator == nil {
				assert.Nil(t, splitter.CurrentTerminator(), "terminator")
			} else if assert.NotNil(t, splitter.CurrentTerminator(), "terminator") {
				assert.Equal(t, test.expCurrentTerminator, actCurrentTerminator, "terminator")
			}
		}
		t.Run(test.name, testFn)
	}
}
