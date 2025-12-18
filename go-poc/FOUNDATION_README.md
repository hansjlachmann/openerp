# OpenERP Foundation Layer

Interactive command-line application for testing the OpenERP foundation layer with 8 core functions.

## Architecture

### Per-Connection State Design
Each `Database` object maintains its own session state:
```go
type Database struct {
    conn           *sql.DB
    path           string
    currentCompany string  // Per-connection state for thread safety
}
```

This design ensures:
- **Thread Safety**: Multiple users can have concurrent connections with isolated sessions
- **NAV-Style Experience**: Each user has their own EnterCompany/ExitCompany session
- **Simple API**: Clean, stateful operations without passing company everywhere

## Core Functions

### 1. CreateDatabase(path string) (*Database, error)
Creates a new persistent database file and initializes the Company table.

**Example**: `openerp.db`, `production.db`

### 2. OpenDatabase(path string) (*Database, error)
Opens an existing OpenERP database and verifies it's valid.

**Error if**: File doesn't exist or Company table missing

### 3. CloseDatabase() error
Closes the database connection and clears any active company session.

### 4. CreateCompany(name string) error
Creates a new company in the database.

**Validation**:
- Name cannot be empty
- No special characters (space, $, ", ', `, \)
- Must be unique

### 5. EnterCompany(name string) error
Sets the current company context for this database connection (session).

**Error if**: Company doesn't exist

### 6. ExitCompany() error
Clears the current company context.

**Error if**: No company session active

### 7. DeleteCompany(name string) error
Deletes a company and ALL its Company$Tables.

**What it does**:
1. Finds all tables matching pattern: `CompanyName$%`
2. Drops each table
3. Deletes company record
4. Exits company session if currently active

### 8. ListCompanies() ([]string, error)
Returns all companies in the database, sorted by name.

## Building

```bash
cd go-poc

# Build the interactive menu application
cd cmd/interactive && go build -o openerp-foundation . && cd ../..

# Build the test suite
cd cmd/test && go build -o test-foundation . && cd ../..

# Or run directly
cd cmd/interactive && go run . && cd ../..  # Interactive menu
cd cmd/test && go run . && cd ../..          # Run tests
```

## Usage

### Interactive Menu

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

Select option (1-9):
```

### Example Session

```
1. Create new database
   Path: myerp.db
   ✓ Database created successfully

4. Create company
   Name: ACME
   ✓ Company created successfully

4. Create company
   Name: GLOBEX
   ✓ Company created successfully

8. List companies
   ✓ Found 2 companies:
     1. ACME
     2. GLOBEX

5. Enter company
   Name: ACME
   ✓ Entered company ACME

   [Database: myerp.db | Company: ACME]

6. Exit company
   ✓ Exited company session

7. Delete company
   Name: GLOBEX
   ⚠️  WARNING: This will delete company 'GLOBEX' and ALL its tables!
   Are you sure? yes
   ✓ Company GLOBEX deleted successfully

3. Close database
   ✓ Database closed successfully

9. Exit application
   ✓ Goodbye!
```

## Multi-Company Table Pattern

When you enter a company and create tables, they use the pattern:
```
CompanyName$TableName
```

Examples:
- `ACME$customers`
- `ACME$orders`
- `GLOBEX$customers`
- `GLOBEX$orders`

This ensures complete data isolation between companies at the physical table level.

## Concurrency Model

### Multiple Users
Each user gets their own `Database` connection object:

```go
// User 1
db1, _ := OpenDatabase("openerp.db")
db1.EnterCompany("ACME")

// User 2 (concurrent, independent)
db2, _ := OpenDatabase("openerp.db")
db2.EnterCompany("GLOBEX")
```

Both users work independently without interfering with each other.

## Error Handling

The foundation layer provides clear error messages:

```go
// No database open
err: "database not open"

// Company doesn't exist
err: "company 'INVALID' does not exist"

// Already in company
err: "already in company 'ACME'. Exit first."

// No company session
err: "no company context set - use EnterCompany() first"
```

## Next Steps

After the foundation layer is tested:
1. Add CRUD operations (using GetFullTableName)
2. Add Python trigger embedding
3. Add metadata management
4. Add HTTP REST API layer

## Files

- `foundation.go` - Core foundation layer library
- `cmd/interactive/` - Interactive menu application
  - `main_foundation.go` - Menu implementation
  - `foundation.go` - Copy of foundation library
- `cmd/test/` - Automated test suite
  - `test_foundation.go` - Test implementation
  - `foundation.go` - Copy of foundation library
- `FOUNDATION_README.md` - This file

## NAV-Style Workflow

This matches the NAV/C-AL pattern:

```c
// NAV C/AL style:
Company.GET('ACME');
Customer.SETRANGE("Company", Company.Name);
Customer.INSERT;

// Our Go style:
db.EnterCompany("ACME")
tableName, _ := db.GetFullTableName("customers")
crud.Insert(tableName, record)
```
