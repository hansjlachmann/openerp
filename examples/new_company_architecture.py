"""
Simple example demonstrating the new Company architecture

This example shows:
- Global Company table with Name as PRIMARY KEY
- Company-specific tables using CompanyName$TableName format
- Global tables vs company-specific tables
"""

from openerp import Database, Company
from openerp.core.crud import CRUDManager


def main():
    print("=== OpenERP Company Architecture Demo ===\n")

    # Initialize database
    db = Database(':memory:')
    crud = CRUDManager(db)

    # 1. Create Companies
    print("1. Creating Companies")
    print("-" * 60)

    acme = Company.create(db, "ACME")
    print(f"✓ Created company: {acme.name}")

    techcorp = Company.create(db, "TechCorp")
    print(f"✓ Created company: {techcorp.name}")

    # List all companies
    companies = Company.list_all(db)
    print(f"\nTotal companies: {len(companies)}")
    for company in companies:
        print(f"  - {company.name}")

    # 2. Create Global Table
    print("\n2. Creating Global Table (SystemSettings)")
    print("-" * 60)

    db.create_table(
        'SystemSettings',
        {
            'key': 'TEXT NOT NULL UNIQUE',
            'value': 'TEXT'
        },
        is_global=True
    )
    print("✓ Created global table: SystemSettings")

    # Insert into global table
    crud.insert('SystemSettings', {
        'key': 'app_version',
        'value': '1.0.0'
    })
    print("✓ Inserted into SystemSettings")

    # 3. Create Company-Specific Tables
    print("\n3. Creating Company-Specific Tables")
    print("-" * 60)

    # ACME$Customers
    db.create_table(
        'Customers',
        {
            'name': 'TEXT NOT NULL',
            'email': 'TEXT',
            'balance': 'REAL DEFAULT 0'
        },
        company_name='ACME',
        on_insert="""
# Auto-lowercase email
if record.get('email'):
    record['email'] = record['email'].lower()
print(f"ACME: New customer {record['name']}")
"""
    )
    print("✓ Created: ACME$Customers")

    # TechCorp$Customers
    db.create_table(
        'Customers',
        {
            'name': 'TEXT NOT NULL',
            'email': 'TEXT',
            'balance': 'REAL DEFAULT 0'
        },
        company_name='TechCorp',
        on_insert="""
if record.get('email'):
    record['email'] = record['email'].lower()
print(f"TechCorp: New customer {record['name']}")
"""
    )
    print("✓ Created: TechCorp$Customers")

    # 4. Insert Data into Company-Specific Tables
    print("\n4. Inserting Company-Specific Data")
    print("-" * 60)

    # ACME customers
    crud.insert('ACME$Customers', {
        'name': 'Alice Johnson',
        'email': 'ALICE@ACME.COM',
        'balance': 1000.0
    })
    crud.insert('ACME$Customers', {
        'name': 'Bob Smith',
        'email': 'BOB@ACME.COM',
        'balance': 2000.0
    })
    print("✓ Inserted 2 customers into ACME$Customers")

    # TechCorp customers
    crud.insert('TechCorp$Customers', {
        'name': 'Charlie Brown',
        'email': 'CHARLIE@TECHCORP.COM',
        'balance': 1500.0
    })
    crud.insert('TechCorp$Customers', {
        'name': 'Diana Prince',
        'email': 'DIANA@TECHCORP.COM',
        'balance': 3000.0
    })
    print("✓ Inserted 2 customers into TechCorp$Customers")

    # 5. Query Company-Specific Data
    print("\n5. Querying Company-Specific Data")
    print("-" * 60)

    acme_customers = crud.get_all('ACME$Customers')
    print(f"\nACME Customers ({acme_customers['count']}):")
    for customer in acme_customers['records']:
        print(f"  - {customer['name']}: {customer['email']} (Balance: ${customer['balance']:.2f})")

    techcorp_customers = crud.get_all('TechCorp$Customers')
    print(f"\nTechCorp Customers ({techcorp_customers['count']}):")
    for customer in techcorp_customers['records']:
        print(f"  - {customer['name']}: {customer['email']} (Balance: ${customer['balance']:.2f})")

    # 6. List Tables
    print("\n6. Table Structure")
    print("-" * 60)

    global_tables = db.list_global_tables()
    print(f"\nGlobal Tables:")
    for table in global_tables:
        print(f"  - {table}")

    acme_tables = db.list_company_tables('ACME')
    print(f"\nACME Company Tables:")
    for table in acme_tables:
        print(f"  - {table}")

    techcorp_tables = db.list_company_tables('TechCorp')
    print(f"\nTechCorp Company Tables:")
    for table in techcorp_tables:
        print(f"  - {table}")

    # 7. Demonstrate Table Name Parsing
    print("\n7. Table Name Parsing")
    print("-" * 60)

    table_names = ['ACME$Customers', 'TechCorp$Customers', 'SystemSettings', 'Company']
    for full_name in table_names:
        company_name, base_name = db.parse_table_name(full_name)
        if company_name:
            print(f"  {full_name} -> Company: '{company_name}', Table: '{base_name}'")
        else:
            print(f"  {full_name} -> Global table: '{base_name}'")

    # 8. Summary
    print("\n8. Architecture Summary")
    print("-" * 60)
    print(f"✓ Companies: {len(companies)}")
    print(f"✓ Global tables: {len(global_tables)}")
    print(f"✓ Company-specific tables: {len(acme_tables) + len(techcorp_tables)}")
    print(f"✓ Data isolation: Each company has separate {acme.name}$Customers and {techcorp.name}$Customers tables")
    print(f"✓ Global data: SystemSettings table accessible by all companies")

    print("\n=== Demo Complete ===")
    db.close()


if __name__ == '__main__':
    main()
