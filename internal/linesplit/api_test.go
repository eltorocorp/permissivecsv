package linesplit_test

import (
	"bufio"
	"testing"

	"github.com/eltorocorp/permissivecsv/internal/linesplit"
	"github.com/stretchr/testify/assert"
)

func Test_Split(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		atEOF      bool
		expAdvance int
		expToken   []byte
		expErr     error
	}{
		{
			name:       "no data",
			data:       nil,
			atEOF:      true,
			expAdvance: 0,
			expToken:   nil,
			expErr:     bufio.ErrFinalToken,
		},
		{
			// In the initial read, Split should return 0, nil, nil, requesting
			// that the search space be increased.
			name:       "no terminator initial split",
			data:       []byte("a,b,c"),
			atEOF:      false,
			expAdvance: 0,
			expToken:   nil,
			expErr:     nil,
		},
		{
			name:       "no terminator final split",
			data:       []byte("a,b,c"),
			atEOF:      true,
			expAdvance: 0,
			expToken:   []byte("a,b,c"),
			expErr:     bufio.ErrFinalToken,
		},
		{
			// The trailing terminator should be included with the record it
			// terminates.
			name:       "unix",
			data:       []byte("a,b,c\nd,e,f"),
			atEOF:      false,
			expAdvance: 6,
			expToken:   []byte("a,b,c\n"),
			expErr:     nil,
		},
		{
			name:       "dos",
			data:       []byte("a,b,c\r\nd,e,f"),
			atEOF:      false,
			expAdvance: 7,
			expToken:   []byte("a,b,c\r\n"),
			expErr:     nil,
		},
		{
			name:       "carriage return",
			data:       []byte("a,b,c\rd,e,f"),
			atEOF:      false,
			expAdvance: 6,
			expToken:   []byte("a,b,c\r"),
			expErr:     nil,
		},
		{
			name:       "inverted dos",
			data:       []byte("a,b,c\r\nd,e,f"),
			atEOF:      false,
			expAdvance: 7,
			expToken:   []byte("a,b,c\r\n"),
			expErr:     nil,
		},
	}

	for _, test := range tests {
		testFn := func(t *testing.T) {
			splitter := new(linesplit.Splitter)
			actAdvance, actToken, actErr := splitter.Split(test.data, test.atEOF)
			assert.Equal(t, test.expAdvance, actAdvance, "advance")
			assert.Equal(t, test.expToken, actToken, "token")
			assert.Equal(t, test.expErr, actErr, "err")
		}
		t.Run(test.name, testFn)
	}
}
