# OpenERP - Fyne GUI Application

A native desktop GUI for the OpenERP NAV-style ERP system, built with Fyne in pure Go.

## Features

✅ **Company Management** - Create and switch between multiple companies
✅ **Object Designer** - Create tables, add fields with primary keys
✅ **Data Manager** - Full CRUD operations with NAV-style primary keys
✅ **Multi-Company Support** - All changes replicate across companies
✅ **Native Desktop App** - Cross-platform (Windows, macOS, Linux)
✅ **Pure Go** - No HTML/CSS/JavaScript required

## Prerequisites

### 1. Install Go
Make sure you have Go 1.21 or later installed:
```bash
go version
```

### 2. Install Fyne Dependencies

#### On Ubuntu/Debian Linux:
```bash
sudo apt-get install gcc libgl1-mesa-dev xorg-dev
```

#### On Fedora/Red Hat:
```bash
sudo dnf install gcc libXcursor-devel libXrandr-devel mesa-libGL-devel libXi-devel libXinerama-devel libXxf86vm-devel
```

#### On macOS:
Xcode or Command Line Tools are required:
```bash
xcode-select --install
```

#### On Windows:
Install MinGW-w64 or use TDM-GCC:
- Download from: https://jmeubank.github.io/tdm-gcc/
- Or use MSYS2: https://www.msys2.org/

## Installation

### Option 1: Quick Install (Recommended)

From the `cmd/gui` directory:

```bash
# Navigate to GUI directory
cd /path/to/openerp/go-poc/cmd/gui

# Install Fyne dependency
go get fyne.io/fyne/v2

# Build the application
go build -o openerp-gui .

# Run it!
./openerp-gui
```

### Option 2: Using Fyne Command (For packaging/distribution)

Install the Fyne command tool:
```bash
go install fyne.io/fyne/v2/cmd/fyne@latest
```

Build with icon and metadata:
```bash
# Build
fyne build -name "OpenERP" -icon icon.png

# Or create installer package
fyne package -name "OpenERP" -icon icon.png
```

## Running the Application

### Development Mode
```bash
cd cmd/gui
go run .
```

### Production Build
```bash
# Build
go build -o openerp-gui .

# Run
./openerp-gui
```

### On Windows
```bash
go build -o openerp-gui.exe .
openerp-gui.exe
```

## Project Structure

```
cmd/gui/
├── main.go          # Main GUI application with all screens
├── database.go      # Database wrapper with foundation logic
├── README.md        # This file
└── go.mod          # Go module file (if separate from parent)
```

## Usage Guide

### 1. Company Selection
- **Open Database**: Enter database path or use default `erp.db`
- **Select Company**: Click on an existing company
- **Create Company**: Enter name and click "Create New Company"

### 2. Object Designer

#### Create Table
1. Click "Create Table"
2. Enter table name (e.g., `Customer`, `Vendor`)
3. Click Submit

#### Add Fields
1. Click "Add Field to Table"
2. Select table from dropdown
3. Enter field name (e.g., `No`, `Name`, `Email`)
4. Select field type (Text, Boolean, Date, Decimal, Integer)
5. **Check "Primary Key"** if this is a key field
6. Click Submit

**Important**: Add all primary key fields BEFORE adding non-primary key fields!

#### List Tables
Shows all tables with their fields and primary keys (marked with 🔑)

#### Delete Table
Select table and confirm deletion (removes from ALL companies)

### 3. Data Manager

#### View All Records
Displays all records with primary keys shown first (marked with *)

#### Add New Record
1. Click "Add New Record"
2. Fill in all fields (primary keys are required)
3. Click Submit

#### View Single Record
1. Enter primary key value(s)
2. Click Submit to view record details

#### Update Record
1. Enter primary key value(s) to identify record
2. Fill in fields to update (leave blank to skip)
3. Primary key fields cannot be updated
4. Click Submit

#### Delete Record
1. Enter primary key value(s)
2. Confirm deletion

## Field Types

| Type    | Description                    | Example          |
|---------|--------------------------------|------------------|
| Text    | String values                  | "John Doe"       |
| Boolean | True/False (enter 1/0, yes/no) | 1, true, yes     |
| Date    | ISO8601 format                 | 2024-01-15       |
| Decimal | Floating point numbers         | 99.95, 1234.5    |
| Integer | Whole numbers                  | 42, 1000         |

## NAV-Style Primary Keys

This system implements Microsoft Dynamics NAV/Business Central style primary keys:

- **User-defined**: You choose the primary key fields (e.g., "No" instead of auto-increment ID)
- **Composite keys**: Multiple fields can form the primary key
- **Business meaning**: Keys have semantic meaning (Customer No., Document Type + No.)
- **No auto-increment**: No hidden ID field - your fields ARE the key

### Example Table Structures

**Customer Table**:
- `No` (Text, Primary Key) → "C-10000"
- `Name` (Text) → "ACME Corporation"
- `Email` (Text) → "contact@acme.com"

**Sales Order Table** (Composite Key):
- `Type` (Text, Primary Key) → "Order"
- `No` (Text, Primary Key) → "SO-10001"
- `Customer_No` (Text) → "C-10000"
- `Amount` (Decimal) → 1500.00

## Multi-Company Architecture

- All tables are stored as `Company$TableName`
- Structure changes replicate to ALL companies automatically
- Data is company-specific
- Example: Creating table `Customer` creates `ACME$Customer`, `FABRIKAM$Customer`, etc.

## Troubleshooting

### Build Errors

**Error: `fyne.io/fyne/v2: module not found`**
```bash
go get fyne.io/fyne/v2
go mod tidy
```

**Error: `gcc not found`** (Linux/macOS)
- Install gcc and development libraries (see Prerequisites above)

**Error: `undefined: fyne.App`**
- Make sure you're in the correct directory (`cmd/gui`)
- Run `go mod tidy` to fix dependencies

### Runtime Issues

**Blank window or UI not showing**
- Make sure Fyne dependencies are correctly installed
- Try running with `go run .` instead of compiled binary

**Database errors**
- Make sure the database file is writable
- Check that you're in a company context before operations

## Building for Distribution

### Windows Executable
```bash
GOOS=windows GOARCH=amd64 go build -o openerp-gui.exe .
```

### macOS App Bundle
```bash
fyne package -os darwin -icon icon.png
```

### Linux Package
```bash
fyne package -os linux -icon icon.png
```

## Comparison with CLI Version

| Feature                | CLI (interactive)  | GUI (Fyne)     |
|------------------------|-------------------|----------------|
| Company Management     | ✅ Text menus      | ✅ Native UI    |
| Object Designer        | ✅ Text prompts    | ✅ Forms/Lists  |
| Data Manager           | ✅ Line-by-line    | ✅ Forms/Tables |
| User Experience        | Terminal-based    | Native desktop |
| Deployment             | Single binary     | Single binary  |
| Cross-platform         | ✅ Yes             | ✅ Yes          |
| Remote Access          | ❌ SSH only        | ❌ Local only   |

**Both versions** share the same database format and business logic!

## Next Steps

Consider adding:
1. **Table designer view** - Visual field editor with drag-and-drop
2. **Import/Export** - CSV/Excel import/export
3. **Reports** - Built-in report generator
4. **Search/Filter** - Advanced record filtering
5. **User preferences** - Save window size, recent databases
6. **Themes** - Light/dark mode support

## Support

For issues or questions:
- Check the main OpenERP documentation
- Review Fyne documentation: https://docs.fyne.io
- File an issue on the project repository

## License

Same as the parent OpenERP project.
