package codeunits

import (
	"fmt"
	"strings"
	"time"

	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/session"
	"github.com/hansjlachmann/openerp/src/foundation/types"
)

// BidirectionalNavDemo - Codeunit 50007: Bidirectional Navigation Demo
// Demonstrates FindSet() vs FindSetBuffered() and Next() with optional steps parameter
const BidirectionalNavDemoID = 50007

type BidirectionalNavDemo struct {
	session *session.Session
}

// NewBidirectionalNavDemo creates a new instance of the codeunit
func NewBidirectionalNavDemo(s *session.Session) *BidirectionalNavDemo {
	return &BidirectionalNavDemo{
		session: s,
	}
}

// RunCLI executes the Bidirectional Navigation Demo codeunit from CLI
func (c *BidirectionalNavDemo) RunCLI() error {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("BIDIRECTIONAL NAVIGATION DEMO - BC/NAV Style Next() Function")
	fmt.Println(strings.Repeat("=", 70))

	// Ensure we have test data
	recordCount := c.ensureTestData()
	fmt.Printf("\nTest data: %d customer records available\n", recordCount)

	fmt.Println("\n" + strings.Repeat("-", 70))
	fmt.Println("PART 1: Forward-Only Streaming with FindSet()")
	fmt.Println(strings.Repeat("-", 70))
	c.demoForwardOnlyStreaming()

	fmt.Println("\n" + strings.Repeat("-", 70))
	fmt.Println("PART 2: Bidirectional Navigation with FindSetBuffered()")
	fmt.Println(strings.Repeat("-", 70))
	c.demoBidirectionalNavigation()

	fmt.Println("\n" + strings.Repeat("-", 70))
	fmt.Println("PART 3: Filtered Buffered Recordset (Memory Optimization)")
	fmt.Println(strings.Repeat("-", 70))
	c.demoFilteredBufferedRecordset()

	fmt.Println("\n" + strings.Repeat("-", 70))
	fmt.Println("PART 4: Performance Comparison")
	fmt.Println(strings.Repeat("-", 70))
	c.demoPerformanceComparison()

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("✓ Demo complete!")

	return nil
}

// ensureTestData creates test customer records if they don't exist
func (c *BidirectionalNavDemo) ensureTestData() int {
	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	// Check current count
	existingCount := customer.Count()
	if existingCount >= 20 {
		return existingCount // Already have enough test data
	}

	// Create 20 test customers
	cities := []string{"New York", "Chicago", "Los Angeles", "Houston", "Phoenix"}
	for i := 1; i <= 20; i++ {
		customer.Init(c.session.GetConnection(), c.session.GetCompany())
		customerNo := fmt.Sprintf("C%04d", i)

		// Check if exists
		if customer.Get(types.NewCode(customerNo)) {
			continue // Skip if already exists
		}

		// Create new customer
		customer.No = types.NewCode(customerNo)
		customer.Name = types.NewText(fmt.Sprintf("Test Customer %d", i))
		customer.Address = types.NewText(fmt.Sprintf("%d Main Street", i*100))
		customer.City = types.NewText(cities[(i-1)%len(cities)])
		customer.Post_code = types.NewCode(fmt.Sprintf("%05d", 10000+i))
		customer.Phonenumber = types.NewText(fmt.Sprintf("555-%04d", i))

		customer.Insert(false) // Don't run triggers for test data
	}

	customer.Init(c.session.GetConnection(), c.session.GetCompany())
	return customer.Count()
}

