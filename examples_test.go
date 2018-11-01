package permissivecsv_test

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/eltorocorp/permissivecsv"
)

func ExampleScanner_Scan() {
	data := strings.NewReader("a,b,c/nd,e,f")
	s := permissivecsv.NewScanner(data, permissivecsv.HeaderCheckAssumeNoHeader)
	for s.Scan() {
		fmt.Println(s.CurrentRecord())
	}
	//Output: [a b c/nd e f]
}

func ExampleScanner_Summary() {
	data := strings.NewReader("a,b,c\nd,ef\ng,h,i")
	s := permissivecsv.NewScanner(data, permissivecsv.HeaderCheckAssumeHeaderExists)
	for s.Scan() {
		continue
	}
	summary := s.Summary()
	fmt.Println(summary.String())
	//Output: Scan Summary
	// ---------------------------------------
	//   Records Scanned:    3
	//   Alterations Made:   1
	//   EOF:                true
	//   Err:                none
	//   Alterations:
	//     Record Number:    2
	//     Alteration:       padded record
	//     Original Data:    d,ef
	//     Resulting Record: ["d","ef",""]
}

// Note that, in this example, we are assuming the header exists, and are also
// instructing Partition to exclude the header from the segments. This is why
// segment 1 starts at offset 6, just after the header record.
func ExampleScanner_Partition() {
	data := strings.NewReader("a,b,c\nd,e,f\ng,h,i\nj,k,l\n")
	s := permissivecsv.NewScanner(data, permissivecsv.HeaderCheckAssumeHeaderExists)
	recordsPerPartition := 2
	excludeHeader := true
	partitions := s.Partition(recordsPerPartition, excludeHeader)

	// serializing to JSON just to prettify the output.
	segmentJSON, _ := json.MarshalIndent(partitions, "", "  ")
	fmt.Println(string(segmentJSON))
	//Output:
	// [
	//   {
	//     "Ordinal": 1,
	//     "LowerOffset": 6,
	//     "UpperOffset": 16,
	//     "SegmentSize": 11
	//   },
	//   {
	//     "Ordinal": 2,
	//     "LowerOffset": 18,
	//     "UpperOffset": 22,
	//     "SegmentSize": 5
	//   }
	// ]
}

func ExampleScanner_RecordIsHeader_assumeHeaderExists() {
	data := strings.NewReader("a,b,c\nd,e,f")
	s := permissivecsv.NewScanner(data, permissivecsv.HeaderCheckAssumeHeaderExists)
	for s.Scan() {
		fmt.Println(s.RecordIsHeader())
	}
	//Output:
	//true
	//false
}

func ExampleScanner_RecordIsHeader_assumeNoHeader() {
	data := strings.NewReader("a,b,c\nd,e,f")
	s := permissivecsv.NewScanner(data, permissivecsv.HeaderCheckAssumeNoHeader)
	for s.Scan() {
		fmt.Println(s.RecordIsHeader())
	}
	//Output:
	//false
	//false
}

// This example demonstrates implementing custom header detection logic.
// The example shows how to properly check for nil conditions, and how the first
// two records of a file can be compared when making a determination about if
// the first record is a header. This is a fairly trivial example of header
// detection. Review the HeaderCheck docs for a full list of implementation
// considerations.
func ExampleScanner_RecordIsHeader_customDetection() {
	headerCheck := func(firstRecord, secondRecord []string) bool {
		// firstRecord will be nil if Scan has not been called, if the file is
		// empty, or the Scanner has advanced beyond the first record.
		if firstRecord == nil {
			return false
		}

		// secondRecord will be nil if the file only contains one record.
		// (Additionally, secondRecord can also be nil for all of the same
		// reasons that firstRecord can be nil.)
		if secondRecord == nil {
			return true
		}

		// If the value of the first field of the first record doesn't match the
		// the value of the first field of the second record, assume a header
		// is present.
		return firstRecord[0] != secondRecord[0]
	}

	data := strings.NewReader("a,b,c\nd,e,f")
	s := permissivecsv.NewScanner(data, headerCheck)
	for s.Scan() {
		fmt.Println(s.RecordIsHeader())
	}
	//Output:
	//true
	//false
}
