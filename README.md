# OpenERP - NAV-Style ERP System in Go

A modern, lightweight ERP system inspired by Microsoft Dynamics NAV/Business Central, built entirely in Go with SQLite.

## 🎯 Project Vision

OpenERP is a 100% Go-based ERP system that implements NAV-style architecture:
- **User-defined primary keys** (not auto-increment IDs)
- **Multi-company support** with automatic schema synchronization
- **Object Designer** for database structure management
- **Native desktop GUI** and CLI interfaces
- **Zero external dependencies** for runtime (single binary deployment)

## ✨ Key Features

### ✅ NAV-Style Primary Keys
- User-defined primary keys with business meaning (e.g., "Customer No.", "Document Type + No.")
- Composite key support (multiple fields as primary key)
- No hidden auto-increment IDs

### ✅ Multi-Company Architecture
- All tables stored as `CompanyName$TableName`
- Structure changes automatically replicate to ALL companies
- Data isolation per company
- Example: `ACME$Customer`, `FABRIKAM$Customer`

### ✅ Object Designer
- Create tables
- Add fields with type selection (Text, Boolean, Date, Decimal, Integer)
- Mark primary key fields
- List/delete tables
- View field definitions

### ✅ Data Manager
- Full CRUD operations (Create, Read, Update, Delete)
- Primary key-based record identification
- View all records with primary keys highlighted
- Form-based data entry

### ✅ Two User Interfaces
1. **CLI** - Terminal-based interactive menus
2. **GUI** - Native desktop app with Fyne

## 📁 Project Structure

```
openerp/
├── go-poc/
│   ├── foundation/               # CLI Application
│   │   ├── main.go               # Main entry point
│   │   ├── foundation.go         # Core database operations
│   │   ├── go.mod                # Go module definition
│   │   ├── types/                # Shared type definitions
│   │   ├── object_designer/      # Table/field management UI
│   │   └── data_manager/         # CRUD operations UI
│   │
│   └── gui/                      # GUI Application (Fyne)
│       ├── main.go               # GUI screens and logic
│       ├── database.go           # Database operations wrapper
│       └── README.md             # GUI-specific documentation
│
└── README.md                     # This file
```

## 🚀 Quick Start

