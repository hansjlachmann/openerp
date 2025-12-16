"""
Create a sample database file for exploration

This creates a persistent SQLite database file (sample.db) that you can
explore using SQL queries or SQLite tools.
"""

from openerp import Database, Company
from openerp.core.crud import CRUDManager


def main():
    print("Creating sample database...\n")

    # Create persistent database file
    db = Database('sample.db')
    print("✓ Database file: sample.db")

    # Create companies
    company1 = Company.create(db, "ACME")
    company2 = Company.create(db, "Globex")
    print(f"✓ Created companies: {company1.name}, {company2.name}")

    # Create customers table for ACME
    db.create_table(
        'customers',
        {
            'name': 'TEXT NOT NULL',
            'email': 'TEXT',
            'phone': 'TEXT',
            'balance': 'REAL DEFAULT 0'
        },
        company_name="ACME",
        on_insert="""
if record.get('email'):
    record['email'] = record['email'].lower()
if 'balance' not in record:
    record['balance'] = 0.0
"""
    )
    print("✓ Created ACME$customers table")

    # Create customers table for Globex
    db.create_table(
        'customers',
        {
            'name': 'TEXT NOT NULL',
            'email': 'TEXT',
            'phone': 'TEXT',
            'balance': 'REAL DEFAULT 0'
        },
        company_name="Globex",
        on_insert="""
if record.get('email'):
    record['email'] = record['email'].lower()
if 'balance' not in record:
    record['balance'] = 0.0
"""
    )
    print("✓ Created Globex$customers table")

    # Add some translations
    db.set_table_translation('ACME$customers', 'es', 'Clientes')
    db.set_field_translation('ACME$customers', 'name', 'es', 'Nombre')
    db.set_field_translation('ACME$customers', 'email', 'es', 'Correo')
    print("✓ Added Spanish translations")

    # Insert sample data
    crud = CRUDManager(db)

    # ACME customers
    crud.insert('ACME$customers', {
        'name': 'John Smith',
        'email': 'JOHN@ACME.COM',
        'phone': '+1-555-0100',
        'balance': 1000.0
    })
    crud.insert('ACME$customers', {
        'name': 'Alice Johnson',
        'email': 'alice@acme.com',
        'phone': '+1-555-0101',
        'balance': 2500.0
    })
    print("✓ Inserted 2 ACME customers")

    # Globex customers
    crud.insert('Globex$customers', {
        'name': 'Bob Wilson',
        'email': 'BOB@GLOBEX.COM',
        'phone': '+1-555-0200',
        'balance': 500.0
    })
    crud.insert('Globex$customers', {
        'name': 'Carol Martinez',
        'email': 'carol@globex.com',
        'phone': '+1-555-0201',
        'balance': 1500.0
    })
    crud.insert('Globex$customers', {
        'name': 'David Lee',
        'email': 'david@globex.com',
        'phone': '+1-555-0202'
    })
    print("✓ Inserted 3 Globex customers")

    print("\n=== Database Created Successfully ===")
    print("File: sample.db")
    print("\nTo explore the database, use:")
    print("  sqlite3 sample.db")
    print("\nOr use the query helper:")
    print("  python examples/query_db.py")


if __name__ == "__main__":
    main()
