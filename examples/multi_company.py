"""
Multi-company example for OpenERP

Demonstrates:
- Creating multiple companies
- Company-specific data isolation
- Parent-subsidiary relationships
"""

from openerp import Database, Company
from openerp.core.crud import CRUDManager


def main():
    print("=== Multi-Company Example ===\n")

    db = Database(':memory:')
    crud = CRUDManager(db)

    # Create parent company
    print("1. Creating Parent Company")
    parent = Company.create(
        db,
        code="ACME",
        name="Acme Holdings",
        legal_name="Acme Holdings Inc.",
        tax_id="12-3456789",
        currency="USD"
    )
    print(f"  ✓ {parent.name} ({parent.code})")

    # Create subsidiaries
    print("\n2. Creating Subsidiary Companies")
    subsidiary_us = Company.create(
        db,
        code="ACME-US",
        name="Acme USA",
        legal_name="Acme USA Inc.",
        tax_id="12-1111111",
        currency="USD",
        parent_id=parent.id
    )
    print(f"  ✓ {subsidiary_us.name} ({subsidiary_us.code})")

    subsidiary_eu = Company.create(
        db,
        code="ACME-EU",
        name="Acme Europe",
        legal_name="Acme Europe GmbH",
        tax_id="EU-2222222",
        currency="EUR",
        parent_id=parent.id
    )
    print(f"  ✓ {subsidiary_eu.name} ({subsidiary_eu.code})")

    subsidiary_asia = Company.create(
        db,
        code="ACME-ASIA",
        name="Acme Asia",
        legal_name="Acme Asia Ltd.",
        tax_id="AS-3333333",
        currency="JPY",
        parent_id=parent.id
    )
    print(f"  ✓ {subsidiary_asia.name} ({subsidiary_asia.code})")

    # Create a shared table structure
    print("\n3. Creating Company-Specific Sales Data")
    db.create_table(
        'sales',
        {
            'product_name': 'TEXT NOT NULL',
            'quantity': 'INTEGER',
            'unit_price': 'REAL',
            'total': 'REAL',
            'currency': 'TEXT'
        },
        on_insert="""
# Calculate total
record['total'] = record['quantity'] * record['unit_price']
"""
    )

    # Add sales for each company
    print("\n4. Adding Sales Records per Company")

    # US sales
    crud.insert('sales', {
        'product_name': 'Widget Pro',
        'quantity': 100,
        'unit_price': 50.0,
        'currency': 'USD'
    }, company_id=subsidiary_us.id)

    crud.insert('sales', {
        'product_name': 'Gadget Elite',
        'quantity': 75,
        'unit_price': 125.0,
        'currency': 'USD'
    }, company_id=subsidiary_us.id)
    print(f"  ✓ Added sales for {subsidiary_us.name}")

    # EU sales
    crud.insert('sales', {
        'product_name': 'Widget Pro',
        'quantity': 80,
        'unit_price': 45.0,
        'currency': 'EUR'
    }, company_id=subsidiary_eu.id)

    crud.insert('sales', {
        'product_name': 'Gadget Elite',
        'quantity': 60,
        'unit_price': 110.0,
        'currency': 'EUR'
    }, company_id=subsidiary_eu.id)
    print(f"  ✓ Added sales for {subsidiary_eu.name}")

    # Asia sales
    crud.insert('sales', {
        'product_name': 'Widget Pro',
        'quantity': 150,
        'unit_price': 5500.0,
        'currency': 'JPY'
    }, company_id=subsidiary_asia.id)

    crud.insert('sales', {
        'product_name': 'Gadget Elite',
        'quantity': 120,
        'unit_price': 13500.0,
        'currency': 'JPY'
    }, company_id=subsidiary_asia.id)
    print(f"  ✓ Added sales for {subsidiary_asia.name}")

    # Query data per company
    print("\n5. Company-Specific Sales Reports")
    print("-" * 60)

    companies = [subsidiary_us, subsidiary_eu, subsidiary_asia]
    for company in companies:
        sales = crud.get_all('sales', company_id=company.id)

        print(f"\n{company.name} ({company.currency}):")
        total_revenue = 0
        for sale in sales['records']:
            revenue = sale['total']
            total_revenue += revenue
            print(f"  - {sale['product_name']}: {sale['quantity']} units × "
                  f"{sale['unit_price']} = {revenue:,.2f} {sale['currency']}")

        print(f"  Total Revenue: {total_revenue:,.2f} {company.currency}")

    # Company hierarchy
    print("\n6. Company Hierarchy")
    print("-" * 60)
    all_companies = Company.list_all(db)

    print(f"{parent.name} (Parent)")
    for company in all_companies:
        if company.parent_id == parent.id:
            print(f"  └─ {company.name} ({company.code})")

    # Summary
    print("\n7. System Summary")
    print("-" * 60)
    print(f"Total Companies: {len(all_companies)}")
    print(f"Parent Companies: 1")
    print(f"Subsidiaries: {len([c for c in all_companies if c.parent_id])}")

    # Query all sales (cross-company)
    all_sales = crud.get_all('sales')
    print(f"Total Sales Records: {all_sales['count']}")

    print("\n=== Multi-Company Demo Complete ===")
    db.close()


if __name__ == '__main__':
    main()
