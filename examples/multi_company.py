"""
Multi-company example for OpenERP

Demonstrates:
- Creating multiple companies
- Company-specific data isolation
- Separate tables for each company (CompanyName$TableName)
- Global tables shared across companies
"""

from openerp import Database, Company
from openerp.core.crud import CRUDManager


def main():
    print("=== Multi-Company Example ===\n")

    db = Database('multicompany_demo.db')
    crud = CRUDManager(db)

    # Create multiple companies
    print("1. Creating Companies")
    acme = Company.create(db, "ACME")
    print(f"  ✓ {acme.name}")

    globex = Company.create(db, "Globex")
    print(f"  ✓ {globex.name}")

    initech = Company.create(db, "Initech")
    print(f"  ✓ {initech.name}")

    # Create a global settings table
    print("\n2. Creating Global Settings Table")
    db.create_table(
        'SystemSettings',
        {
            'key': 'TEXT NOT NULL UNIQUE',
            'value': 'TEXT',
            'description': 'TEXT'
        },
        is_global=True
    )
    print("  ✓ Created global table: SystemSettings")

    # Insert global settings
    crud.insert('SystemSettings', {
        'key': 'app_version',
        'value': '1.0.0',
        'description': 'Application version'
    })
    crud.insert('SystemSettings', {
        'key': 'default_currency',
        'value': 'USD',
        'description': 'Default system currency'
    })
    print("  ✓ Inserted global settings")

    # Create company-specific customer tables
    print("\n3. Creating Company-Specific Customer Tables")
    for company_name in ["ACME", "Globex", "Initech"]:
        db.create_table(
            'Customers',
            {
                'name': 'TEXT NOT NULL',
                'email': 'TEXT NOT NULL',
                'phone': 'TEXT',
                'country': 'TEXT'
            },
            company_name=company_name
        )
        print(f"  ✓ Created {company_name}$Customers")

    # Insert data for ACME
    print("\n4. Inserting ACME Customers")
    crud.insert('ACME$Customers', {
        'name': 'John Smith',
        'email': 'john@acme-customer.com',
        'phone': '+1-555-0100',
        'country': 'USA'
    })
    crud.insert('ACME$Customers', {
        'name': 'Alice Johnson',
        'email': 'alice@acme-customer.com',
        'phone': '+1-555-0101',
        'country': 'USA'
    })
    print("  ✓ Inserted 2 customers for ACME")

    # Insert data for Globex
    print("\n5. Inserting Globex Customers")
    crud.insert('Globex$Customers', {
        'name': 'Hans Mueller',
        'email': 'hans@globex-customer.de',
        'phone': '+49-555-0200',
        'country': 'Germany'
    })
    crud.insert('Globex$Customers', {
        'name': 'Marie Dubois',
        'email': 'marie@globex-customer.fr',
        'phone': '+33-555-0201',
        'country': 'France'
    })
    crud.insert('Globex$Customers', {
        'name': 'Piet Janssen',
        'email': 'piet@globex-customer.nl',
        'phone': '+31-555-0202',
        'country': 'Netherlands'
    })
    print("  ✓ Inserted 3 customers for Globex")

    # Insert data for Initech
    print("\n6. Inserting Initech Customers")
    crud.insert('Initech$Customers', {
        'name': 'Raj Patel',
        'email': 'raj@initech-customer.in',
        'phone': '+91-555-0300',
        'country': 'India'
    })
    print("  ✓ Inserted 1 customer for Initech")

    # Query each company's data
    print("\n7. Querying Company-Specific Data")

    acme_result = crud.get_all('ACME$Customers')
    print(f"\n  ACME Customers ({acme_result['count']}):")
    for customer in acme_result['records']:
        print(f"    - {customer['name']} ({customer['country']})")

    globex_result = crud.get_all('Globex$Customers')
    print(f"\n  Globex Customers ({globex_result['count']}):")
    for customer in globex_result['records']:
        print(f"    - {customer['name']} ({customer['country']})")

    initech_result = crud.get_all('Initech$Customers')
    print(f"\n  Initech Customers ({initech_result['count']}):")
    for customer in initech_result['records']:
        print(f"    - {customer['name']} ({customer['country']})")

    # Query global settings
    print("\n8. Querying Global Settings (accessible to all companies)")
    settings_result = crud.get_all('SystemSettings')
    print(f"  Total settings: {settings_result['count']}")
    for setting in settings_result['records']:
        print(f"    - {setting['key']}: {setting['value']}")

    # Demonstrate data isolation
    print("\n9. Demonstrating Data Isolation")
    print("  Each company's customer data is physically separated:")
    print(f"    ACME has {acme_result['count']} customers")
    print(f"    Globex has {globex_result['count']} customers")
    print(f"    Initech has {initech_result['count']} customer")
    print("  These are stored in separate tables:")
    print("    - ACME$Customers")
    print("    - Globex$Customers")
    print("    - Initech$Customers")

    # List all companies
    print("\n10. All Companies in System")
    all_companies = Company.list_all(db)
    print(f"  Total companies: {len(all_companies)}")
    for company in all_companies:
        print(f"    - {company.name}")

    # List all tables
    print("\n11. All Tables in Database")
    all_tables = db.list_tables()
    print(f"  Total tables: {len(all_tables)}")
    for table in all_tables:
        if table.startswith('__'):
            print(f"    - {table} (metadata)")
        elif '$' in table:
            print(f"    - {table} (company-specific)")
        else:
            print(f"    - {table} (global)")

    print("\n=== Multi-Company Demo Complete ===")
    print("Key Takeaways:")
    print("  1. Each company has physically separate tables (CompanyName$TableName)")
    print("  2. Complete data isolation at the database level")
    print("  3. Global tables are shared across all companies")
    print("  4. No company can access another company's data")


if __name__ == "__main__":
    main()
