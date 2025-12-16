"""
Advanced trigger examples for OpenERP Phase 1

This example demonstrates various trigger scenarios with customers:
1. Data validation (email format)
2. Auto-formatting (phone numbers, names)
3. Default values (timestamps, status)
4. Audit logging (tracking changes)
"""

from openerp import Database, Company
from openerp.core.crud import CRUDManager


def main():
    print("=== Advanced Trigger Examples ===\n")

    db = Database(':memory:')
    company = Company.create(db, "AdvDemo")

    # Example 1: Data Validation Trigger
    print("1. Data Validation Trigger")
    db.create_table(
        'customers',
        {
            'name': 'TEXT NOT NULL',
            'email': 'TEXT',
            'phone': 'TEXT',
            'balance': 'REAL DEFAULT 0'
        },
        company_name="AdvDemo",
        on_insert=r"""
# Validate email format
email = record.get('email')
if email:
    import re
    email_pattern = r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'
    if not re.match(email_pattern, email):
        raise ValueError("Invalid email format: " + email)

# Validate balance
balance = record.get('balance', 0)
if balance < 0:
    raise ValueError("Balance cannot be negative")

name = record.get('name', 'Unknown')
print("Validated customer: " + name)
"""
    )

    crud = CRUDManager(db)

    # Valid customer
    result = crud.insert('AdvDemo$customers', {
        'name': 'Alice Johnson',
        'email': 'alice@company.com',
        'phone': '+1-555-0100',
        'balance': 100.0
    })
    if result['success']:
        print(f"  ✓ Valid customer inserted: {result['record']['name']}")
    else:
        print(f"  ✗ Insert failed: {result.get('errors', ['Unknown error'])}")

    # Invalid customer (will fail)
    result = crud.insert('AdvDemo$customers', {
        'name': 'Bob Smith',
        'email': 'invalid-email',
        'phone': '+1-555-0200'
    })
    if not result['success']:
        print(f"  ✗ Validation failed as expected: {result['errors'][0]}")

    # Example 2: Auto-Formatting Trigger
    print("\n2. Auto-Formatting Trigger")
    db.create_table(
        'contacts',
        {
            'name': 'TEXT NOT NULL',
            'email': 'TEXT',
            'phone': 'TEXT',
            'country': 'TEXT'
        },
        company_name="AdvDemo",
        on_insert="""
# Auto-uppercase name
name = record.get('name', '')
if name:
    record['name'] = name.upper()

# Auto-lowercase email
email = record.get('email', '')
if email:
    record['email'] = email.lower()

# Format phone number (remove spaces and dashes)
phone = record.get('phone', '')
if phone:
    record['phone'] = phone.replace(' ', '').replace('-', '')

formatted_name = record.get('name', 'Unknown')
print("Formatted contact: " + formatted_name)
"""
    )
    crud.reload_triggers()  # Reload to include the newly created contacts table trigger

    result = crud.insert('AdvDemo$contacts', {
        'name': 'john doe',  # Will be uppercased
        'email': 'JOHN@EXAMPLE.COM',  # Will be lowercased
        'phone': '+1-555-0100',  # Will be cleaned
        'country': 'USA'
    })
    if result['success']:
        print(f"  ✓ Contact inserted:")
        print(f"    Name: {result['record']['name']} (auto-uppercased)")
        print(f"    Email: {result['record']['email']} (auto-lowercased)")
        print(f"    Phone: {result['record']['phone']} (formatted)")

    # Example 3: Default Values and Timestamps
    print("\n3. Default Values Trigger")
    db.create_table(
        'leads',
        {
            'name': 'TEXT NOT NULL',
            'email': 'TEXT',
            'status': 'TEXT',
            'priority': 'TEXT',
            'created_date': 'TEXT'
        },
        company_name="AdvDemo",
        on_insert="""
from datetime import datetime

# Set default status if not provided
if not record.get('status'):
    record['status'] = 'new'

# Set default priority
if not record.get('priority'):
    record['priority'] = 'medium'

# Set created date
record['created_date'] = datetime.now().isoformat()

name = record.get('name', 'Unknown')
status = record.get('status', 'unknown')
print("New lead: " + name + " (Status: " + status + ")")
"""
    )
    crud.reload_triggers()  # Reload to include the newly created leads table trigger

    result = crud.insert('AdvDemo$leads', {
        'name': 'Potential Customer',
        'email': 'potential@example.com'
        # status and priority will be set by trigger
    })
    if result['success']:
        print(f"  ✓ Lead created:")
        print(f"    Name: {result['record']['name']}")
        print(f"    Status: {result['record']['status']} (auto-set)")
        print(f"    Priority: {result['record']['priority']} (auto-set)")
        print(f"    Created: {result['record']['created_date'][:19]}")

    # Example 4: Update Trigger (Track Changes)
    print("\n4. Update Tracking Trigger")
    db.create_table(
        'accounts',
        {
            'name': 'TEXT NOT NULL',
            'status': 'TEXT',
            'last_modified': 'TEXT',
            'modification_count': 'INTEGER DEFAULT 0'
        },
        company_name="AdvDemo",
        on_insert="""
from datetime import datetime

# Initialize tracking fields
record['modification_count'] = 0
record['last_modified'] = datetime.now().isoformat()

name = record.get('name', 'Unknown')
print("Account created: " + name)
""",
        on_update="""
from datetime import datetime

# Update modification tracking
old_count = old_record.get('modification_count', 0) if old_record else 0
record['modification_count'] = old_count + 1
record['last_modified'] = datetime.now().isoformat()

name = record.get('name', 'Unknown')
count = record.get('modification_count', 0)
print("Account updated: " + name + " (Modification #" + str(count) + ")")
"""
    )
    crud.reload_triggers()  # Reload to include the newly created accounts table trigger

    # Create account
    result = crud.insert('AdvDemo$accounts', {
        'name': 'Customer Account',
        'status': 'active'
    })
    if result['success']:
        account_id = result['record']['id']
        print(f"  ✓ Account created: {result['record']['name']}")
        print(f"    Modifications: {result['record']['modification_count']}")

        # Update account
        crud.update('AdvDemo$accounts', account_id, {'status': 'pending'})
        crud.update('AdvDemo$accounts', account_id, {'status': 'active'})

        # Check final state
        final = crud.get_by_id('AdvDemo$accounts', account_id)
        if final['success']:
            print(f"  ✓ Final state:")
            print(f"    Modifications: {final['record']['modification_count']}")
            print(f"    Last modified: {final['record']['last_modified'][:19]}")

    print("\n=== All trigger examples completed ===")


if __name__ == "__main__":
    main()
