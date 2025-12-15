# OpenERP - Phase 1 Foundation

A lightweight, flexible ERP system built from scratch with Python.

## Phase 1 Features

- **Object Table Storage**: Dynamic table creation and management
- **Python Code Execution Engine**: Safe execution of custom business logic
- **Company Management**: Multi-company support with physical table separation
- **Table CRUD Operations**: Full CRUD with trigger support (OnInsert, OnUpdate, OnDelete)
- **Multi-Language Support (i18n)**: Complete translation system for UI elements, table/field names, messages, and more

## Architecture

```
openerp/
├── core/           # Core ERP functionality
│   ├── database.py    # Object table storage engine
│   ├── executor.py    # Python code execution engine
│   ├── triggers.py    # Trigger system
│   ├── crud.py        # CRUD operations
│   └── i18n.py        # Multi-language support
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

OpenERP includes built-in support for multi-language applications:

```python
from openerp import Database, Language, init_i18n, t

db = Database('openerp.db')

# Initialize translation system
tm = init_i18n(db, default_language="en")

# Create languages
Language.create(db, "en", "English", "English", is_default=True)
Language.create(db, "es", "Spanish", "Español")
Language.create(db, "nl", "Dutch", "Nederlands")

# Add translations
tm.add_translation("es", "table_name", "Customers", "Clientes")
tm.add_translation("nl", "table_name", "Customers", "Klanten")

# Use translations
tm.set_language("es")
print(t("Customers", "table_name"))  # "Clientes"
```

**Translatable Elements:**
- Table names, Field names, Form labels
- Buttons, Messages, Dialogs
- Reports, Menus, Validation messages

See [MULTI_LANGUAGE.md](MULTI_LANGUAGE.md) for complete documentation.

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
- [x] Multi-language support (i18n) for all UI elements
- [x] Comprehensive test suite
- [x] Example scripts and documentation

## License

MIT
