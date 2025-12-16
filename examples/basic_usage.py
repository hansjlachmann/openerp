"""
Basic usage example for OpenERP Phase 1

This example demonstrates:
- Creating a database
- Creating a company
- Creating company-specific customer table with triggers
- CRUD operations on customers
- Multi-language translations
"""

from openerp import Database, Company
from openerp.core.crud import CRUDManager


def main():
    # Initialize database
    print("=== OpenERP Phase 1 Demo ===\n")

    db = Database(':memory:')  # Use in-memory database for clean runs
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

# Initialize balance if not provided
if 'balance' not in record:
    record['balance'] = 0.0

name = record.get('name', 'Unknown')
print("New customer added: " + name)
"""
    )
    print("✓ Created 'DemoCorp$customers' table with ON_INSERT trigger")

    # Add translations for the customers table
    db.set_table_translation('DemoCorp$customers', 'es', 'Clientes')
    db.set_field_translation('DemoCorp$customers', 'name', 'es', 'Nombre')
    db.set_field_translation('DemoCorp$customers', 'email', 'es', 'Correo')
    db.set_field_translation('DemoCorp$customers', 'phone', 'es', 'Teléfono')
    db.set_field_translation('DemoCorp$customers', 'balance', 'es', 'Saldo')
    print("✓ Added Spanish translations\n")

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
        print(f"✓ Inserted customer: {result1['record']['name']}")
        customer1_id = result1['record']['id']
    else:
        print(f"✗ Failed to insert customer: {result1.get('errors', ['Unknown error'])}")
        customer1_id = None

    result2 = crud.insert('DemoCorp$customers', {
        'name': 'Jane Smith',
        'email': 'jane@example.com',
        'phone': '+1-555-0200',
        'balance': 1000.0
    })
    if result2['success']:
        print(f"✓ Inserted customer: {result2['record']['name']}")
        customer2_id = result2['record']['id']
    else:
        print(f"✗ Failed to insert customer: {result2.get('errors', ['Unknown error'])}")
        customer2_id = None

    result3 = crud.insert('DemoCorp$customers', {
        'name': 'Bob Johnson',
        'email': 'BOB@EXAMPLE.COM',
        'phone': '+1-555-0300',
        'balance': 250.0
    })
    if result3['success']:
        print(f"✓ Inserted customer: {result3['record']['name']}")

    # Query all customers
    print("\n=== Querying All Customers ===")
    customers_result = crud.get_all('DemoCorp$customers')
    if customers_result['success']:
        print(f"Total customers: {customers_result['count']}")
        for customer in customers_result['records']:
            print(f"  - {customer['name']}: {customer['email']} (Balance: ${customer['balance']:.2f})")

    # Search customers
    print("\n=== Searching Customers ===")
    search_result = crud.search('DemoCorp$customers', {'name': 'John Doe'})
    if search_result['success']:
        print(f"Found {search_result['count']} customer(s) matching 'John Doe'")
        for customer in search_result['records']:
            print(f"  - {customer['name']}: {customer['email']}")

    # Update customer
    if customer1_id:
        print("\n=== Updating Customer ===")
        update_result = crud.update('DemoCorp$customers', customer1_id, {
            'balance': 500.0
        })
        if update_result['success']:
            print(f"✓ Updated {update_result['record']['name']} balance to ${update_result['record']['balance']:.2f}")

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
    balance_es = db.get_field_translation('DemoCorp$customers', 'balance', 'es')
    print(f"Table 'customers' in Spanish: {table_es}")
    print(f"Field 'name' in Spanish: {name_es}")
    print(f"Field 'email' in Spanish: {email_es}")
    print(f"Field 'balance' in Spanish: {balance_es}")

    print("\n=== Demo Complete ===")
    print("Database: In-memory (not persisted)")
    print("Table created: DemoCorp$customers")
    print("All CRUD operations completed successfully")


if __name__ == "__main__":
    main()
