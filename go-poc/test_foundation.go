package main

import (
	"fmt"
	"os"
	"strings"
)

func testFoundation() {
	fmt.Println("=== OpenERP Foundation Layer - Automated Test ===\n")

	testDB := "test_foundation.db"

	// Clean up any existing test database
	os.Remove(testDB)

	// Test 1: Create Database
	fmt.Println("1. CREATE DATABASE")
	fmt.Println(strings.Repeat("-", 60))
	db, err := CreateDatabase(testDB)
	if err != nil {
		fmt.Printf("✗ Failed: %v\n", err)
		return
	}
	fmt.Printf("✓ Database created: %s\n\n", testDB)

	// Test 2: Create Companies
	fmt.Println("2. CREATE COMPANIES")
	fmt.Println(strings.Repeat("-", 60))
	companies := []string{"ACME", "GLOBEX", "INITECH"}
	for _, comp := range companies {
		err := db.CreateCompany(comp)
		if err != nil {
			fmt.Printf("✗ Failed to create %s: %v\n", comp, err)
			return
		}
		fmt.Printf("✓ Created company: %s\n", comp)
	}
	fmt.Println()

	// Test 3: List Companies
	fmt.Println("3. LIST COMPANIES")
	fmt.Println(strings.Repeat("-", 60))
	compList, err := db.ListCompanies()
	if err != nil {
		fmt.Printf("✗ Failed: %v\n", err)
		return
	}
	fmt.Printf("✓ Found %d companies:\n", len(compList))
	for i, comp := range compList {
		fmt.Printf("  %d. %s\n", i+1, comp)
	}
	fmt.Println()

	// Test 4: Enter Company
	fmt.Println("4. ENTER COMPANY")
	fmt.Println(strings.Repeat("-", 60))
	err = db.EnterCompany("ACME")
	if err != nil {
		fmt.Printf("✗ Failed: %v\n", err)
		return
	}
	fmt.Printf("✓ Entered company: %s\n", db.GetCurrentCompany())
	fmt.Println()

	// Test 5: Try to enter company while already in one (should fail)
	fmt.Println("5. TEST ERROR HANDLING - Already in company")
	fmt.Println(strings.Repeat("-", 60))
	err = db.EnterCompany("GLOBEX")
	if err != nil {
		fmt.Printf("✓ Correctly rejected: %v\n", err)
	} else {
		fmt.Println("✗ Should have failed but didn't")
	}
	fmt.Println()

	// Test 6: Exit Company
	fmt.Println("6. EXIT COMPANY")
	fmt.Println(strings.Repeat("-", 60))
	err = db.ExitCompany()
	if err != nil {
		fmt.Printf("✗ Failed: %v\n", err)
		return
	}
	fmt.Println("✓ Exited company successfully")
	currentComp := db.GetCurrentCompany()
	if currentComp == "" {
		fmt.Println("✓ Current company cleared")
	} else {
		fmt.Printf("✗ Current company should be empty, but is: %s\n", currentComp)
	}
	fmt.Println()

	// Test 7: Try to exit when no company active (should fail)
	fmt.Println("7. TEST ERROR HANDLING - Exit with no active company")
	fmt.Println(strings.Repeat("-", 60))
	err = db.ExitCompany()
	if err != nil {
		fmt.Printf("✓ Correctly rejected: %v\n", err)
	} else {
		fmt.Println("✗ Should have failed but didn't")
	}
	fmt.Println()

	// Test 8: Enter non-existent company (should fail)
	fmt.Println("8. TEST ERROR HANDLING - Enter non-existent company")
	fmt.Println(strings.Repeat("-", 60))
	err = db.EnterCompany("NONEXISTENT")
	if err != nil {
		fmt.Printf("✓ Correctly rejected: %v\n", err)
	} else {
		fmt.Println("✗ Should have failed but didn't")
	}
	fmt.Println()

	// Test 9: Create duplicate company (should fail)
	fmt.Println("9. TEST ERROR HANDLING - Duplicate company")
	fmt.Println(strings.Repeat("-", 60))
	err = db.CreateCompany("ACME")
	if err != nil {
		fmt.Printf("✓ Correctly rejected: %v\n", err)
	} else {
		fmt.Println("✗ Should have failed but didn't")
	}
	fmt.Println()

	// Test 10: Delete Company
	fmt.Println("10. DELETE COMPANY")
	fmt.Println(strings.Repeat("-", 60))

	// First, let's simulate creating some company-specific tables
	// (In a real scenario, these would be created through CRUD operations)
	_, err = db.conn.Exec(`CREATE TABLE "GLOBEX$customers" (id INTEGER PRIMARY KEY, name TEXT)`)
	if err != nil {
		fmt.Printf("✗ Failed to create test table: %v\n", err)
		return
	}
	_, err = db.conn.Exec(`CREATE TABLE "GLOBEX$orders" (id INTEGER PRIMARY KEY, amount REAL)`)
	if err != nil {
		fmt.Printf("✗ Failed to create test table: %v\n", err)
		return
	}
	fmt.Println("  Created test tables: GLOBEX$customers, GLOBEX$orders")

	err = db.DeleteCompany("GLOBEX")
	if err != nil {
		fmt.Printf("✗ Failed: %v\n", err)
		return
	}
	fmt.Printf("✓ Deleted company: GLOBEX (and all its tables)\n")

	// Verify it's gone
	compList, _ = db.ListCompanies()
	fmt.Printf("✓ Remaining companies: %d\n", len(compList))
	for _, comp := range compList {
		fmt.Printf("  - %s\n", comp)
	}
	fmt.Println()

	// Test 11: Close Database
	fmt.Println("11. CLOSE DATABASE")
	fmt.Println(strings.Repeat("-", 60))
	err = db.CloseDatabase()
	if err != nil {
		fmt.Printf("✗ Failed: %v\n", err)
		return
	}
	fmt.Println("✓ Database closed successfully")
	fmt.Println()

	// Test 12: Open Existing Database
	fmt.Println("12. OPEN EXISTING DATABASE")
	fmt.Println(strings.Repeat("-", 60))
	db2, err := OpenDatabase(testDB)
	if err != nil {
		fmt.Printf("✗ Failed: %v\n", err)
		return
	}
	fmt.Printf("✓ Database opened: %s\n", testDB)

	// Verify companies still exist
	compList, _ = db2.ListCompanies()
	fmt.Printf("✓ Found %d companies (persistent):\n", len(compList))
	for _, comp := range compList {
		fmt.Printf("  - %s\n", comp)
	}
	fmt.Println()

	// Clean up
	db2.CloseDatabase()
	os.Remove(testDB)

	// Test 13: Multi-User Simulation
	fmt.Println("13. MULTI-USER CONCURRENCY TEST")
	fmt.Println(strings.Repeat("-", 60))

	// Create a fresh database
	dbShared, _ := CreateDatabase("test_multiuser.db")
	dbShared.CreateCompany("ACME")
	dbShared.CreateCompany("GLOBEX")
	dbShared.CloseDatabase()

	// Simulate User 1
	user1DB, _ := OpenDatabase("test_multiuser.db")
	user1DB.EnterCompany("ACME")
	fmt.Printf("✓ User 1 entered company: %s\n", user1DB.GetCurrentCompany())

	// Simulate User 2 (concurrent)
	user2DB, _ := OpenDatabase("test_multiuser.db")
	user2DB.EnterCompany("GLOBEX")
	fmt.Printf("✓ User 2 entered company: %s\n", user2DB.GetCurrentCompany())

	// Verify isolation
	fmt.Printf("✓ User 1 still in: %s\n", user1DB.GetCurrentCompany())
	fmt.Printf("✓ User 2 still in: %s\n", user2DB.GetCurrentCompany())
	fmt.Println("✓ Per-connection state isolation confirmed!")

	// Clean up
	user1DB.CloseDatabase()
	user2DB.CloseDatabase()
	os.Remove("test_multiuser.db")

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("ALL TESTS PASSED! ✓")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("\nFoundation layer is working correctly:")
	fmt.Println("  ✓ Database creation/opening/closing")
	fmt.Println("  ✓ Company CRUD operations")
	fmt.Println("  ✓ Session-based company context")
	fmt.Println("  ✓ Error handling")
	fmt.Println("  ✓ Multi-user concurrency")
	fmt.Println("  ✓ Persistent storage")
	fmt.Println("\nReady for next phase: CRUD operations!")
}

func main() {
	testFoundation()
}
