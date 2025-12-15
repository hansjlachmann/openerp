# Phase 1: Foundation - Implementation Plan

## Overview
This document outlines the Phase 1 foundation implementation for OpenERP, a lightweight ERP system built from scratch.

## Timeline
Phase 1: 2-3 months

## Completed Components

### 1. Object Table Storage ✓
- **File**: `openerp/core/database.py`
- **Features**:
  - Dynamic table creation with custom schemas
  - Metadata storage for table definitions
  - Field-level metadata tracking
  - Multi-company support built-in
  - Audit fields (created_at, updated_at)
  - SQLite-based storage with row factory support

**Key Methods**:
- `create_table()` - Create new tables dynamically
- `get_table_metadata()` - Retrieve table information
- `list_tables()` - List all tables (with company filter)
- `drop_table()` - Remove tables and metadata

### 2. Basic Python Code Execution Engine ✓
- **File**: `openerp/core/executor.py`
- **Features**:
  - RestrictedPython-based safe execution
  - Sandboxed environment for user code
  - Access to safe builtins (datetime, math operations)
  - Restricted imports (no os, sys, etc.)
  - Code validation before execution
  - Both exec and eval modes

**Key Methods**:
- `execute()` - Run Python code in sandbox
- `validate_code()` - Validate code without execution

### 3. Trigger System ✓
- **File**: `openerp/core/triggers.py`
- **Features**:
  - ON_INSERT triggers
  - ON_UPDATE triggers
  - ON_DELETE triggers
  - Access to record and old_record in triggers
  - Trigger validation on registration
  - Error handling and reporting

**Key Methods**:
- `register_trigger()` - Register trigger for table
- `execute_trigger()` - Execute trigger with context
- `has_trigger()` - Check trigger existence

### 4. Table CRUD Operations ✓
- **File**: `openerp/core/crud.py`
- **Features**:
  - Insert with ON_INSERT trigger execution
  - Update with ON_UPDATE trigger execution
  - Delete with ON_DELETE trigger execution
  - Get by ID
  - Get all with pagination
  - Search with conditions
  - Company-aware operations
  - Automatic updated_at timestamp management

**Key Methods**:
- `insert()` - Create new records
- `update()` - Update existing records
- `delete()` - Remove records
- `get_by_id()` - Fetch single record
- `get_all()` - List records with pagination
- `search()` - Query records by conditions

### 5. Company Management ✓
- **File**: `openerp/models/company.py`
- **Features**:
  - Multi-company support
  - Company hierarchy (parent-subsidiary)
  - Company-specific data isolation
  - Currency support per company
  - Active/inactive status
  - Tax ID management
  - Automatic code uppercasing

**Key Methods**:
- `Company.create()` - Create new company
- `Company.get_by_id()` - Retrieve by ID
- `Company.get_by_code()` - Retrieve by code
- `Company.list_all()` - List all companies
- `update()` - Update company fields
- `deactivate()` - Deactivate company

## Project Structure

```
openerp/
├── openerp/
│   ├── __init__.py           # Package initialization
│   ├── core/                 # Core ERP functionality
│   │   ├── __init__.py
│   │   ├── database.py       # Object table storage
│   │   ├── executor.py       # Python execution engine
│   │   ├── triggers.py       # Trigger management
│   │   └── crud.py           # CRUD operations
│   ├── models/               # Data models
│   │   ├── __init__.py
│   │   ├── company.py        # Company management
│   │   └── table.py          # Table definitions
│   └── utils/                # Utilities
│       └── __init__.py       # Helper functions
├── tests/                    # Test suite
│   ├── test_database.py      # Database tests
│   ├── test_executor.py      # Executor tests
│   ├── test_crud.py          # CRUD tests
│   └── test_company.py       # Company tests
├── examples/                 # Example scripts
│   ├── basic_usage.py        # Basic usage demo
│   ├── advanced_triggers.py  # Advanced trigger examples
│   └── multi_company.py      # Multi-company demo
├── requirements.txt          # Python dependencies
├── setup.py                  # Package setup
├── pytest.ini                # Test configuration
└── README.md                 # Project documentation
```

## Dependencies

- **SQLAlchemy** (>=2.0.0) - Database ORM
- **Pydantic** (>=2.0.0) - Data validation
- **RestrictedPython** (>=6.0) - Safe code execution
- **python-dateutil** (>=2.8.0) - Date utilities
- **pytz** (>=2023.3) - Timezone support
- **pytest** (>=7.4.0) - Testing framework

## Testing

All core functionality is covered by comprehensive tests:

- **test_database.py**: Table creation, metadata, CRUD
- **test_executor.py**: Code execution, validation, security
- **test_crud.py**: CRUD operations, triggers
- **test_company.py**: Company management, hierarchy

Run tests:
```bash
pytest tests/
```

## Example Usage

### Basic Table with Trigger

```python
from openerp import Database, Company
from openerp.core.crud import CRUDManager

# Initialize
db = Database('openerp.db')
company = Company.create(db, code="ACME", name="Acme Corp")

# Create table with trigger
db.create_table(
    'customers',
    {
        'name': 'TEXT NOT NULL',
        'email': 'TEXT UNIQUE',
        'balance': 'REAL DEFAULT 0'
    },
    on_insert="""
# Auto-lowercase email
record['email'] = record['email'].lower()
print(f"New customer: {record['name']}")
"""
)

# Insert with trigger execution
crud = CRUDManager(db)
result = crud.insert('customers', {
    'name': 'John Doe',
    'email': 'JOHN@EXAMPLE.COM'
})
# Email is now: john@example.com
```

### Multi-Company Data Isolation

```python
# Create companies
us_company = Company.create(db, code="US", name="US Division")
eu_company = Company.create(db, code="EU", name="EU Division")

# Insert company-specific data
crud.insert('sales', {'amount': 1000}, company_id=us_company.id)
crud.insert('sales', {'amount': 2000}, company_id=eu_company.id)

# Query per company
us_sales = crud.get_all('sales', company_id=us_company.id)
eu_sales = crud.get_all('sales', company_id=eu_company.id)
```

## Security Features

1. **RestrictedPython Sandbox**
   - No access to os, sys, subprocess
   - Limited imports
   - Safe builtins only

2. **SQL Injection Prevention**
   - Parameterized queries throughout
   - No dynamic SQL string building

3. **Company Data Isolation**
   - Company ID in all queries
   - No cross-company data leakage

## Next Steps (Phase 2)

1. **Web Interface**
   - REST API with FastAPI
   - Admin dashboard
   - Table designer UI

2. **Advanced Triggers**
   - BEFORE/AFTER triggers
   - Trigger chaining
   - Conditional triggers

3. **User Management**
   - Authentication
   - Authorization
   - Role-based access control

4. **Report Engine**
   - Query builder
   - Export to CSV/Excel/PDF
   - Scheduled reports

## Known Limitations

1. SQLite-based (suitable for small-medium deployments)
2. Single-threaded execution
3. No built-in backup/restore
4. No audit log (yet)
5. Limited to Python-based business logic

## Contributing

See examples in `examples/` directory for usage patterns.

## License

MIT
