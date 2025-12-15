"""Object table storage system"""

import sqlite3
import json
from datetime import datetime
from typing import Any, Dict, List, Optional, Union
from pathlib import Path


class Database:
    """
    Object table storage engine for dynamic table management.

    This class provides:
    - Dynamic table creation and management
    - Metadata storage for table definitions
    - Connection management
    - Schema introspection
    """

    def __init__(self, db_path: Union[str, Path] = ":memory:"):
        """
        Initialize database connection.

        Args:
            db_path: Path to SQLite database file or ":memory:" for in-memory DB
        """
        self.db_path = str(db_path)
        self.conn = sqlite3.connect(self.db_path)
        self.conn.row_factory = sqlite3.Row
        self._init_metadata_tables()

    def _init_metadata_tables(self):
        """Initialize internal metadata tables for storing table definitions."""
        cursor = self.conn.cursor()

        # Table to store table metadata
        cursor.execute("""
            CREATE TABLE IF NOT EXISTS __table_metadata (
                table_name TEXT PRIMARY KEY,
                company_id INTEGER,
                schema_definition TEXT,
                on_insert_trigger TEXT,
                on_update_trigger TEXT,
                on_delete_trigger TEXT,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )
        """)

        # Table to store field metadata
        cursor.execute("""
            CREATE TABLE IF NOT EXISTS __field_metadata (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                table_name TEXT,
                field_name TEXT,
                field_type TEXT,
                required INTEGER DEFAULT 0,
                default_value TEXT,
                FOREIGN KEY (table_name) REFERENCES __table_metadata(table_name),
                UNIQUE(table_name, field_name)
            )
        """)

        self.conn.commit()

    def create_table(
        self,
        table_name: str,
        fields: Dict[str, str],
        company_id: Optional[int] = None,
        on_insert: Optional[str] = None,
        on_update: Optional[str] = None,
        on_delete: Optional[str] = None
    ) -> bool:
        """
        Create a new dynamic table with optional triggers.

        Args:
            table_name: Name of the table to create
            fields: Dictionary of field_name: field_type
            company_id: Optional company ID for multi-tenancy
            on_insert: Python code to execute on insert
            on_update: Python code to execute on update
            on_delete: Python code to execute on delete

        Returns:
            True if successful

        Example:
            db.create_table('customers', {
                'name': 'TEXT NOT NULL',
                'email': 'TEXT UNIQUE',
                'balance': 'REAL DEFAULT 0'
            })
        """
        cursor = self.conn.cursor()

        # Check if table already exists
        cursor.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name=?",
            (table_name,)
        )
        if cursor.fetchone():
            raise ValueError(f"Table '{table_name}' already exists")

        # Build CREATE TABLE statement
        # Always include id as primary key
        field_defs = ["id INTEGER PRIMARY KEY AUTOINCREMENT"]

        for field_name, field_type in fields.items():
            field_defs.append(f"{field_name} {field_type}")

        # Add company_id if specified
        if company_id is not None:
            field_defs.append("company_id INTEGER")

        # Add audit fields
        field_defs.extend([
            "created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
            "updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP"
        ])

        create_sql = f"CREATE TABLE {table_name} ({', '.join(field_defs)})"
        cursor.execute(create_sql)

        # Store metadata
        schema_json = json.dumps(fields)
        cursor.execute("""
            INSERT INTO __table_metadata
            (table_name, company_id, schema_definition, on_insert_trigger,
             on_update_trigger, on_delete_trigger)
            VALUES (?, ?, ?, ?, ?, ?)
        """, (table_name, company_id, schema_json, on_insert, on_update, on_delete))

        # Store field metadata
        for field_name, field_type in fields.items():
            required = 1 if "NOT NULL" in field_type.upper() else 0
            cursor.execute("""
                INSERT INTO __field_metadata
                (table_name, field_name, field_type, required)
                VALUES (?, ?, ?, ?)
            """, (table_name, field_name, field_type, required))

        self.conn.commit()
        return True

    def get_table_metadata(self, table_name: str) -> Optional[Dict[str, Any]]:
        """Get metadata for a specific table."""
        cursor = self.conn.cursor()
        cursor.execute(
            "SELECT * FROM __table_metadata WHERE table_name = ?",
            (table_name,)
        )
        row = cursor.fetchone()
        if row:
            return dict(row)
        return None

    def list_tables(self, company_id: Optional[int] = None) -> List[str]:
        """
        List all tables, optionally filtered by company.

        Args:
            company_id: Optional company ID filter

        Returns:
            List of table names
        """
        cursor = self.conn.cursor()
        if company_id is not None:
            cursor.execute(
                "SELECT table_name FROM __table_metadata WHERE company_id = ?",
                (company_id,)
            )
        else:
            cursor.execute("SELECT table_name FROM __table_metadata")

        return [row[0] for row in cursor.fetchall()]

    def drop_table(self, table_name: str) -> bool:
        """
        Drop a table and its metadata.

        Args:
            table_name: Name of table to drop

        Returns:
            True if successful
        """
        cursor = self.conn.cursor()

        # Drop the actual table
        cursor.execute(f"DROP TABLE IF EXISTS {table_name}")

        # Remove metadata
        cursor.execute("DELETE FROM __table_metadata WHERE table_name = ?", (table_name,))
        cursor.execute("DELETE FROM __field_metadata WHERE table_name = ?", (table_name,))

        self.conn.commit()
        return True

    def execute(self, sql: str, params: tuple = ()) -> sqlite3.Cursor:
        """Execute raw SQL query."""
        cursor = self.conn.cursor()
        cursor.execute(sql, params)
        self.conn.commit()
        return cursor

    def close(self):
        """Close database connection."""
        self.conn.close()

    def __enter__(self):
        """Context manager entry."""
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit."""
        self.close()
