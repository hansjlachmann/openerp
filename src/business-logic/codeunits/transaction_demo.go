package codeunits

import (
	"database/sql"
	"fmt"

	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/session"
	"github.com/hansjlachmann/openerp/src/foundation/types"
)

// TransactionDemo demonstrates transaction management with Commit and Rollback
// Codeunit ID: 50013
func TransactionDemo(db *sql.DB, company string) {
	// Get current session
	sess := session.GetCurrent()
	if sess == nil {
		fmt.Println("✗ No active session")
		return
	}

	fmt.Println("========================================")
	fmt.Println("Codeunit 50013 - Transaction Demo")
	fmt.Println("========================================\n")

	// Demo 1: Successful transaction with COMMIT
	fmt.Println("Demo 1: Successful Transaction (COMMIT)")
	fmt.Println("========================================")
	demo1SuccessfulCommit(sess, company)

	fmt.Println()

	// Demo 2: Failed transaction with ROLLBACK
	fmt.Println("Demo 2: Failed Transaction (ROLLBACK)")
	fmt.Println("========================================")
	demo2FailedRollback(sess, company)

	fmt.Println()

	// Demo 3: Multiple operations in one transaction
	fmt.Println("Demo 3: Multiple Operations in One Transaction")
	fmt.Println("========================================")
	demo3MultipleOperations(sess, company)

	fmt.Println("\n========================================")
	fmt.Println("Transaction Demo Complete!")
	fmt.Println("========================================")
}

func demo1SuccessfulCommit(sess *session.Session, company string) {
	fmt.Println("Creating a customer within a transaction...")

	// Begin transaction
	if err := sess.BeginTransaction(); err != nil {
		fmt.Printf("✗ Failed to begin transaction: %v\n", err)
		return
	}
	fmt.Println("✓ Transaction started")

	// Create customer using the transaction executor
	cust := tables.NewCustomer()
	cust.Init(sess.GetExecutor(), company)
	cust.No = types.NewCode("TX-CUST-001")
	cust.Name = types.NewText("Transaction Test Customer 1")
	cust.Status = tables.Customer_Status.Open
	cust.Credit_limit = types.NewDecimal(50000.00)
	cust.Last_order_date = types.Today()
	cust.Created_at = types.Now()

	if !cust.Insert(false) {
		fmt.Println("✗ Failed to insert customer")
		sess.Rollback()
		return
	}
	fmt.Printf("✓ Customer %s created (not yet committed)\n", cust.No.String())

	// Check if customer exists before commit (should exist in transaction)
	fmt.Println("  Checking if customer exists in transaction...")
	checkCust := tables.NewCustomer()
	checkCust.Init(sess.GetExecutor(), company)
	if checkCust.Get(types.NewCode("TX-CUST-001")) {
		fmt.Println("  ✓ Customer visible within transaction")
	}

	// Commit the transaction
	if err := sess.Commit(); err != nil {
		fmt.Printf("✗ Failed to commit: %v\n", err)
		return
	}
	fmt.Println("✓ Transaction committed successfully")

	// Verify customer exists after commit
	verifyCust := tables.NewCustomer()
	verifyCust.Init(sess.GetConnection(), company)
	if verifyCust.Get(types.NewCode("TX-CUST-001")) {
		fmt.Printf("✓ Customer persisted: %s - %s\n", verifyCust.No.String(), verifyCust.Name.String())
	}
}

func demo2FailedRollback(sess *session.Session, company string) {
	fmt.Println("Creating a customer, then rolling back...")

	// Begin transaction
	if err := sess.BeginTransaction(); err != nil {
		fmt.Printf("✗ Failed to begin transaction: %v\n", err)
		return
	}
	fmt.Println("✓ Transaction started")

	// Create customer
	cust := tables.NewCustomer()
	cust.Init(sess.GetExecutor(), company)
	cust.No = types.NewCode("TX-CUST-ROLLBACK")
	cust.Name = types.NewText("This Will Be Rolled Back")
	cust.Status = tables.Customer_Status.Open
	cust.Credit_limit = types.NewDecimal(25000.00)
	cust.Last_order_date = types.Today()
	cust.Created_at = types.Now()

	if !cust.Insert(false) {
		fmt.Println("✗ Failed to insert customer")
		sess.Rollback()
		return
	}
	fmt.Printf("✓ Customer %s created (in transaction)\n", cust.No.String())

	// Simulate an error condition
	fmt.Println("  Simulating error condition...")
	fmt.Println("  Rolling back transaction...")

	// Rollback the transaction
	if err := sess.Rollback(); err != nil {
		fmt.Printf("✗ Failed to rollback: %v\n", err)
		return
	}
	fmt.Println("✓ Transaction rolled back successfully")

	// Verify customer does NOT exist after rollback
	verifyCust := tables.NewCustomer()
	verifyCust.Init(sess.GetConnection(), company)
	if !verifyCust.Get(types.NewCode("TX-CUST-ROLLBACK")) {
		fmt.Println("✓ Customer was not persisted (rollback successful)")
	} else {
		fmt.Println("✗ Customer still exists (rollback failed!)")
	}
}

func demo3MultipleOperations(sess *session.Session, company string) {
	fmt.Println("Creating multiple customers in one transaction...")

	// Begin transaction
	if err := sess.BeginTransaction(); err != nil {
		fmt.Printf("✗ Failed to begin transaction: %v\n", err)
		return
	}
	fmt.Println("✓ Transaction started")

	successCount := 0
	customersToCreate := []struct {
		code string
		name string
	}{
		{"TX-BATCH-001", "Batch Customer 1"},
		{"TX-BATCH-002", "Batch Customer 2"},
		{"TX-BATCH-003", "Batch Customer 3"},
	}

	// Insert multiple customers in the same transaction
	for _, custData := range customersToCreate {
		cust := tables.NewCustomer()
		cust.Init(sess.GetExecutor(), company)
		cust.No = types.NewCode(custData.code)
		cust.Name = types.NewText(custData.name)
		cust.Status = tables.Customer_Status.Open
		cust.Credit_limit = types.NewDecimal(10000.00)
		cust.Last_order_date = types.Today()
		cust.Created_at = types.Now()

		if !cust.Insert(false) {
			fmt.Printf("✗ Failed to insert customer %s\n", custData.code)
			fmt.Println("  Rolling back all changes...")
			sess.Rollback()
			return
		}
		fmt.Printf("  ✓ Created %s\n", custData.code)
		successCount++
	}

	// All inserts succeeded, commit
	if err := sess.Commit(); err != nil {
		fmt.Printf("✗ Failed to commit: %v\n", err)
		return
	}
	fmt.Printf("✓ Transaction committed: %d customers created\n", successCount)

	// Verify all customers exist
	fmt.Println("\nVerification:")
	for _, custData := range customersToCreate {
		verifyCust := tables.NewCustomer()
		verifyCust.Init(sess.GetConnection(), company)
		if verifyCust.Get(types.NewCode(custData.code)) {
			fmt.Printf("  ✓ %s exists\n", custData.code)
		} else {
			fmt.Printf("  ✗ %s not found\n", custData.code)
		}
	}
}