### Prerequisites
- **Go 1.21+** - [Download](https://golang.org/dl/)
- **GCC** (for Fyne GUI on Linux/macOS)

### Running the CLI

```bash
cd go-poc/foundation
go build -o openerp-cli
./openerp-cli
```

Or run directly:
```bash
go run .
```

### Running the GUI

```bash
cd go-poc/gui

# Install Fyne (first time only)
go get fyne.io/fyne/v2

# Install platform dependencies
# Ubuntu/Debian:
sudo apt-get install gcc libgl1-mesa-dev xorg-dev

# macOS:
xcode-select --install

# Build and run
go build -o openerp-gui
./openerp-gui
```

See [`gui/README.md`](go-poc/gui/README.md) for detailed GUI instructions.

## 📖 Usage Guide

### 1. Create a Database
Both CLI and GUI will prompt for database path on first run (default: `erp.db`)

### 2. Create a Company
```
1. Enter company name (e.g., "ACME", "FABRIKAM")
2. System creates company record
3. All future tables will be prefixed with company name
```

### 3. Design Your Tables (Object Designer)

**Example: Customer Table**
```
1. Create Table: "Customer"
2. Add Fields:
   - No (Text, Primary Key)          ← User-defined key!
   - Name (Text)
   - Email (Text)
   - Active (Boolean)
   - Credit_Limit (Decimal)
```

**Important**: Add all primary key fields BEFORE adding non-primary key fields!

### 4. Manage Data (Data Manager)

**Add Customer Record**:
```
No: C-10000
Name: ACME Corporation
Email: contact@acme.com
Active: yes
Credit_Limit: 50000.00
```

**Update Record**:
```
Enter PK: C-10000
Update fields (leave blank to skip):
  Email: sales@acme.com
  Credit_Limit: 75000.00
```

## 🏗️ Architecture

### Database Layer
```
SQLite Database
    ├── Company Table
    ├── FieldDefinition Table (metadata)
    └── CompanyName$TableName Tables (actual data)
```

### Core Components

1. **Foundation Layer** (`foundation.go`)
   - Database connection management
   - Company operations
   - Table/field CRUD
   - Record CRUD with primary key support
   - Multi-company synchronization

2. **Type System** (`types/types.go`)
   - FieldInfo (Name, Type, IsPrimaryKey, FieldOrder)
   - Common data structures

3. **Object Designer** (UI package)
   - Table creation/deletion
   - Field management
   - Primary key designation

4. **Data Manager** (UI package)
   - Record viewing (all/single)
   - Record creation
   - Record updates (PK fields protected)
   - Record deletion

### Key Design Decisions

#### NAV-Style Primary Keys
- Tables have user-defined primary keys (no hidden ID)
- Primary keys can be composite (multiple fields)
- Primary keys cannot be updated (recreate record instead)
- On-demand table creation when first used

#### Multi-Company Implementation
```sql
-- Instead of:
CREATE TABLE customer (id INTEGER PRIMARY KEY, name TEXT);

-- We create:
CREATE TABLE ACME$customer (no TEXT PRIMARY KEY, name TEXT);
CREATE TABLE FABRIKAM$customer (no TEXT PRIMARY KEY, name TEXT);
```

#### Metadata-Driven Schema
- Field definitions stored in FieldDefinition table
- Supports late table creation (defined fields → SQL table)
- `ensureTableExists()` creates tables on-demand from metadata

## 🔧 Development

### Building

**CLI**:
```bash
cd go-poc/foundation
go build -o openerp-cli
```

**GUI**:
```bash
cd go-poc/gui
go build -o openerp-gui
```

### Cross-Compilation

**Windows from Linux/Mac**:
```bash
GOOS=windows GOARCH=amd64 go build -o openerp.exe
```

**macOS from Linux/Windows**:
```bash
GOOS=darwin GOARCH=amd64 go build -o openerp
```

### Testing

Currently uses manual testing. Future: Add Go tests for foundation layer.

```bash
# Run CLI and test all features
./openerp-cli

# Test sequence:
# 1. Create company
# 2. Create table
# 3. Add PK field
# 4. Add regular fields
# 5. Insert records
# 6. Update/delete records
```

## 📊 Supported Field Types

| Type    | Go Type  | SQLite Type | Example Values          |
|---------|----------|-------------|-------------------------|
| Text    | string   | TEXT        | "John Doe", "ABC-123"   |
| Boolean | int      | INTEGER     | 1 (true), 0 (false)     |
| Date    | string   | TEXT        | "2024-01-15"            |
| Decimal | float64  | REAL        | 99.95, 1234.56          |
| Integer | int64    | INTEGER     | 42, 1000                |

## 🎯 Roadmap / Future Enhancements

### Phase 1: Foundation ✅ COMPLETE
- [x] Multi-company support
- [x] NAV-style primary keys
- [x] Object Designer (tables/fields)
- [x] Data Manager (CRUD)
- [x] CLI interface
- [x] GUI interface

### Phase 2: Business Logic (Next)
- [ ] Table relationships (foreign keys)
- [ ] Field validation rules
- [ ] Default values
- [ ] Calculated fields
- [ ] Triggers/business events

### Phase 3: Advanced Features
- [ ] Import/Export (CSV, Excel)
- [ ] Reports (PDF generation)
- [ ] Search/filtering system
- [ ] Audit trail (change tracking)
- [ ] User authentication & permissions

### Phase 4: Domain Modules
- [ ] Sales module (quotes, orders, invoices)
- [ ] Purchase module
- [ ] Inventory management
- [ ] General ledger
- [ ] CRM basics

## 🤝 Contributing

This is currently a personal project. For suggestions or issues:
1. Test the feature thoroughly
2. Document the issue/enhancement
3. Provide example use case

## 📄 License

[Specify your license here]

## 🎓 Learning Resources

### Go Language
- Official Go Tour: https://go.dev/tour/
- Go by Example: https://gobyexample.com/

### Fyne GUI Framework
- Documentation: https://docs.fyne.io
- Widget Gallery: https://docs.fyne.io/widget/
- Examples: https://github.com/fyne-io/examples

### Microsoft Dynamics NAV/Business Central
- NAV Documentation: https://learn.microsoft.com/dynamics-nav/
- BC Development: https://learn.microsoft.com/dynamics365/business-central/dev-itpro/

## 📈 Project Status

**Current Version**: Phase 1 - Foundation Complete
**Status**: Active Development
**Language**: 100% Go (Python code removed)
**Database**: SQLite (single file, no server required)
**Deployment**: Single binary (CLI + GUI)

---

## 🔗 Quick Links

- **CLI Application**: [`go-poc/foundation/`](go-poc/foundation/)
- **GUI Application**: [`go-poc/gui/`](go-poc/gui/)
- **GUI Documentation**: [`go-poc/gui/README.md`](go-poc/gui/README.md)

---

**Built with ❤️ in Go** | **Inspired by NAV/Business Central Architecture**
