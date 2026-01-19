# OpenERP

## Project Structure

```
openerp/
├── cmd/api-server/      # Go API server entry point
├── backend/
│   ├── api/             # HTTP handlers, middleware, server
│   ├── business-logic/  # Tables, pages, codeunits definitions
│   └── foundation/      # Core framework (types, database, i18n, etc.)
├── frontend/            # SvelteKit frontend
├── tools/tablegen/      # Code generation tool
└── translations/        # i18n files (en-US, nb-NO)
```

## Build Commands

### API Server (Go)
```bash
# Build
cd cmd/api-server && go build -o api-server .

# Run (from project root)
./cmd/api-server/api-server

# Run without building
go run ./cmd/api-server
```

### Frontend (SvelteKit)
```bash
cd frontend

# Install dependencies
npm install

# Development server
npm run dev

# Production build
npm run build
```

### Code Generation (tablegen)
```bash
cd tools/tablegen && go build -o tablegen .

# Generate table code (from table definitions directory)
cd backend/business-logic/tables
../../../tools/tablegen/tablegen -input=definitions/customer.yaml -output=.
```

### General Go Commands
```bash
# Format code
go fmt ./...

# Check code
go vet ./...

# Update dependencies
go mod tidy
```

## Database

Connect to SQLite database:
```bash
sqlite3 test.db
```

Useful SQLite commands:
```sql
-- List all tables
.tables

-- Show table schema
.schema "cronus$Payment Terms"

-- Show all data with formatting
.mode column
.headers on
SELECT * FROM "cronus$Payment Terms";

-- Exit
.exit
```
