"""
Database query helper - Explore the sample database with SQL

This script provides useful SQL queries to explore the database structure
and data.
"""

import sqlite3
import sys


def execute_query(db_file, query, description):
    """Execute a query and display results."""
    print(f"\n{'='*70}")
    print(f"Query: {description}")
    print(f"{'='*70}")
    print(f"SQL: {query}\n")

    conn = sqlite3.connect(db_file)
    conn.row_factory = sqlite3.Row
    cursor = conn.cursor()

    try:
        cursor.execute(query)
        rows = cursor.fetchall()

        if rows:
            # Print column headers
            columns = rows[0].keys()
            header = " | ".join(columns)
            print(header)
            print("-" * len(header))

            # Print rows
            for row in rows:
                values = " | ".join(str(row[col]) for col in columns)
                print(values)

            print(f"\nRows returned: {len(rows)}")
        else:
            print("No rows returned.")
    except Exception as e:
        print(f"Error: {e}")
    finally:
        conn.close()


def main():
    db_file = 'sample.db'

    if len(sys.argv) > 1:
        db_file = sys.argv[1]

    print(f"Exploring database: {db_file}\n")

    # Query 1: List all tables
    execute_query(
        db_file,
        "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name",
        "List all tables"
    )

    # Query 2: Show Company table
    execute_query(
        db_file,
        "SELECT * FROM Company",
        "All companies"
    )

    # Query 3: Show table metadata
    execute_query(
        db_file,
        """SELECT table_name, company_name, is_global
           FROM __table_metadata
           ORDER BY table_name""",
        "Table metadata (which tables belong to which companies)"
    )

    # Query 4: Show field metadata
    execute_query(
        db_file,
        """SELECT table_name, field_name, field_type, required
           FROM __field_metadata
           WHERE table_name LIKE '%customers%'
           ORDER BY table_name, field_name""",
        "Field definitions for customer tables"
    )

    # Query 5: Show ACME customers
    execute_query(
        db_file,
        'SELECT * FROM "ACME$customers"',
        "ACME customers (note: emails are lowercased by trigger)"
    )

    # Query 6: Show Globex customers
    execute_query(
        db_file,
        'SELECT * FROM "Globex$customers"',
        "Globex customers"
    )

    # Query 7: Show translations (stored as JSON in metadata)
    execute_query(
        db_file,
        """SELECT table_name, translations
           FROM __table_metadata
           WHERE translations IS NOT NULL
           ORDER BY table_name""",
        "Table translations (stored as JSON)"
    )

    execute_query(
        db_file,
        """SELECT table_name, field_name, translations
           FROM __field_metadata
           WHERE translations IS NOT NULL
           ORDER BY table_name, field_name""",
        "Field translations (stored as JSON)"
    )

    # Query 8: Count customers per company
    execute_query(
        db_file,
        """SELECT
             'ACME' as company, COUNT(*) as customer_count
           FROM "ACME$customers"
           UNION ALL
           SELECT
             'Globex' as company, COUNT(*) as customer_count
           FROM "Globex$customers"
        """,
        "Customer count by company"
    )

    # Query 9: Show triggers stored in metadata
    execute_query(
        db_file,
        """SELECT
             table_name,
             CASE WHEN on_insert_trigger IS NOT NULL THEN 'Yes' ELSE 'No' END as has_insert,
             CASE WHEN on_update_trigger IS NOT NULL THEN 'Yes' ELSE 'No' END as has_update,
             CASE WHEN on_delete_trigger IS NOT NULL THEN 'Yes' ELSE 'No' END as has_delete
           FROM __table_metadata
           WHERE table_name LIKE '%customers%'
        """,
        "Triggers defined on customer tables"
    )

    print("\n" + "="*70)
    print("Exploration complete!")
    print("="*70)
    print("\nTo run custom SQL queries, use:")
    print(f"  sqlite3 {db_file}")
    print("\nThen type your SQL queries, for example:")
    print('  SELECT * FROM "ACME$customers" WHERE balance > 1000;')
    print("  .tables                    -- List all tables")
    print("  .schema Company            -- Show table structure")
    print("  .quit                      -- Exit sqlite3")


if __name__ == "__main__":
    main()
