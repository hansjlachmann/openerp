"""
NAV-style Record API example

Demonstrates NAV/C-AL style record operations:
- Customer.get(id) - Like NAV's customer.GET('1000')
- customer.insert() - Like NAV's customer.INSERT
- customer.modify() - Like NAV's customer.MODIFY
- customer.delete() - Like NAV's customer.DELETE
- Customer.find_set() - Like NAV's customer.FINDSET
"""

from openerp import Database, Company
from openerp.models.record import Record


# Define a Customer record type (like NAV's Customer table)
class Customer(Record):
    TABLE_NAME = 'customers'
    COMPANY_NAME = 'ACME'


def main():
    print("=== NAV-Style Record API Demo ===\n")

    # Setup
    db = Database(':memory:')
    Company.create(db, "ACME")
    db.create_table(
        'customers',
        {
            'name': 'TEXT NOT NULL',
            'email': 'TEXT',
            'phone': 'TEXT',
            'balance': 'REAL DEFAULT 0'
        },
        company_name="ACME"
    )

    # Example 1: INSERT (like NAV: customer.INSERT)
    print("1. INSERT - Creating new customer")
    print("-" * 50)

    customer = Customer(db)
    customer['name'] = 'John Doe'
    customer['email'] = 'john@example.com'
    customer['phone'] = '+1-555-0100'
    customer['balance'] = 1000.0

    if customer.insert():
        print(f"✓ Customer created with ID: {customer['id']}")
        print(f"  Name: {customer['name']}")
        print(f"  Email: {customer['email']}")
        print(f"  Balance: ${customer['balance']}")
        customer_id = customer['id']
    else:
        print("✗ Failed to create customer")
        return

    # Example 2: GET (like NAV: customer.GET('1000'))
    print("\n2. GET - Retrieving customer by ID")
    print("-" * 50)

    customer = Customer.get(db, customer_id)
    if customer:
        print(f"✓ Found customer ID {customer_id}")
        print(f"  Name: {customer['name']}")
        print(f"  Email: {customer['email']}")
        print(f"  Balance: ${customer['balance']}")
    else:
        print(f"✗ Customer {customer_id} not found")

    # Example 3: MODIFY (like NAV: customer.MODIFY)
    print("\n3. MODIFY - Updating customer")
    print("-" * 50)

    customer = Customer.get(db, customer_id)
    print(f"  Before: Balance = ${customer['balance']}")

    customer['balance'] = 2500.0
    customer['email'] = 'john.doe@example.com'

    if customer.modify():
        print(f"  After: Balance = ${customer['balance']}")
        print(f"  After: Email = {customer['email']}")
        print("✓ Customer updated successfully")
    else:
        print("✗ Failed to update customer")

    # Example 4: FINDSET - Getting multiple records
    print("\n4. FINDSET - Creating and finding multiple customers")
    print("-" * 50)

    # Create more customers
    customers_data = [
        {'name': 'Jane Smith', 'email': 'jane@example.com', 'balance': 3000.0},
        {'name': 'Bob Wilson', 'email': 'bob@example.com', 'balance': 1500.0},
        {'name': 'Alice Brown', 'email': 'alice@example.com', 'balance': 500.0},
    ]

    for data in customers_data:
        customer = Customer(db)
        customer['name'] = data['name']
        customer['email'] = data['email']
        customer['balance'] = data['balance']
        customer.insert()

    # Get all customers
    all_customers = Customer.find_set(db)
    print(f"✓ Found {len(all_customers)} customers:")
    for cust in all_customers:
        print(f"  - {cust['name']}: {cust['email']} (${cust['balance']})")

    # Example 5: FINDFIRST - Finding with filter
    print("\n5. FINDFIRST - Finding specific customer")
    print("-" * 50)

    customer = Customer.find_first(db, {'name': 'Jane Smith'})
    if customer:
        print(f"✓ Found: {customer['name']}")
        print(f"  Email: {customer['email']}")
        print(f"  Balance: ${customer['balance']}")
    else:
        print("✗ Customer not found")

    # Example 6: FINDSET with filter
    print("\n6. FINDSET with filter - High balance customers")
    print("-" * 50)

    # Note: Currently only supports exact match filters
    # For complex queries, use SQL or CRUD directly
    print("All customers with balance info:")
    for cust in Customer.find_set(db):
        if cust['balance'] >= 2000:
            print(f"  ✓ High value: {cust['name']} - ${cust['balance']}")

    # Example 7: DELETE (like NAV: customer.DELETE)
    print("\n7. DELETE - Removing a customer")
    print("-" * 50)

    customer = Customer.find_first(db, {'name': 'Bob Wilson'})
    if customer:
        print(f"  Deleting: {customer['name']}")
        if customer.delete():
            print("✓ Customer deleted successfully")

            # Verify deletion
            all_customers = Customer.find_set(db)
            print(f"  Remaining customers: {len(all_customers)}")
        else:
            print("✗ Failed to delete customer")

    print("\n" + "="*50)
    print("Demo Complete!")
    print("="*50)
    print("\nComparison with NAV C/AL:")
    print("  NAV: customer.GET('1000')      → Python: Customer.get(db, 1)")
    print("  NAV: customer.INSERT           → Python: customer.insert()")
    print("  NAV: customer.MODIFY           → Python: customer.modify()")
    print("  NAV: customer.DELETE           → Python: customer.delete()")
    print("  NAV: customer.FINDSET          → Python: Customer.find_set(db)")
    print("  NAV: customer.FINDFIRST        → Python: Customer.find_first(db, filters)")


if __name__ == "__main__":
    main()