// demoForwardOnlyStreaming demonstrates FindSet() with Next() for forward-only iteration
func (c *BidirectionalNavDemo) demoForwardOnlyStreaming() {
	fmt.Println("\nUsing FindSet() for memory-efficient forward streaming:")
	fmt.Println("  - Low memory usage (streaming from database)")
	fmt.Println("  - Next() or Next(1): Move forward")
	fmt.Println("  - Next(5): Skip forward 5 records")
	fmt.Println("  - Next(-1): NOT ALLOWED (error)")

	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	// Set filter to limit results
	customer.SetRange("no", "C0001", "C0010")

	fmt.Println("\n1. Basic Forward Iteration with Next():")
	if customer.FindSet() {
		count := 1
		fmt.Printf("  Record %d: %s - %s\n", count, customer.No, customer.Name)

		// Loop through next 3 records
		for i := 0; i < 3 && customer.Next(); i++ {
			count++
			fmt.Printf("  Record %d: %s - %s\n", count, customer.No, customer.Name)
		}
	}

	fmt.Println("\n2. Skip Forward with Next(3):")
	customer.Init(c.session.GetConnection(), c.session.GetCompany())
	customer.SetRange("no", "C0001", "C0010")
	if customer.FindSet() {
		fmt.Printf("  First record: %s - %s\n", customer.No, customer.Name)

		// Skip forward 3 records
		if customer.Next(3) {
			fmt.Printf("  After Next(3): %s - %s\n", customer.No, customer.Name)
		}
	}

	fmt.Println("\n3. Attempting Backward Navigation (will fail):")
	customer.Init(c.session.GetConnection(), c.session.GetCompany())
	customer.SetRange("no", "C0001", "C0010")
	if customer.FindSet() {
		fmt.Printf("  Current record: %s\n", customer.No)

		// Try to move backward (this will fail with error message)
		if customer.Next(-1) {
			fmt.Printf("  After Next(-1): %s\n", customer.No)
		} else {
			fmt.Println("  ✓ Next(-1) correctly rejected (requires FindSetBuffered)")
		}
	}
}

// demoBidirectionalNavigation demonstrates FindSetBuffered() with Next() for bidirectional iteration
func (c *BidirectionalNavDemo) demoBidirectionalNavigation() {
	fmt.Println("\nUsing FindSetBuffered() for bidirectional navigation:")
	fmt.Println("  - Higher memory usage (all records loaded into buffer)")
	fmt.Println("  - Next(1): Move forward")
	fmt.Println("  - Next(-1): Move backward")
	fmt.Println("  - Next(5): Skip forward 5 records")
	fmt.Println("  - Next(-3): Skip backward 3 records")

	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	// Set filter to limit results
	customer.SetRange("no", "C0001", "C0010")

	fmt.Println("\n1. Load Buffered Recordset:")
	if customer.FindSetBuffered() {
		fmt.Printf("  ✓ Loaded buffered recordset\n")
		fmt.Printf("  First record: %s - %s\n", customer.No, customer.Name)
	}

	fmt.Println("\n2. Move Forward with Next():")
	for i := 0; i < 3 && customer.Next(); i++ {
		fmt.Printf("  Record: %s - %s\n", customer.No, customer.Name)
	}
	fmt.Printf("  Current position: %s\n", customer.No)

	fmt.Println("\n3. Move Backward with Next(-1):")
	if customer.Next(-1) {
		fmt.Printf("  After Next(-1): %s - %s\n", customer.No, customer.Name)
	}

	fmt.Println("\n4. Skip Forward with Next(3):")
	if customer.Next(3) {
		fmt.Printf("  After Next(3): %s - %s\n", customer.No, customer.Name)
	}

	fmt.Println("\n5. Skip Backward with Next(-2):")
	if customer.Next(-2) {
		fmt.Printf("  After Next(-2): %s - %s\n", customer.No, customer.Name)
	}

	fmt.Println("\n6. Navigate to End and Try Beyond (will fail):")
	// Move to last record
	for customer.Next() {
		// Keep moving forward
	}
	fmt.Printf("  Last record: %s - %s\n", customer.No, customer.Name)

	if customer.Next() {
		fmt.Printf("  After Next(): %s\n", customer.No)
	} else {
		fmt.Println("  ✓ Next() correctly returned false (end of recordset)")
	}

	fmt.Println("\n7. Move Back from End:")
	if customer.Next(-1) {
		fmt.Printf("  After Next(-1): %s - %s\n", customer.No, customer.Name)
	}
}

