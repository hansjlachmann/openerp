
#####  Individual commands:
```
  Build CLI:
  go build -o openerp-cli main.go

  Build GUI:
  go build -o openerp-gui ./src/gui/

  Run CLI (without building):
  go run main.go

  Run GUI (without building):
  go run ./src/gui/

  Clean binaries:
  rm openerp-cli openerp-gui

  Format code:
  go fmt ./...

  Check code:
  go vet ./...

  Update dependencies:
  go mod tidy
```

#####  Connect to the database:

  sqlite3 test.db

  Useful SQLite commands:

  1. List all tables:
  .tables

  2. Show table schema:
  .schema "cronus$Payment Terms"

  3. Show all data in Payment Terms table:
  SELECT * FROM "cronus$Payment Terms";

  4. Show data with formatting:
  .mode column
  .headers on
  SELECT * FROM "cronus$Payment Terms";

  5. Count records:
  SELECT COUNT(*) FROM "cronus$Payment Terms";

  6. Show specific records:
  SELECT * FROM "cronus$Payment Terms" WHERE code LIKE 'TEST%';

  7. Show only modified records:
  SELECT code, description, active FROM "cronus$Payment Terms" WHERE code BETWEEN 'TEST001' AND 'TEST020' ORDER BY code;

  8. Exit SQLite:
  .exit
  or press Ctrl+D


