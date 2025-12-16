# OpenERP - Phase 1 Foundation

A lightweight, flexible ERP system built from scratch with Python.

## Phase 1 Features

- **Object Table Storage**: Dynamic table creation and management
- **Python Code Execution Engine**: Safe execution of custom business logic
- **Company Management**: Multi-company support with physical table separation
- **Table CRUD Operations**: Full CRUD with trigger support (OnInsert, OnUpdate, OnDelete)
- **Multi-Language Support**: Translation system for table and field names

## Architecture

```
openerp/
├── core/           # Core ERP functionality
│   ├── database.py    # Object table storage engine
│   ├── executor.py    # Python code execution engine
│   ├── triggers.py    # Trigger system
│   └── crud.py        # CRUD operations
├── models/         # Data models
│   ├── company.py     # Company management
│   └── table.py       # Table definitions
└── utils/          # Utility functions
```

## Installation

```bash
pip install -r requirements.txt
```

## Company Architecture

OpenERP uses **physical table separation** for multi-company support:

- **Global Tables**: `TableName` (e.g., `Company`, `SystemSettings`)
  - Accessible to all companies
  - Used for system-wide configuration

- **Company-Specific Tables**: `CompanyName$TableName` (e.g., `ACME$Customers`)
  - Each company has physically separate tables
  - Complete data isolation at the database level

See [COMPANY_ARCHITECTURE.md](COMPANY_ARCHITECTURE.md) for detailed documentation.

## Multi-Language Support

OpenERP includes built-in multi-language translation support for table and field names:

- **Translation Storage**: Translations stored as JSON in metadata tables
- **Multiple Languages**: Support for unlimited language codes (ISO 639-1)
- **Fallback Support**: Automatic fallback to original names when translations missing
- **Easy API**: Simple methods to set and retrieve translations

```python
# Set translations for a table
db.set_table_translation("ACME$Customers", "es", "Clientes")
db.set_table_translation("ACME$Customers", "fr", "Clients")

# Set translations for a field
db.set_field_translation("ACME$Customers", "Email", "es", "Correo electrónico")
db.set_field_translation("ACME$Customers", "Email", "fr", "Courriel")

# Retrieve translations
table_name_es = db.get_table_translation("ACME$Customers", "es")
# Returns: "Clientes"
```

See [TRANSLATIONS.md](TRANSLATIONS.md) for detailed documentation.

## Quick Start

```python
from openerp import Database, Company
from openerp.core.crud import CRUDManager

# Initialize database
db = Database('openerp.db')
crud = CRUDManager(db)

# Create a company
company = Company.create(db, "ACME")  # Name is PRIMARY KEY

# Create a company-specific table with OnInsert trigger
db.create_table(
    'Customers',  # Base table name
    {
        'name': 'TEXT',
        'email': 'TEXT'
    },
    company_name='ACME',  # Creates: ACME$Customers
    on_insert="""
from datetime import datetime
record['email'] = record['email'].lower()
print(f"New customer: {record['name']}")
"""
)

# Insert data into company-specific table (trigger will execute)
crud.insert('ACME$Customers', {
    'name': 'John Doe',
    'email': 'JOHN@EXAMPLE.COM'  # Will be lowercased by trigger
})

# Create a global table
db.create_table(
    'SystemSettings',
    {'key': 'TEXT', 'value': 'TEXT'},
    is_global=True
)
```

## Development Status

Phase 1 - Foundation (Complete)
- [x] Project structure
- [x] Object table storage with CompanyName$TableName architecture
- [x] Python execution engine (RestrictedPython-based)
- [x] Company management (Name as PRIMARY KEY)
- [x] CRUD with triggers (OnInsert, OnUpdate, OnDelete)
- [x] Global vs company-specific table support
- [x] Multi-language translation support (table and field names)
- [x] Comprehensive test suite
- [x] Example scripts and documentation

## License

MIT
