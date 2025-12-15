# OpenERP - Phase 1 Foundation

A lightweight, flexible ERP system built from scratch with Python.

## Phase 1 Features

- **Object Table Storage**: Dynamic table creation and management
- **Python Code Execution Engine**: Safe execution of custom business logic
- **Company Management**: Multi-company support
- **Table CRUD Operations**: Full CRUD with trigger support (OnInsert, OnUpdate, OnDelete)

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

## Quick Start

```python
from openerp.core.database import Database
from openerp.models.company import Company

# Initialize database
db = Database('openerp.db')

# Create a company
company = Company.create(db, name="Acme Corp", code="ACME")

# Create a custom table with OnInsert trigger
table = db.create_table('customers', {
    'name': 'TEXT',
    'email': 'TEXT',
    'created_at': 'TIMESTAMP'
}, on_insert="""
record['created_at'] = datetime.now()
print(f"New customer: {record['name']}")
""")

# Insert data (trigger will execute)
db.insert('customers', {'name': 'John Doe', 'email': 'john@example.com'})
```

## Development Status

Phase 1 - Foundation (In Progress)
- [x] Project structure
- [ ] Object table storage
- [ ] Python execution engine
- [ ] Company management
- [ ] CRUD with triggers

## License

MIT
