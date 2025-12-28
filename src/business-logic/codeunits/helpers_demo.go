package codeunits

import (
	"fmt"
	"strings"

	"github.com/hansjlachmann/openerp/src/foundation/common"
	"github.com/hansjlachmann/openerp/src/foundation/session"
)

// HelpersDemo - Codeunit 50005: Helper Functions Demo
// Demonstrates IncStr, CopyStr, and other string helpers
const HelpersDemoID = 50005

type HelpersDemo struct {
	session *session.Session
}

// NewHelpersDemo creates a new instance of the codeunit
func NewHelpersDemo(s *session.Session) *HelpersDemo {
	return &HelpersDemo{
		session: s,
	}
}

// RunCLI executes the Helpers Demo codeunit from CLI
func (c *HelpersDemo) RunCLI() error {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("HELPER FUNCTIONS DEMO - BC/NAV Style")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\n--- IncStr() - Increment String ---")
	c.testIncStr()

	fmt.Println("\n--- CopyStr() - Copy Substring ---")
	c.testCopyStr()

	fmt.Println("\n--- PadStr() - Pad String ---")
	c.testPadStr()

	fmt.Println("\n--- DelChr() - Delete Characters ---")
	c.testDelChr()

	fmt.Println("\n--- Other Functions ---")
	c.testOtherFunctions()

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("✓ Demo complete!")

	return nil
}

// testIncStr demonstrates IncStr function
func (c *HelpersDemo) testIncStr() {
	tests := []string{"001", "099", "ABC", "A99", "XYZ", ""}

	for _, test := range tests {
		result := common.IncStr(test)
		fmt.Printf("  IncStr(\"%s\") = \"%s\"\n", test, result)
	}

	// Practical example: Generate next customer number
	fmt.Println("\n  Practical Example - Next Customer Number:")
	lastCustomerNo := "CUST-0099"
	nextCustomerNo := common.IncStr(lastCustomerNo)
	fmt.Printf("  Last: %s  →  Next: %s\n", lastCustomerNo, nextCustomerNo)
}

// testCopyStr demonstrates CopyStr function
func (c *HelpersDemo) testCopyStr() {
	str := "Hello World"

	fmt.Printf("  Source: \"%s\"\n\n", str)
	fmt.Printf("  CopyStr(\"%s\", 1, 5) = \"%s\"\n", str, common.CopyStr(str, 1, 5))
	fmt.Printf("  CopyStr(\"%s\", 7) = \"%s\"\n", str, common.CopyStr(str, 7))
	fmt.Printf("  CopyStr(\"%s\", 1) = \"%s\"\n", str, common.CopyStr(str, 1))
	fmt.Printf("  CopyStr(\"%s\", 10, 5) = \"%s\"\n", str, common.CopyStr(str, 10, 5))

	// Practical example: Extract parts of a formatted string
	fmt.Println("\n  Practical Example - Extract Date Parts:")
	dateStr := "2025-12-27"
	year := common.CopyStr(dateStr, 1, 4)
	month := common.CopyStr(dateStr, 6, 2)
	day := common.CopyStr(dateStr, 9, 2)
	fmt.Printf("  Date: %s  →  Year: %s, Month: %s, Day: %s\n", dateStr, year, month, day)
}

// testPadStr demonstrates PadStr function
func (c *HelpersDemo) testPadStr() {
	str := "42"

	fmt.Printf("  PadStr(\"%s\", 5, \"0\", true) = \"%s\"\n", str, common.PadStr(str, 5, "0", true))
	fmt.Printf("  PadStr(\"%s\", 5, \"0\", false) = \"%s\"\n", str, common.PadStr(str, 5, "0", false))

	// Practical example: Format invoice number
	invoiceNum := "123"
	formatted := common.PadStr(invoiceNum, 8, "0", true)
	fmt.Printf("\n  Invoice Number: %s  →  Formatted: INV-%s\n", invoiceNum, formatted)
}

// testDelChr demonstrates DelChr function
func (c *HelpersDemo) testDelChr() {
	str := "  Hello World  "

	fmt.Printf("  Source: \"%s\"\n\n", str)
	fmt.Printf("  DelChr(\"%s\", '<>', ' ') = \"%s\"\n", str, common.DelChr(str, "<>", " "))
	fmt.Printf("  DelChr(\"%s\", '<', ' ') = \"%s\"\n", str, common.DelChr(str, "<", " "))
	fmt.Printf("  DelChr(\"%s\", '>', ' ') = \"%s\"\n", str, common.DelChr(str, ">", " "))

	str2 := "ABC-123-XYZ"
	fmt.Printf("\n  DelChr(\"%s\", '=', '-') = \"%s\"\n", str2, common.DelChr(str2, "=", "-"))
}

// testOtherFunctions demonstrates other helper functions
func (c *HelpersDemo) testOtherFunctions() {
	str := "Hello World"

	fmt.Printf("  UpperCase(\"%s\") = \"%s\"\n", str, common.UpperCase(str))
	fmt.Printf("  LowerCase(\"%s\") = \"%s\"\n", str, common.LowerCase(str))
	fmt.Printf("  StrLen(\"%s\") = %d\n", str, common.StrLen(str))
	fmt.Printf("  StrPos(\"%s\", \"World\") = %d\n", str, common.StrPos(str, "World"))
	fmt.Printf("  InsStr(\"%s\", \"Beautiful \", 7) = \"%s\"\n", str, common.InsStr(str, "Beautiful ", 7))
	fmt.Printf("  DelStr(\"%s\", 7, 6) = \"%s\"\n", str, common.DelStr(str, 7, 6))
	fmt.Printf("  ConvertStr(\"%s\", \"el\", \"ip\") = \"%s\"\n", str, common.ConvertStr(str, "el", "ip"))
}

// RunHelpersDemo is the main entry point for running this codeunit from the application
func RunHelpersDemo() {
	// Get global session
	sess := session.GetCurrent()
	if sess == nil {
		fmt.Println("✗ Error: No active session")
		return
	}

	// Create codeunit instance
	demo := NewHelpersDemo(sess)

	// Execute codeunit
	err := demo.RunCLI()
	if err != nil {
		fmt.Printf("\n✗ Error: %v\n", err)
	}

	// Wait for user
	fmt.Print("\nPress Enter to continue...")
	sess.GetScanner().Scan()
}
