# OpenERP Company Architecture

## Overview

OpenERP uses a **physical table separation** approach for multi-company data isolation. This document explains the architecture and how to work with company-specific and global tables.

## Table Types

### 1. Global Tables

Global tables are accessible to all companies and use standard table names.

**Naming**: `TableName` (e.g., `Company`, `SystemSettings`)

**Characteristics**:
- Shared across all companies
- No company prefix
- Used for system-wide configuration and master data

**Example**:
```python
# Create global table
db.create_table(
    'SystemSettings',
    {'key': 'TEXT', 'value': 'TEXT'},
    is_global=True
)

# Insert into global table
crud.insert('SystemSettings', {'key': 'theme', 'value': 'dark'})
```

### 2. Company-Specific Tables (Local Tables)

Company-specific tables belong to a single company and use the `CompanyName$TableName` naming convention.

**Naming**: `CompanyName$TableName` (e.g., `ACME$Customers`, `TechCorp$Orders`)

**Characteristics**:
- Data is physically isolated per company
- Each company can have its own version of the same table
- Company name is part of the actual table name in the database

**Example**:
```python
# Create company-specific table
db.create_table(
    'Customers',
    {'name': 'TEXT', 'email': 'TEXT'},
    company_name='ACME'
)
# Creates table: ACME$Customers

# Insert into company-specific table
crud.insert('ACME$Customers', {'name': 'John Doe', 'email': 'john@acme.com'})
```

## Company Table Structure

The `Company` table is a **global table** with a simple structure:

```sql
CREATE TABLE Company (
    Name TEXT PRIMARY KEY NOT NULL
)
```

- **Name**: Company name (PRIMARY KEY)
- This name is used in the `CompanyName$TableName` format

## Working with Companies

### Creating a Company

```python
from openerp import Database, Company

db = Database('myerp.db')

# Create a company
acme = Company.create(db, "ACME")
# Company name must be alphanumeric, underscore, or hyphen only
```

### Retrieving Companies

```python
# Get by name
company = Company.get_by_name(db, "ACME")

# Check if exists
if Company.exists(db, "ACME"):
    print("Company exists")

# List all companies
companies = Company.list_all(db)
for company in companies:
    print(company.name)
```

## Creating Tables

### Global Table

```python
db.create_table(
    'SystemSettings',
    {
        'key': 'TEXT NOT NULL UNIQUE',
        'value': 'TEXT',
        'description': 'TEXT'
    },
    is_global=True  # Mark as global
)
```

### Company-Specific Table

```python
# Create for ACME company
db.create_table(
    'Customers',  # Base table name
    {
        'name': 'TEXT NOT NULL',
        'email': 'TEXT',
        'phone': 'TEXT'
    },
    company_name='ACME',  # Company name
    on_insert="""
# Trigger code
record['email'] = record['email'].lower()
"""
)

# This creates a table named: ACME$Customers
```

### Creating Same Table for Multiple Companies

```python
# Each company gets its own physical table
for company_name in ['ACME', 'TechCorp', 'GlobalCo']:
    db.create_table(
        'Orders',
        {'order_number': 'TEXT', 'amount': 'REAL'},
        company_name=company_name
    )

# Creates: ACME$Orders, TechCorp$Orders, GlobalCo$Orders
```

## CRUD Operations

All CRUD operations use the **full table name** (including company prefix for company-specific tables).

### Insert

```python
from openerp.core.crud import CRUDManager

crud = CRUDManager(db)

# Global table
crud.insert('SystemSettings', {'key': 'version', 'value': '1.0'})

# Company-specific table (use full name with $)
crud.insert('ACME$Customers', {'name': 'John Doe', 'email': 'john@acme.com'})
crud.insert('TechCorp$Customers', {'name': 'Jane Smith', 'email': 'jane@techcorp.com'})
```

### Read

```python
# Get by ID
customer = crud.get_by_id('ACME$Customers', 1)

# Get all
all_customers = crud.get_all('ACME$Customers')

# Search
results = crud.search('ACME$Customers', {'name': 'John Doe'})
```

### Update

```python
crud.update('ACME$Customers', 1, {'email': 'newemail@acme.com'})
```

### Delete

```python
crud.delete('ACME$Customers', 1)
```

## Listing Tables

### List All Tables

