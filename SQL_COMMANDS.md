# SQL Commands for OpenERP Database Exploration

Quick reference for exploring OpenERP databases using sqlite3 in the terminal.

## Getting Started

```bash
# Create sample database first
python examples/create_sample_db.py

# Open database in sqlite3
sqlite3 sample.db

# Enable column headers and formatted output
.headers on
.mode column
```

## Basic SQLite Commands

```sql
-- List all tables
.tables

-- Show structure of a specific table
.schema Company
.schema "ACME$customers"

-- Show all schemas
.schema

-- Exit sqlite3
.quit
```

## Exploring Companies

```sql
-- List all companies
SELECT * FROM Company;

-- Count companies
SELECT COUNT(*) as total_companies FROM Company;
```

## Exploring Tables and Metadata

```sql
-- List all user tables (excluding metadata)
SELECT name FROM sqlite_master
WHERE type='table'
  AND name NOT LIKE '__%'
  AND name != 'sqlite_sequence'
ORDER BY name;

-- Show which tables belong to which companies
SELECT table_name, company_name, is_global
FROM __table_metadata
ORDER BY company_name, table_name;

-- Show fields for a specific table
SELECT field_name, field_type, required
FROM __field_metadata
WHERE table_name = 'ACME$customers';

-- Show all customer tables
SELECT table_name FROM __table_metadata
WHERE table_name LIKE '%customers%';
```

## Querying Customer Data

```sql
-- View all ACME customers
SELECT * FROM "ACME$customers";

-- View all Globex customers
SELECT * FROM "Globex$customers";

-- Find customers with high balance
SELECT name, email, balance
FROM "ACME$customers"
WHERE balance > 1000;

-- Count customers per company
SELECT 'ACME' as company, COUNT(*) as total
FROM "ACME$customers"
UNION ALL
SELECT 'Globex', COUNT(*)
FROM "Globex$customers";

-- Search by name
SELECT * FROM "ACME$customers"
WHERE name LIKE '%Smith%';

-- Search by email domain
SELECT * FROM "ACME$customers"
WHERE email LIKE '%@acme.com';
```

## Viewing Triggers

```sql
-- Show which tables have triggers
SELECT
  table_name,
  CASE WHEN on_insert_trigger IS NOT NULL THEN 'Yes' ELSE 'No' END as has_insert,
  CASE WHEN on_update_trigger IS NOT NULL THEN 'Yes' ELSE 'No' END as has_update,
  CASE WHEN on_delete_trigger IS NOT NULL THEN 'Yes' ELSE 'No' END as has_delete
FROM __table_metadata
WHERE table_name LIKE '%customers%';

-- View trigger code for a specific table
SELECT on_insert_trigger
FROM __table_metadata
WHERE table_name = 'ACME$customers';
```

## Viewing Translations

```sql
-- Show tables with translations
SELECT table_name, translations
FROM __table_metadata
WHERE translations IS NOT NULL;

-- Show field translations
SELECT table_name, field_name, translations
FROM __field_metadata
WHERE translations IS NOT NULL;
```

## Aggregations and Analytics

```sql
-- Total balance across all ACME customers
SELECT SUM(balance) as total_balance
FROM "ACME$customers";

-- Average balance per company
SELECT 'ACME' as company, AVG(balance) as avg_balance
FROM "ACME$customers"
UNION ALL
SELECT 'Globex', AVG(balance)
FROM "Globex$customers";

-- Customers grouped by balance ranges
SELECT
  CASE
    WHEN balance = 0 THEN 'Zero'
    WHEN balance < 1000 THEN 'Low (< $1000)'
    WHEN balance < 2000 THEN 'Medium ($1000-$2000)'
    ELSE 'High (> $2000)'
  END as balance_range,
  COUNT(*) as customer_count
FROM "ACME$customers"
GROUP BY balance_range;
```

## Join Queries (Multi-table)

```sql
-- Join table metadata with field metadata
SELECT
  tm.table_name,
  tm.company_name,
  fm.field_name,
  fm.field_type
FROM __table_metadata tm
JOIN __field_metadata fm ON tm.table_name = fm.table_name
WHERE tm.table_name = 'ACME$customers'
ORDER BY fm.field_name;
```

## Modifying Data (Use with caution!)

```sql
-- Update a customer's balance
UPDATE "ACME$customers"
SET balance = 1500.0
WHERE name = 'John Smith';

-- Insert a new customer
INSERT INTO "ACME$customers" (name, email, phone, balance, created_at, updated_at)
VALUES ('New Customer', 'new@acme.com', '+1-555-9999', 500.0,
        datetime('now'), datetime('now'));

-- Delete a customer
DELETE FROM "ACME$customers"
WHERE name = 'New Customer';
```

## Useful SQLite Settings

```sql
-- Show current settings
.show

-- Change output format
.mode csv          -- CSV format
.mode column       -- Column format (pretty)
.mode list         -- List format
.mode table        -- Table format (very pretty)

-- Output to file
.output results.txt
SELECT * FROM "ACME$customers";
.output stdout

-- Execute SQL from file
.read my_queries.sql
```

## Tips

1. **Always use quotes** around table names with special characters:
   ```sql
   SELECT * FROM "ACME$customers";  -- Correct
   SELECT * FROM ACME$customers;    -- Error!
   ```

2. **Use LIKE for pattern matching**:
   ```sql
   WHERE name LIKE '%John%'    -- Contains 'John'
   WHERE email LIKE '%@acme.com'  -- Ends with @acme.com
   ```

3. **Check table structure before querying**:
   ```sql
   .schema "ACME$customers"
   ```

4. **Limit results for large datasets**:
   ```sql
   SELECT * FROM "ACME$customers" LIMIT 10;
   ```
