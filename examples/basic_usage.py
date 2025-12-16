"""
Basic usage example for OpenERP Phase 1

This example demonstrates:
- Creating a database
- Creating a company
- Creating company-specific tables with triggers
- CRUD operations
- Multi-language translations
"""

from openerp import Database, Company
from openerp.core.crud import CRUDManager


def main():
    # Initialize database
    print("=== OpenERP Phase 1 Demo ===\n")

    db = Database('demo.db')
    print("✓ Database initialized")

    # Create a company
    company = Company.create(db, "DemoCorp")
    print(f"✓ Created company: {company.name}")

    # Create a customers table with ON_INSERT trigger
    db.create_table(
        'customers',
        {
            'name': 'TEXT NOT NULL',
            'email': 'TEXT',
            'phone': 'TEXT',
            'balance': 'REAL DEFAULT 0'
        },
        company_name="DemoCorp",
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
    print("✓ Created 'DemoCorp$customers' table with ON_INSERT trigger")

    # Add translations for the customers table
    db.set_table_translation('DemoCorp$customers', 'es', 'Clientes')
    db.set_field_translation('DemoCorp$customers', 'name', 'es', 'Nombre')
    db.set_field_translation('DemoCorp$customers', 'email', 'es', 'Correo')
    db.set_field_translation('DemoCorp$customers', 'phone', 'es', 'Teléfono')
    db.set_field_translation('DemoCorp$customers', 'balance', 'es', 'Saldo')
    print("✓ Added Spanish translations")

    # Create an orders table with ON_INSERT trigger
    db.create_table(
        'orders',
        {
            'customer_id': 'INTEGER NOT NULL',
            'amount': 'REAL NOT NULL',
            'status': 'TEXT',
            'order_date': 'TIMESTAMP'
        },
        company_name="DemoCorp",
        on_insert="""
from datetime import datetime

# Set default status
record['status'] = 'pending'

# Set order date
record['order_date'] = datetime.now().isoformat()

print(f"Order created: ${record['amount']:.2f} - Status: {record['status']}")
"""
    )
    print("✓ Created 'DemoCorp$orders' table with ON_INSERT trigger\n")

    # CRUD operations
    crud = CRUDManager(db)

    # Insert customers
    print("=== Inserting Customers ===")
    result1 = crud.insert('DemoCorp$customers', {
        'name': 'John Doe',
        'email': 'JOHN@EXAMPLE.COM',  # Will be lowercased by trigger
        'phone': '+1-555-0100'
    })
    if result1['success']:
        print(f"✓ Inserted customer ID: {result1['record']['id']}")
        customer1_id = result1['record']['id']

    result2 = crud.insert('DemoCorp$customers', {
        'name': 'Jane Smith',
        'email': 'jane@example.com',
        'phone': '+1-555-0200',
        'balance': 1000.0
    })
    if result2['success']:
        print(f"✓ Inserted customer ID: {result2['record']['id']}")
        customer2_id = result2['record']['id']

    # Insert orders
    print("\n=== Inserting Orders ===")
    order_result = crud.insert('DemoCorp$orders', {
        'customer_id': customer1_id,
        'amount': 299.99
    })
    if order_result['success']:
        print(f"✓ Inserted order ID: {order_result['record']['id']}")

    # Query all customers
    print("\n=== Querying Customers ===")
    customers_result = crud.get_all('DemoCorp$customers')
    if customers_result['success']:
        print(f"Total customers: {customers_result['count']}")
        for customer in customers_result['records']:
            print(f"  - {customer['name']}: {customer['email']} (Balance: ${customer['balance']:.2f})")

    # Query orders
    print("\n=== Querying Orders ===")
    orders_result = crud.get_all('DemoCorp$orders')
    if orders_result['success']:
        print(f"Total orders: {orders_result['count']}")
        for order in orders_result['records']:
            print(f"  - Order #{order['id']}: ${order['amount']:.2f} - Status: {order['status']}")

    # Search customers
    print("\n=== Searching Customers ===")
    search_result = crud.search('DemoCorp$customers', {'name': 'John Doe'})
    if search_result['success']:
        print(f"Found {search_result['count']} customer(s) matching 'John Doe'")
        for customer in search_result['records']:
            print(f"  - {customer['name']}: {customer['email']}")

    # Update customer
    print("\n=== Updating Customer ===")
    update_result = crud.update('DemoCorp$customers', customer1_id, {
        'balance': 500.0
    })
    if update_result['success']:
        print(f"✓ Updated customer {customer1_id} balance to $500.00")

    # Get updated customer
    get_result = crud.get_by_id('DemoCorp$customers', customer1_id)
    if get_result['success']:
        updated = get_result['record']
        print(f"  Verified: {updated['name']} - Balance: ${updated['balance']:.2f}")

    # Show translations
    print("\n=== Translations ===")
    table_es = db.get_table_translation('DemoCorp$customers', 'es')
    name_es = db.get_field_translation('DemoCorp$customers', 'name', 'es')
    email_es = db.get_field_translation('DemoCorp$customers', 'email', 'es')
    print(f"Table 'customers' in Spanish: {table_es}")
    print(f"Field 'name' in Spanish: {name_es}")
    print(f"Field 'email' in Spanish: {email_es}")

    print("\n=== Demo Complete ===")
    print(f"Database file: demo.db")
    print(f"Tables created: DemoCorp$customers, DemoCorp$orders")


if __name__ == "__main__":
    main()