```python
# All tables
all_tables = db.list_tables()
# Returns: ['Company', 'SystemSettings', 'ACME$Customers', 'TechCorp$Customers', ...]

# Company-specific tables only
acme_tables = db.list_tables('ACME', include_global=False)
# Returns: ['ACME$Customers', 'ACME$Orders', ...]

# Company tables + global tables
acme_all = db.list_tables('ACME', include_global=True)
# Returns: ['Company', 'SystemSettings', 'ACME$Customers', ...]
```

### List Global Tables

```python
global_tables = db.list_global_tables()
# Returns: ['Company', 'SystemSettings']
```

### List Company-Specific Tables

```python
# Full names
acme_tables = db.list_company_tables('ACME')
# Returns: ['ACME$Customers', 'ACME$Orders']

# Base names only
acme_base = db.list_company_tables('ACME', base_names_only=True)
# Returns: ['Customers', 'Orders']
```

## Table Name Utilities

### Get Full Table Name

```python
# Build full table name
full_name = Database.get_full_table_name('Customers', 'ACME')
# Returns: "ACME$Customers"

# Global table (no company)
full_name = Database.get_full_table_name('SystemSettings', None)
# Returns: "SystemSettings"
```

### Parse Table Name

```python
# Parse company-specific table
company, table = Database.parse_table_name('ACME$Customers')
# Returns: ("ACME", "Customers")

# Parse global table
company, table = Database.parse_table_name('SystemSettings')
# Returns: (None, "SystemSettings")
```

### Check if Global

```python
is_global = db.is_global_table('SystemSettings')  # True
is_global = db.is_global_table('ACME$Customers')  # False
```

## Complete Example

```python
from openerp import Database, Company
from openerp.core.crud import CRUDManager

# Initialize
db = Database('myerp.db')
crud = CRUDManager(db)

# Create companies
acme = Company.create(db, "ACME")
techcorp = Company.create(db, "TechCorp")

# Create global table
db.create_table('Currency', {'code': 'TEXT', 'name': 'TEXT'}, is_global=True)
crud.insert('Currency', {'code': 'USD', 'name': 'US Dollar'})

# Create company-specific tables
for company in [acme, techcorp]:
    db.create_table(
        'Invoices',
        {'invoice_no': 'TEXT', 'amount': 'REAL', 'currency_code': 'TEXT'},
        company_name=company.name
    )

# Insert company-specific data
crud.insert('ACME$Invoices', {
    'invoice_no': 'INV-001',
    'amount': 1000.0,
    'currency_code': 'USD'
})

crud.insert('TechCorp$Invoices', {
    'invoice_no': 'INV-001',  # Same invoice number, different company
    'amount': 2000.0,
    'currency_code': 'USD'
})

# Query
acme_invoices = crud.get_all('ACME$Invoices')
techcorp_invoices = crud.get_all('TechCorp$Invoices')

print(f"ACME has {acme_invoices['count']} invoices")
print(f"TechCorp has {techcorp_invoices['count']} invoices")
```

## Benefits of This Architecture

1. **Physical Data Isolation**: Company data is completely separated at the database level
2. **No Cross-Company Data Leaks**: Impossible to accidentally query another company's data
3. **Independent Schemas**: Each company can have different table structures if needed
4. **Clear Naming**: Table names explicitly show which company they belong to
5. **Performance**: No need for company_id filtering in WHERE clauses
6. **Scalability**: Easy to move a company's tables to a different database

## Migration from Old Architecture

If you have code using the old `company_id` parameter:

**Old**:
```python
crud.insert('customers', {'name': 'John'}, company_id=1)
crud.get_all('customers', company_id=1)
```

**New**:
```python
crud.insert('ACME$Customers', {'name': 'John'})
crud.get_all('ACME$Customers')
```

## Best Practices

1. **Always use full table names** in CRUD operations
2. **Create companies before creating company-specific tables**
3. **Use helper methods** like `get_full_table_name()` when building table names dynamically
4. **Use global tables** for shared master data (currencies, countries, etc.)
5. **Use company-specific tables** for transactional data (invoices, orders, etc.)
6. **Validate company names** - only alphanumeric, underscore, and hyphen allowed

## Forbidden Characters in Company Names

Company names **cannot contain**:
- Dollar sign `$` (reserved for table name separator)
- Spaces
- Special characters (except `-` and `_`)

**Valid**: `ACME`, `Acme-Corp`, `Company_123`
**Invalid**: `ACME Corp`, `ACME$Corp`, `ACME@Corp`
