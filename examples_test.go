package permissivecsv_test

import (
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
