# OpenERP Go - Foundation Layer

This is the foundation layer for OpenERP in Go, implementing 8 core database and company management functions.

## Quick Start

```bash
cd /home/user/openerp/go-poc/cmd/interactive

# Run the interactive menu
go run .

# Or build and run
go build -o openerp-foundation .
./openerp-foundation
```

## What This Is

The foundation layer provides the base functionality for OpenERP:

**8 Core Functions:**
1. CreateDatabase - Create new persistent database
2. OpenDatabase - Open existing database
3. CloseDatabase - Close database connection
4. CreateCompany - Create new company
5. EnterCompany - Enter company session (NAV-style)
6. ExitCompany - Exit company session
7. DeleteCompany - Delete company and all its tables
8. ListCompanies - List all companies

## Architecture

- **Per-Connection State**: Each database connection maintains its own company session
- **Thread-Safe**: Multiple users can work concurrently with isolated sessions
- **NAV-Style**: Session-based company context (EnterCompany/ExitCompany)
- **Multi-Company**: Physical table isolation using `Company$TableName` pattern

## Project Structure

```
go-poc/
├── cmd/
│   └── interactive/           # Foundation Layer Application
│       ├── foundation.go      # Core foundation library
│       ├── main_foundation.go # Interactive menu
│       ├── go.mod
│       └── go.sum
├── README.md                  # This file
└── FOUNDATION_README.md       # Detailed documentation
```

## Features

✅ **Implemented:**
- Database connection (SQLite)
- Multi-company support with physical table separation
- Session-based company context
- NAV-style API patterns
- Comprehensive error handling
- Per-connection state for thread safety

🚧 **Next Phase:**
- CRUD operations (Insert, Get, Update, Delete, FindSet)
- Python trigger execution (embedded)
- Metadata management
- HTTP REST API

## Usage Example

```
=== OpenERP Foundation Layer - Interactive Menu ===

[No database open]

============================================================
MAIN MENU
============================================================
1. Create new database
2. Open existing database
3. Close database
4. Create company
5. Enter company
6. Exit company
7. Delete company
8. List companies
9. Exit application
============================================================

Select option (1-9): 1
Enter database path: myerp.db
✓ Database created successfully: myerp.db

[Database: myerp.db]

Select option (1-9): 4
Enter company name: ACME
✓ Company 'ACME' created successfully

Select option (1-9): 5
Available companies:
  1. ACME
Enter company name: ACME
✓ Entered company 'ACME'

[Database: myerp.db | Company: ACME]
```

## Requirements

- Go 1.21+
- SQLite3 development files
- GCC (for CGO)

```bash
# Ubuntu/Debian
sudo apt install golang-go libsqlite3-dev gcc
```

## Documentation

See [FOUNDATION_README.md](FOUNDATION_README.md) for complete documentation including:
- Detailed function descriptions
- Architecture details
- Multi-company table patterns
- Concurrency model
- Error handling
- NAV-style workflow examples

## Next Steps

After the foundation layer, we'll add:
1. **CRUD Operations** - Insert, Get, Update, Delete, FindSet using GetFullTableName()
2. **Python Integration** - Embed Python for user-defined triggers
3. **Metadata Layer** - Table and field definitions
4. **REST API** - HTTP endpoints for external access
