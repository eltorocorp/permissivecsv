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
