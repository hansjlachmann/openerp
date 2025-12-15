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
    company = Company.create(db, code="ADV", name="Advanced Demo")

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
        on_insert="""
# Validate email format
if record.get('email'):
    import re
    email_pattern = r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'
    if not re.match(email_pattern, record['email']):
        raise ValueError(f"Invalid email format: {record['email']}")

# Validate age
if record.get('age'):
    if record['age'] < 18 or record['age'] > 70:
        raise ValueError(f"Age must be between 18 and 70, got {record['age']}")

# Validate salary
if record.get('salary'):
    if record['salary'] < 0:
        raise ValueError("Salary cannot be negative")

print(f"Validated employee: {record['name']}")
"""
    )

    crud = CRUDManager(db)

    # Valid employee
    result = crud.insert('employees', {
        'name': 'Alice Johnson',
        'email': 'alice@company.com',
        'age': 30,
        'salary': 75000.0
    })
    print(f"  ✓ Valid employee inserted: {result['record']['name']}")

    # Invalid employee (will fail)
    result = crud.insert('employees', {
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
        on_insert="""
# Calculate tax amount
record['tax_amount'] = record['subtotal'] * record.get('tax_rate', 0.10)

# Apply discount
discount = record['subtotal'] * (record.get('discount_percent', 0) / 100)

# Calculate total
record['total'] = record['subtotal'] + record['tax_amount'] - discount

# Generate invoice number if not provided
if not record.get('invoice_number'):
    from datetime import datetime
    timestamp = datetime.now().strftime('%Y%m%d%H%M%S')
    record['invoice_number'] = f"INV-{timestamp}"

print(f"Invoice {record['invoice_number']}: ${record['total']:.2f}")
"""
    )

    result = crud.insert('invoices', {
        'subtotal': 1000.00,
        'discount_percent': 5
    })
    print(f"  ✓ Invoice: {result['record']['invoice_number']}")
    print(f"    Subtotal: ${result['record']['subtotal']:.2f}")
    print(f"    Tax: ${result['record']['tax_amount']:.2f}")
    print(f"    Total: ${result['record']['total']:.2f}")

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
        on_update="""
from datetime import datetime

# Track status changes
old_status = old_record.get('status') if old_record else None
new_status = record.get('status')

if old_status != new_status:
    if new_status == 'in_progress' and not record.get('started_at'):
        record['started_at'] = datetime.now().isoformat()
        print(f"Task '{record['title']}' started")

    elif new_status == 'completed':
        record['completed_at'] = datetime.now().isoformat()
        print(f"Task '{record['title']}' completed")
"""
    )

    # Create task
    task = crud.insert('tasks', {
        'title': 'Implement feature X',
        'status': 'todo',
        'assigned_to': 'developer@company.com'
    })
    print(f"  ✓ Task created: {task['record']['title']}")

    # Start task
    crud.update('tasks', task['id'], {'status': 'in_progress'})
    updated_task = crud.get_by_id('tasks', task['id'])
    print(f"  ✓ Task started at: {updated_task['record']['started_at']}")

    # Complete task
    crud.update('tasks', task['id'], {'status': 'completed'})
    completed_task = crud.get_by_id('tasks', task['id'])
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
        }
    )

    db.create_table(
        'sensitive_data',
        {
            'data_key': 'TEXT',
            'data_value': 'TEXT'
        },
        on_insert="""
# Log the insert
print(f"Logging insert of sensitive data: {record['data_key']}")
""",
        on_delete="""
from datetime import datetime

# Would log to audit_log table in real implementation
print(f"Audit: Deleted {old_record['data_key']} at {datetime.now()}")
"""
    )

    sensitive = crud.insert('sensitive_data', {
        'data_key': 'api_key',
        'data_value': 'secret-key-12345'
    })
    print(f"  ✓ Sensitive data logged on insert")

    crud.delete('sensitive_data', sensitive['id'])
    print(f"  ✓ Sensitive data deletion logged")

    print("\n=== All trigger examples completed ===")
    db.close()


if __name__ == '__main__':
    main()