// demoFilteredBufferedRecordset demonstrates using filters to minimize buffer size
func (c *BidirectionalNavDemo) demoFilteredBufferedRecordset() {
	fmt.Println("\nFilters are applied in SQL before loading buffer:")
	fmt.Println("  - SetRange/SetFilter reduce records loaded into memory")
	fmt.Println("  - Only filtered records consume memory")
	fmt.Println("  - Best practice: filter first, then FindSetBuffered()")

	var customer tables.Customer

	fmt.Println("\n1. Without Filter:")
	customer.Init(c.session.GetConnection(), c.session.GetCompany())
	totalCount := customer.Count()
	fmt.Printf("  Total customers in database: %d\n", totalCount)

	fmt.Println("\n2. With SetRange Filter:")
	customer.Init(c.session.GetConnection(), c.session.GetCompany())
	customer.SetRange("no", "C0005", "C0010")
	filteredCount := customer.Count()
	fmt.Printf("  Customers matching filter (C0005..C0010): %d\n", filteredCount)

	fmt.Println("\n3. Load Filtered Buffered Recordset:")
	customer.Init(c.session.GetConnection(), c.session.GetCompany())
	customer.SetRange("no", "C0005", "C0010")
	if customer.FindSetBuffered() {
		fmt.Printf("  ✓ Loaded %d records into buffer (not %d)\n", filteredCount, totalCount)
		fmt.Printf("  Memory saved: Only %d records buffered instead of %d\n",
			filteredCount, totalCount)

		// Navigate through filtered results
		fmt.Println("\n  Buffered records:")
		count := 1
		fmt.Printf("    %d. %s - %s\n", count, customer.No, customer.Name)
		for customer.Next() {
			count++
			fmt.Printf("    %d. %s - %s\n", count, customer.No, customer.Name)
		}
	}

	fmt.Println("\n4. Advanced Filter with SetFilter:")
	customer.Init(c.session.GetConnection(), c.session.GetCompany())
	customer.SetFilter("city", "New York|Chicago") // Only these cities
	cityFilteredCount := customer.Count()
	fmt.Printf("  Customers in New York or Chicago: %d\n", cityFilteredCount)

	if customer.FindSetBuffered() {
		fmt.Printf("  ✓ Buffered only %d city-filtered records\n", cityFilteredCount)
	}
}

// demoPerformanceComparison shows when to use FindSet vs FindSetBuffered
func (c *BidirectionalNavDemo) demoPerformanceComparison() {
	fmt.Println("\nWhen to use FindSet() vs FindSetBuffered():")
	fmt.Println()
	fmt.Println("FindSet() - Forward-Only Streaming:")
	fmt.Println("  ✓ Low memory usage")
	fmt.Println("  ✓ Fast startup (immediate first record)")
	fmt.Println("  ✓ Good for large datasets")
	fmt.Println("  ✓ Good for processing once forward")
	fmt.Println("  ✗ No backward navigation")
	fmt.Println("  ✗ Cannot skip backward")
	fmt.Println()
	fmt.Println("FindSetBuffered() - Bidirectional Navigation:")
	fmt.Println("  ✓ Full bidirectional navigation")
	fmt.Println("  ✓ Can skip forward/backward")
	fmt.Println("  ✓ Multiple passes over data")
	fmt.Println("  ✗ Higher memory usage")
	fmt.Println("  ✗ Slower startup (loads all records)")
	fmt.Println("  ✗ Not suitable for very large datasets")

	fmt.Println("\nTiming Comparison (processing all records once):")

	var customer tables.Customer

	// Time FindSet
	customer.Init(c.session.GetConnection(), c.session.GetCompany())
	customer.SetRange("no", "C0001", "C0020")

	start := time.Now()
	count := 0
	if customer.FindSet() {
		count++
		for customer.Next() {
			count++
		}
	}
	findSetDuration := time.Since(start)
	fmt.Printf("  FindSet():         %v (%d records)\n", findSetDuration, count)

	// Time FindSetBuffered
	customer.Init(c.session.GetConnection(), c.session.GetCompany())
	customer.SetRange("no", "C0001", "C0020")

	start = time.Now()
	count = 0
	if customer.FindSetBuffered() {
		count++
		for customer.Next() {
			count++
		}
	}
	findSetBufferedDuration := time.Since(start)
	fmt.Printf("  FindSetBuffered(): %v (%d records)\n", findSetBufferedDuration, count)

	fmt.Println("\nRecommendation:")
	fmt.Println("  - Use FindSet() when only moving forward (most common)")
	fmt.Println("  - Use FindSetBuffered() when you need backward navigation")
	fmt.Println("  - Always filter first to minimize buffer size")
}

// RunBidirectionalNavDemo is the main entry point for running this codeunit from the application
func RunBidirectionalNavDemo() {
	// Get global session
	sess := session.GetCurrent()
	if sess == nil {
		fmt.Println("✗ Error: No active session")
		return
	}

	// Create codeunit instance
	demo := NewBidirectionalNavDemo(sess)

	// Execute codeunit
	err := demo.RunCLI()
	if err != nil {
		fmt.Printf("\n✗ Error: %v\n", err)
	}

	// Wait for user
	fmt.Print("\nPress Enter to continue...")
	sess.GetScanner().Scan()
}
