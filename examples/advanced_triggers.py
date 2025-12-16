"""
Advanced trigger examples for OpenERP

Demonstrates complex trigger scenarios:
- Data validation
- Computed fields
- Cross-table operations
- Business logic enforcement
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
        'employees',
        {
            'name': 'TEXT NOT NULL',
            'email': 'TEXT',
            'age': 'INTEGER',
            'salary': 'REAL'
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

# Validate age
age = record.get('age')
if age:
    if age < 18 or age > 70:
        raise ValueError("Age must be between 18 and 70, got " + str(age))

# Validate salary
salary = record.get('salary')
if salary:
    if salary < 0:
        raise ValueError("Salary cannot be negative")

name = record.get('name', 'Unknown')
print("Validated employee: " + name)
"""
    )

    crud = CRUDManager(db)

    # Valid employee
    result = crud.insert('AdvDemo$employees', {
        'name': 'Alice Johnson',
        'email': 'alice@company.com',
        'age': 30,
        'salary': 75000.0
    })
    if result['success']:
        print(f"  ✓ Valid employee inserted: {result['record']['name']}")
    else:
        print(f"  ✗ Insert failed: {result.get('errors', ['Unknown error'])}")

    # Invalid employee (will fail)
    result = crud.insert('AdvDemo$employees', {
        'name': 'Bob Smith',
        'email': 'invalid-email',
        'age': 25,
        'salary': 60000.0
    })
    if not result['success']:
        print(f"  ✗ Validation failed: {result['errors'][0]}")

    # Example 2: Computed Fields
    print("\n2. Computed Fields Trigger")
    db.create_table(
        'invoices',
        {
            'invoice_number': 'TEXT',
            'subtotal': 'REAL NOT NULL',
            'tax_rate': 'REAL DEFAULT 0.10',
            'tax_amount': 'REAL',
            'total': 'REAL',
            'discount_percent': 'REAL DEFAULT 0'
        },
        company_name="AdvDemo",
        on_insert="""
# Calculate tax amount
subtotal = record.get('subtotal', 0)
tax_rate = record.get('tax_rate', 0.10)
record['tax_amount'] = subtotal * tax_rate

# Apply discount
discount_percent = record.get('discount_percent', 0)
discount = subtotal * (discount_percent / 100)

# Calculate total
tax_amount = record.get('tax_amount', 0)
record['total'] = subtotal + tax_amount - discount

# Generate invoice number if not provided
if not record.get('invoice_number'):
    from datetime import datetime
    timestamp = datetime.now().strftime('%Y%m%d%H%M%S')
    record['invoice_number'] = "INV-" + timestamp

invoice_num = record.get('invoice_number', 'N/A')
total = record.get('total', 0)
print("Invoice " + invoice_num + ": $" + str(round(total, 2)))
"""
    )
    crud.reload_triggers()  # Reload to include the newly created invoices table trigger

    result = crud.insert('AdvDemo$invoices', {
        'subtotal': 1000.00,
        'discount_percent': 5
    })
    if result['success']:
        print(f"  ✓ Invoice: {result['record'].get('invoice_number', 'NOT SET')}")
        print(f"    Subtotal: ${result['record']['subtotal']:.2f}")
        print(f"    Tax: ${result['record']['tax_amount']:.2f}")
        print(f"    Total: ${result['record']['total']:.2f}")
    else:
        print(f"  ✗ Invoice creation failed: {result.get('errors', ['Unknown error'])}")

    # Example 3: Status Workflow
    print("\n3. Status Workflow Trigger")
    db.create_table(
        'tasks',
        {
            'title': 'TEXT NOT NULL',
            'status': 'TEXT',
            'assigned_to': 'TEXT',
            'started_at': 'TIMESTAMP',
            'completed_at': 'TIMESTAMP'
        },
        company_name="AdvDemo",
        on_update="""
from datetime import datetime

# Track status changes
old_status = old_record.get('status') if old_record else None
new_status = record.get('status')

if old_status != new_status:
    title = record.get('title', 'Unknown')
    if new_status == 'in_progress' and not record.get('started_at'):
        record['started_at'] = datetime.now().isoformat()
        print("Task '" + title + "' started")

    elif new_status == 'completed':
        record['completed_at'] = datetime.now().isoformat()
        print("Task '" + title + "' completed")
"""
    )
    crud.reload_triggers()  # Reload to include the newly created tasks table trigger

    # Create task
    task = crud.insert('AdvDemo$tasks', {
        'title': 'Implement feature X',
        'status': 'todo',
        'assigned_to': 'developer@company.com'
    })
    print(f"  ✓ Task created: {task['record']['title']}")

    # Start task
    task_id = task['record']['id']
    crud.update('AdvDemo$tasks', task_id, {'status': 'in_progress'})
    updated_task = crud.get_by_id('AdvDemo$tasks', task_id)
    print(f"  ✓ Task started at: {updated_task['record']['started_at']}")

    # Complete task
    crud.update('AdvDemo$tasks', task_id, {'status': 'completed'})
    completed_task = crud.get_by_id('AdvDemo$tasks', task_id)
    print(f"  ✓ Task completed at: {completed_task['record']['completed_at']}")

    # Example 4: Audit Trail
    print("\n4. Audit Trail Trigger")
    db.create_table(
        'audit_log',
        {
            'table_name': 'TEXT',
            'record_id': 'INTEGER',
            'action': 'TEXT',
            'data': 'TEXT',
            'timestamp': 'TIMESTAMP'
        },
        company_name="AdvDemo"
    )

    db.create_table(
        'sensitive_data',
        {
            'data_key': 'TEXT',
            'data_value': 'TEXT'
        },
        company_name="AdvDemo",
        on_insert="""
# Log the insert
data_key = record.get('data_key', 'unknown')
print("Logging insert of sensitive data: " + data_key)
""",
        on_delete="""
from datetime import datetime

# Would log to audit_log table in real implementation
data_key = old_record.get('data_key', 'unknown')
timestamp = str(datetime.now())
print("Audit: Deleted " + data_key + " at " + timestamp)
"""
    )
    crud.reload_triggers()  # Reload to include the newly created sensitive_data table triggers

    sensitive = crud.insert('AdvDemo$sensitive_data', {
        'data_key': 'api_key',
        'data_value': 'secret-key-12345'
    })
    print(f"  ✓ Sensitive data logged on insert")

    sensitive_id = sensitive['record']['id']
    crud.delete('AdvDemo$sensitive_data', sensitive_id)
    print(f"  ✓ Sensitive data deletion logged")

    print("\n=== All trigger examples completed ===")
    db.close()


if __name__ == '__main__':
    main()
