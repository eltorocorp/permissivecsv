package permissivecsv_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/eltorocorp/permissivecsv"
	"github.com/stretchr/testify/assert"
)

const testFileLocation = "integrationtestdata"

func Test_ScanIntegration(t *testing.T) {
	tests := []struct {
		filename       string
		headerCheck    permissivecsv.HeaderCheck
		expRecordCount int
		expHeader      []string
		expRecords     [][]string
	}{
		{
			filename:       "simpleunix.csv",
			headerCheck:    permissivecsv.HeaderCheckAssumeHeaderExists,
			expRecordCount: 3,
			expHeader:      []string{"field1", "field2", "field3"},
			expRecords: [][]string{
				[]string{"field1", "field2", "field3"},
				[]string{"1", "2", "3"},
				[]string{"4", "5", "6"},
			},
		},
		{
			filename:       "simpledos.csv",
			headerCheck:    permissivecsv.HeaderCheckAssumeHeaderExists,
			expRecordCount: 3,
			expHeader:      []string{"field1", "field2", "field3"},
			expRecords: [][]string{
				[]string{"field1", "field2", "field3"},
				[]string{"1", "2", "3"},
				[]string{"4", "5", "6"},
			},
		},
		{
			filename:       "inverteddos.csv",
			headerCheck:    permissivecsv.HeaderCheckAssumeHeaderExists,
			expRecordCount: 3,
			expHeader:      []string{"field1", "field2", "field3"},
			expRecords: [][]string{
				[]string{"field1", "field2", "field3"},
				[]string{"1", "2", "3"},
				[]string{"4", "5", "6"},
			},
		},
		{
			filename:       "carriagereturn.csv",
			headerCheck:    permissivecsv.HeaderCheckAssumeHeaderExists,
			expRecordCount: 3,
			expHeader:      []string{"field1", "field2", "field3"},
			expRecords: [][]string{
				[]string{"field1", "field2", "field3"},
				[]string{"1", "2", "3"},
				[]string{"4", "5", "6"},
			},
		},
		{
			filename:       "emptyrecords.csv",
			headerCheck:    permissivecsv.HeaderCheckAssumeHeaderExists,
			expRecordCount: 4,
			expHeader:      []string{"field1", "field2", "field3"},
			expRecords: [][]string{
				[]string{"field1", "field2", "field3"},
				[]string{"1", "2", "3"},
				[]string{"4", "5", "6"},
				[]string{"7", "8", "9"},
			},
		},
		{
			filename:       "emptyrecordsnoheader.csv",
			headerCheck:    permissivecsv.HeaderCheckAssumeNoHeader,
			expRecordCount: 3,
			expHeader:      nil,
			expRecords: [][]string{
				[]string{"1", "2", "3"},
				[]string{"4", "5", "6"},
				[]string{"7", "8", "9"},
			},
		},
		{
			filename:       "mixedterminators.csv",
			headerCheck:    permissivecsv.HeaderCheckAssumeHeaderExists,
			expRecordCount: 9,
			expHeader:      []string{"field1", "field2", "field3"},
			expRecords: [][]string{
				[]string{"field1", "field2", "field3"},
				[]string{"a", "b", "c"},
				[]string{"d", "e", "f"},
				[]string{"g", "h", "i"},
				[]string{"j", "k", "l"},
				[]string{"m", "n", "o"},
				[]string{"p", "q", "r"},
				[]string{"s", "t", "u"},
				[]string{"v", "w", "x"},
			},
		},
		{
			filename:       "inconsistentfieldcounts.csv",
			headerCheck:    permissivecsv.HeaderCheckAssumeHeaderExists,
			expRecordCount: 6,
			expHeader:      []string{"field1", "field2", "field3"},
			expRecords: [][]string{
				[]string{"field1", "field2", "field3"},
				[]string{"a", "", ""},
				[]string{"a", "b", ""},
				[]string{"a", "b", "c"},
				[]string{"a", "b", "c"},
				[]string{"a", "b", "c"},
			},
		},
		{
			filename:       "quotedcontrolcharacters.csv",
			headerCheck:    permissivecsv.HeaderCheckAssumeNoHeader,
			expRecordCount: 2,
			expHeader:      nil,
			expRecords: [][]string{
				[]string{"1", "2\n", "3"},
				[]string{"4", "\r5", "6"},
			},
		},
		{
			filename:       "unquotedcontrolcharacters.csv",
			headerCheck:    permissivecsv.HeaderCheckAssumeNoHeader,
			expRecordCount: 3,
			expHeader:      nil,
			expRecords: [][]string{
				[]string{"1", "\r2", "3"},
				[]string{"4", "5", ""},
				[]string{"", "6", ""},
			},
		},
	}

	for _, test := range tests {
		testFn := func(t *testing.T) {
			file, err := os.Open(filepath.Join(testFileLocation, test.filename))
			if err != nil {
				panic(err)
			}
			s := permissivecsv.NewScanner(file, test.headerCheck)
			actRecordCount := 0
			actHeader := []string(nil)
			actRecords := [][]string{}
			for s.Scan() {
				if s.RecordIsHeader() {
					actHeader = s.CurrentRecord()
					t.Log("Header:", actHeader)
				}
				actRecords = append(actRecords, s.CurrentRecord())
				recordJSON, _ := json.Marshal(s.CurrentRecord())
				t.Log("File:", test.filename, "CurrentRecord:", string(recordJSON))
				actRecordCount++
			}
			assert.Equal(t, test.expRecordCount, actRecordCount, "record count")
			assert.Equal(t, test.expHeader, actHeader, "header")
			assert.Equal(t, test.expRecords, actRecords, "records")
		}
		t.Run(test.filename, testFn)
	}
}
