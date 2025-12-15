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
    - Company-specific tables (CompanyName$TableName)
    - Global tables (available to all companies)
    - Metadata storage for table definitions
    - Connection management
    - Schema introspection

    Table Naming Convention:
    - Global tables: TableName (e.g., "Company", "SystemSettings")
    - Company-specific tables: CompanyName$TableName (e.g., "ACME$Customers")
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
                company_name TEXT,
                is_global INTEGER DEFAULT 0,
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

    @staticmethod
    def get_full_table_name(table_name: str, company_name: Optional[str] = None) -> str:
        """
        Get the full table name with company prefix if applicable.

        Args:
            table_name: Base table name
            company_name: Company name (None for global tables)

        Returns:
            Full table name (CompanyName$TableName or TableName)

        Example:
            get_full_table_name("Customers", "ACME") -> "ACME$Customers"
            get_full_table_name("Company", None) -> "Company"
        """
        if company_name:
            return f"{company_name}${table_name}"
        return table_name

    @staticmethod
    def parse_table_name(full_table_name: str) -> tuple[Optional[str], str]:
        """
        Parse a full table name into company name and table name.

        Args:
            full_table_name: Full table name (may include company prefix)

        Returns:
            Tuple of (company_name, table_name)

        Example:
            parse_table_name("ACME$Customers") -> ("ACME", "Customers")
            parse_table_name("Company") -> (None, "Company")
        """
        if '$' in full_table_name:
            parts = full_table_name.split('$', 1)
            return parts[0], parts[1]
        return None, full_table_name

    def create_table(
        self,
        table_name: str,
        fields: Dict[str, str],
        company_name: Optional[str] = None,
        on_insert: Optional[str] = None,
        on_update: Optional[str] = None,
        on_delete: Optional[str] = None,
        is_global: bool = False
    ) -> bool:
        """
        Create a new dynamic table with optional triggers.

        Args:
            table_name: Base name of the table to create
            fields: Dictionary of field_name: field_type
            company_name: Company name for company-specific table
            on_insert: Python code to execute on insert
            on_update: Python code to execute on update
            on_delete: Python code to execute on delete
            is_global: If True, creates a global table (ignores company_name)

        Returns:
            True if successful

        Example:
            # Global table
            db.create_table('SystemSettings', {...}, is_global=True)

            # Company-specific table
            db.create_table('Customers', {...}, company_name='ACME')
            # Creates: ACME$Customers
        """
        if is_global:
            return self._create_global_table(table_name, fields, on_insert, on_update, on_delete)
        else:
            if not company_name:
                raise ValueError("company_name is required for company-specific tables")
            return self._create_company_table(
                table_name, company_name, fields, on_insert, on_update, on_delete
            )

    def _create_global_table(
        self,
        table_name: str,
        fields: Dict[str, str],
        on_insert: Optional[str] = None,
        on_update: Optional[str] = None,
        on_delete: Optional[str] = None
    ) -> bool:
        """Create a global table (accessible to all companies)."""
        cursor = self.conn.cursor()

        # Check if table already exists
        cursor.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name=?",
            (table_name,)
        )
        if cursor.fetchone():
            raise ValueError(f"Global table '{table_name}' already exists")

        # Build CREATE TABLE statement
        field_defs = ["id INTEGER PRIMARY KEY AUTOINCREMENT"]

        for field_name, field_type in fields.items():
            field_defs.append(f"{field_name} {field_type}")

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
            (table_name, company_name, is_global, schema_definition, on_insert_trigger,
             on_update_trigger, on_delete_trigger)
            VALUES (?, NULL, 1, ?, ?, ?, ?)
        """, (table_name, schema_json, on_insert, on_update, on_delete))

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

    def _create_company_table(
        self,
        table_name: str,
        company_name: str,
        fields: Dict[str, str],
        on_insert: Optional[str] = None,
        on_update: Optional[str] = None,
        on_delete: Optional[str] = None
    ) -> bool:
        """Create a company-specific table (CompanyName$TableName)."""
        cursor = self.conn.cursor()

        # Build full table name with company prefix
        full_table_name = self.get_full_table_name(table_name, company_name)

        # Check if table already exists
        cursor.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name=?",
            (full_table_name,)
        )
        if cursor.fetchone():
            raise ValueError(f"Company table '{full_table_name}' already exists")

        # Build CREATE TABLE statement
        field_defs = ["id INTEGER PRIMARY KEY AUTOINCREMENT"]

        for field_name, field_type in fields.items():
            field_defs.append(f"{field_name} {field_type}")

        # Add audit fields
        field_defs.extend([
            "created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
            "updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP"
        ])

        create_sql = f'CREATE TABLE "{full_table_name}" ({", ".join(field_defs)})'
        cursor.execute(create_sql)

        # Store metadata
        schema_json = json.dumps(fields)
        cursor.execute("""
            INSERT INTO __table_metadata
            (table_name, company_name, is_global, schema_definition, on_insert_trigger,
             on_update_trigger, on_delete_trigger)
            VALUES (?, ?, 0, ?, ?, ?, ?)
        """, (full_table_name, company_name, schema_json, on_insert, on_update, on_delete))

        # Store field metadata
        for field_name, field_type in fields.items():
            required = 1 if "NOT NULL" in field_type.upper() else 0
            cursor.execute("""
                INSERT INTO __field_metadata
                (table_name, field_name, field_type, required)
                VALUES (?, ?, ?, ?)
            """, (full_table_name, field_name, field_type, required))

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

    def list_tables(self, company_name: Optional[str] = None, include_global: bool = True) -> List[str]:
        """
        List all tables, optionally filtered by company.

        Args:
            company_name: Optional company name filter (returns CompanyName$TableName tables)
            include_global: If True, includes global tables in the result

        Returns:
            List of full table names

        Example:
            list_tables() -> ["Company", "ACME$Customers", "ACME$Orders"]
            list_tables("ACME") -> ["Company", "ACME$Customers", "ACME$Orders"]
            list_tables("ACME", include_global=False) -> ["ACME$Customers", "ACME$Orders"]
        """
        cursor = self.conn.cursor()

        if company_name is not None:
            if include_global:
                cursor.execute(
                    "SELECT table_name FROM __table_metadata WHERE company_name = ? OR is_global = 1",
                    (company_name,)
                )
            else:
                cursor.execute(
                    "SELECT table_name FROM __table_metadata WHERE company_name = ?",
                    (company_name,)
                )
        else:
            cursor.execute("SELECT table_name FROM __table_metadata")

        return [row[0] for row in cursor.fetchall()]

    def list_global_tables(self) -> List[str]:
        """
        List all global tables.

        Returns:
            List of global table names
        """
        cursor = self.conn.cursor()
        cursor.execute("SELECT table_name FROM __table_metadata WHERE is_global = 1")
        return [row[0] for row in cursor.fetchall()]

    def list_company_tables(self, company_name: str, base_names_only: bool = False) -> List[str]:
        """
        List all tables for a specific company.

        Args:
            company_name: Company name
            base_names_only: If True, returns just the table names without company prefix

        Returns:
            List of table names

        Example:
            list_company_tables("ACME") -> ["ACME$Customers", "ACME$Orders"]
            list_company_tables("ACME", base_names_only=True) -> ["Customers", "Orders"]
        """
        cursor = self.conn.cursor()
        cursor.execute(
            "SELECT table_name FROM __table_metadata WHERE company_name = ?",
            (company_name,)
        )

        tables = [row[0] for row in cursor.fetchall()]

        if base_names_only:
            return [self.parse_table_name(t)[1] for t in tables]

        return tables

    def is_global_table(self, table_name: str) -> bool:
        """
        Check if a table is global.

        Args:
            table_name: Table name to check

        Returns:
            True if the table is global
        """
        metadata = self.get_table_metadata(table_name)
        if metadata:
            return bool(metadata['is_global'])
        return False

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
