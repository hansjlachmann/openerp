"""
Basic usage example for OpenERP Phase 1

This example demonstrates:
- Creating a database
- Creating a company
- Creating tables with triggers
- CRUD operations
"""

from openerp import Database, Company
from openerp.core.crud import CRUDManager


def main():
    # Initialize database
    print("=== OpenERP Phase 1 Demo ===\n")

    db = Database('demo.db')
    print("✓ Database initialized")

    # Create a company
    company = Company.create(
        db,
        code="DEMO",
        name="Demo Corporation",
        legal_name="Demo Corporation Inc.",
        tax_id="12-3456789",
        currency="USD"
    )
    print(f"✓ Created company: {company.name} ({company.code})")

    # Create a customers table with ON_INSERT trigger
    db.create_table(
        'customers',
        {
            'name': 'TEXT NOT NULL',
            'email': 'TEXT UNIQUE',
            'phone': 'TEXT',
            'balance': 'REAL DEFAULT 0'
        },
        company_id=company.id,
        on_insert="""
# Auto-format email to lowercase
if record.get('email'):
    record['email'] = record['email'].lower()

# Initialize balance
if 'balance' not in record:
    record['balance'] = 0.0

print(f"New customer added: {record['name']}")
"""
    )
    print("✓ Created 'customers' table with ON_INSERT trigger")

    # Create an orders table with ON_INSERT trigger
    db.create_table(
        'orders',
        {
            'customer_id': 'INTEGER NOT NULL',
            'amount': 'REAL NOT NULL',
            'status': 'TEXT',
            'order_date': 'TIMESTAMP'
        },
        company_id=company.id,
        on_insert="""
from datetime import datetime

# Set default status
record['status'] = 'pending'

# Set order date
record['order_date'] = datetime.now().isoformat()

print(f"Order created: ${record['amount']:.2f} - Status: {record['status']}")
"""
    )
    print("✓ Created 'orders' table with ON_INSERT trigger\n")

    # CRUD operations
    crud = CRUDManager(db)

    # Insert customers
    print("=== Inserting Customers ===")
    customer1 = crud.insert('customers', {
        'name': 'John Doe',
        'email': 'JOHN@EXAMPLE.COM',  # Will be lowercased by trigger
        'phone': '+1-555-0100'
    }, company_id=company.id)
    print(f"✓ Inserted customer ID: {customer1['id']}")
    print(f"  Email (after trigger): {customer1['record']['email']}")

    customer2 = crud.insert('customers', {
        'name': 'Jane Smith',
        'email': 'jane@example.com',
        'phone': '+1-555-0101'
    }, company_id=company.id)
    print(f"✓ Inserted customer ID: {customer2['id']}\n")

    # Insert orders
    print("=== Creating Orders ===")
    order1 = crud.insert('orders', {
        'customer_id': customer1['id'],
        'amount': 150.00
    }, company_id=company.id)
    print(f"✓ Order ID: {order1['id']}")
    print(f"  Status (set by trigger): {order1['record']['status']}")
    print(f"  Date (set by trigger): {order1['record']['order_date']}")

    order2 = crud.insert('orders', {
        'customer_id': customer2['id'],
        'amount': 275.50
    }, company_id=company.id)
    print(f"✓ Order ID: {order2['id']}\n")

    # Read operations
    print("=== Reading Data ===")
    all_customers = crud.get_all('customers', company_id=company.id)
    print(f"Total customers: {all_customers['count']}")
    for customer in all_customers['records']:
        print(f"  - {customer['name']} ({customer['email']})")

    all_orders = crud.get_all('orders', company_id=company.id)
    print(f"\nTotal orders: {all_orders['count']}")
    for order in all_orders['records']:
        print(f"  - Order #{order['id']}: ${order['amount']:.2f} - {order['status']}")

    # Update operation
    print("\n=== Updating Data ===")
    crud.update('customers', customer1['id'], {'phone': '+1-555-9999'})
    updated = crud.get_by_id('customers', customer1['id'])
    print(f"✓ Updated customer phone: {updated['record']['phone']}")

    # Search operation
    print("\n=== Searching Data ===")
    search_results = crud.search('customers', {'name': 'Jane Smith'}, company_id=company.id)
    print(f"Search results for 'Jane Smith': {search_results['count']} found")

    # List all tables
    print("\n=== Table Summary ===")
    tables = db.list_tables(company_id=company.id)
    print(f"Tables for company {company.code}:")
    for table in tables:
        print(f"  - {table}")

    print("\n=== Demo Complete ===")
    db.close()


if __name__ == '__main__':
    main()
