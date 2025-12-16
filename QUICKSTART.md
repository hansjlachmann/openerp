# Quick Start Guide - Running OpenERP

This guide will help you get OpenERP running on your Ubuntu Linux workstation.

## Prerequisites

- Python 3.8 or higher
- pip (Python package manager)
- Git

## Setup Instructions

### 1. Navigate to the Project Directory

```bash
cd ~/Workspace/Python/openerp
# Or wherever you cloned the repository
```

### 2. Install python3-venv (Ubuntu/Debian Only)

On Ubuntu/Debian systems, you need to install the venv package first:

```bash
sudo apt install python3.12-venv
# Or for other Python versions: sudo apt install python3-venv
```

### 3. Create a Virtual Environment (Recommended)

```bash
# Create virtual environment
python3 -m venv venv

# Activate virtual environment
source venv/bin/activate
```

You should see `(venv)` in your terminal prompt after activation.

### 4. Install Dependencies

**Important for Ubuntu/Debian:** Use `python -m pip` instead of just `pip`:

```bash
# Upgrade pip first
python -m pip install --upgrade pip

# Install project dependencies
python -m pip install -r requirements.txt

# Install OpenERP in editable mode
python -m pip install -e .
```

This will install:
- `RestrictedPython` - For safe code execution
- `pytz` - For timezone support
- `SQLAlchemy` - Database toolkit
- `pydantic` - Data validation
- `openerp` - The OpenERP package itself (editable mode)

### 5. Verify Installation

```bash
python verify_setup.py
```

You should see all green checkmarks ✓ indicating everything is working correctly.

## Running Examples

### Basic Usage Example

```bash
python3 examples/basic_usage.py
```

This demonstrates:
- Database initialization
- Table creation
- CRUD operations
- Basic triggers

### Multi-Company Architecture Example

```bash
python3 examples/new_company_architecture.py
```

This demonstrates:
- Creating multiple companies
- Global vs company-specific tables
- CompanyName$TableName architecture
- Querying company-specific data

### Translation System Example

```bash
python3 examples/translation_demo.py
```

This demonstrates:
- Setting up multi-language translations
- Table name translations (EN, ES, NL, FR, DE)
- Field name translations
- Building multi-language forms
- Fallback behavior

### Advanced Triggers Example

```bash
python3 examples/advanced_triggers.py
```

This demonstrates:
- OnInsert triggers
- OnUpdate triggers
- OnDelete triggers
- Chaining triggers
- Error handling in triggers

### Multi-Company Example

```bash
python3 examples/multi_company.py
```

This demonstrates:
- Creating multiple companies
- Company-specific data isolation
- Cross-company scenarios

## Interactive Python Session

You can also use OpenERP interactively:

```bash
python3
```

Then in the Python shell:

```python
from openerp import Database, Company
from openerp.core.crud import CRUDManager

# Create database
db = Database('test.db')
crud = CRUDManager(db)

# Create a company
company = Company.create(db, "MyCompany")
print(f"Created company: {company.name}")

# Create a company-specific table
db.create_table('Employees', {
    'name': 'TEXT NOT NULL',
    'email': 'TEXT NOT NULL',
    'department': 'TEXT'
}, company_name='MyCompany')

# Add translations
db.set_table_translation('MyCompany$Employees', 'es', 'Empleados')
db.set_field_translation('MyCompany$Employees', 'name', 'es', 'Nombre')
db.set_field_translation('MyCompany$Employees', 'email', 'es', 'Correo')
db.set_field_translation('MyCompany$Employees', 'department', 'es', 'Departamento')

# Insert data
crud.insert('MyCompany$Employees', {
    'name': 'John Doe',
    'email': 'john@example.com',
    'department': 'Engineering'
})

# Query data
result = crud.get_all('MyCompany$Employees')
for emp in result['records']:
    print(f"Employee: {emp['name']} - {emp['email']}")

# Get translations
print(f"Table in Spanish: {db.get_table_translation('MyCompany$Employees', 'es')}")
print(f"Field 'name' in Spanish: {db.get_field_translation('MyCompany$Employees', 'name', 'es')}")
```

## Creating Your Own Application

### Step 1: Create a new Python file

```bash
touch my_app.py
```

### Step 2: Add your code

```python
#!/usr/bin/env python3
"""
My OpenERP Application
"""

from openerp import Database, Company
from openerp.core.crud import CRUDManager

def main():
    # Initialize
    db = Database('myapp.db')
    crud = CRUDManager(db)

    # Create company
    if not Company.exists(db, "ACME"):
        company = Company.create(db, "ACME")
        print(f"✓ Created company: {company.name}")
    else:
        print("✓ Company ACME already exists")

    # Create your tables
    # ... your code here ...

    print("✓ Application initialized successfully!")

if __name__ == "__main__":
    main()
```

### Step 3: Run your application

```bash
python3 my_app.py
```

## Common Commands

### Clean up database files

```bash
rm -f *.db
```

### Run all examples

```bash
for example in examples/*.py; do
    echo "Running $example..."
    python3 "$example"
    echo ""
done
```

### Check Python version

```bash
python3 --version
```

### List installed packages

```bash
pip list
```

### Deactivate virtual environment

```bash
deactivate
```

## Troubleshooting

### Issue: "error: externally-managed-environment" (Ubuntu/Debian)

**Problem:** Getting an error about externally-managed-environment when trying to install packages.

**Solution:** Use `python -m pip` instead of just `pip`:

```bash
# Instead of: pip install -r requirements.txt
python -m pip install -r requirements.txt

# Instead of: pip install -e .
python -m pip install -e .
```

This explicitly uses your virtual environment's Python interpreter.

### Issue: "ModuleNotFoundError: No module named 'openerp'"

**Solution 1:** Install the package in editable mode:

```bash
python -m pip install -e .
```

**Solution 2:** Make sure you're in the project root directory and have installed dependencies:

```bash
cd ~/Workspace/Python/openerp
python -m pip install -r requirements.txt
python -m pip install -e .
```

### Issue: "ModuleNotFoundError: No module named 'RestrictedPython'"

**Solution:** Install dependencies:

```bash
python -m pip install -r requirements.txt
```

### Issue: "ensurepip is not available" when creating venv

**Problem:** Cannot create virtual environment on Ubuntu/Debian.

**Solution:** Install python3-venv package:

```bash
sudo apt install python3.12-venv
# Or for other Python versions: sudo apt install python3-venv
```

Then create the virtual environment again:

```bash
python3 -m venv venv
source venv/bin/activate
```

### Issue: "Permission denied" when creating database files

**Solution:** Ensure you have write permissions in the current directory:

```bash
chmod 755 .
```

### Issue: Virtual environment not activating

**Solution:** Make sure you created it correctly:

```bash
python3 -m venv venv
source venv/bin/activate
```

### Issue: Import errors

**Solution:** Verify your Python path includes the project:

```bash
python3 -c "import sys; print(sys.path)"
```

## File Structure

```
openerp/
├── examples/               # Example scripts
│   ├── basic_usage.py
│   ├── new_company_architecture.py
│   ├── translation_demo.py
│   ├── advanced_triggers.py
│   └── multi_company.py
├── openerp/               # Main package
│   ├── __init__.py
│   ├── core/              # Core functionality
│   │   ├── database.py    # Database engine
│   │   ├── crud.py        # CRUD operations
│   │   ├── executor.py    # Code execution
│   │   └── triggers.py    # Trigger system
│   └── models/            # Data models
│       └── company.py     # Company management
├── tests/                 # Test suite
├── requirements.txt       # Dependencies
├── setup.py              # Package setup
├── README.md             # Main documentation
├── COMPANY_ARCHITECTURE.md  # Company architecture docs
├── TRANSLATIONS.md       # Translation system docs
└── QUICKSTART.md        # This file
```

## Next Steps

1. **Run the examples** to see OpenERP in action
2. **Read the documentation**:
   - [README.md](README.md) - Overview and basic usage
   - [COMPANY_ARCHITECTURE.md](COMPANY_ARCHITECTURE.md) - Multi-company design
   - [TRANSLATIONS.md](TRANSLATIONS.md) - Translation system
3. **Create your own application** using the interactive session or create a new Python file
4. **Explore the code** in `openerp/core/` to understand the internals

## Getting Help

- Check the documentation in the root directory
- Look at example scripts in `examples/`
- Read the docstrings in the source code
- Check the test suite in `tests/`

## Development Mode

If you want to modify OpenERP itself:

```bash
# Install in development mode
pip install -e .

# Run tests
python3 -m pytest tests/

# Or run tests with coverage
python3 -m pytest tests/ --cov=openerp
```

## Database Files

OpenERP creates SQLite database files with `.db` extension. These files contain:
- All tables (global and company-specific)
- Metadata tables (`__table_metadata`, `__field_metadata`)
- All data and translations

You can inspect these files using:

```bash
# Install sqlite3 if not already installed
sudo apt-get install sqlite3

# Open a database
sqlite3 myapp.db

# Inside sqlite3:
.tables              # List all tables
.schema Company      # Show table schema
SELECT * FROM Company;  # Query data
.quit                # Exit
```

## Performance Tips

1. **Use transactions** for bulk operations
2. **Index frequently queried columns** using raw SQL
3. **Cache translations** in memory for high-traffic applications
4. **Use global tables** for shared data across companies
5. **Clean up old database files** you don't need

## Security Notes

- RestrictedPython provides sandboxed code execution
- SQL injection is prevented by parameterized queries
- Company data is physically isolated in separate tables
- Always validate user input before inserting into triggers

Enjoy using OpenERP! 🚀
