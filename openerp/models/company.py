"""Company management module"""

from typing import Dict, List, Optional, Any
from openerp.core.database import Database


class Company:
    """
    Company management for multi-tenant ERP system.

    Each company represents a separate business entity with its own
    company-specific tables (CompanyName$TableName).

    The Company table is a GLOBAL table with Name as the PRIMARY KEY.
    """

    TABLE_NAME = "Company"  # Global table

    def __init__(self, name: str = ""):
        """
        Initialize a Company instance.

        Args:
            name: Company name (PRIMARY KEY)
        """
        self.name = name

    @classmethod
    def _ensure_table_exists(cls, db: Database):
        """Ensure the Company global table exists in the database."""
        cursor = db.conn.cursor()
        cursor.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name=?",
            (cls.TABLE_NAME,)
        )

        if not cursor.fetchone():
            # Create Company global table with Name as PRIMARY KEY
            # Note: This bypasses the standard create_table to avoid auto-id
            cursor.execute(f"""
                CREATE TABLE {cls.TABLE_NAME} (
                    Name TEXT PRIMARY KEY NOT NULL
                )
            """)
            db.conn.commit()

            # Register in metadata as global table
            cursor.execute("""
                INSERT INTO __table_metadata
                (table_name, company_id, schema_definition, on_insert_trigger,
                 on_update_trigger, on_delete_trigger)
                VALUES (?, NULL, ?, NULL, NULL, NULL)
            """, (cls.TABLE_NAME, '{"Name": "TEXT PRIMARY KEY"}'))
            db.conn.commit()

    @classmethod
    def create(cls, db: Database, name: str) -> "Company":
        """
        Create a new company.

        Args:
            db: Database instance
            name: Company name (must be unique, used as PRIMARY KEY)

        Returns:
            Company instance

        Example:
            company = Company.create(db, "ACME")

        Note:
            This name will be used in company-specific tables as: {name}$TableName
        """
        cls._ensure_table_exists(db)

        cursor = db.conn.cursor()

        # Validate name format (no special characters except allowed ones)
        if not cls._is_valid_company_name(name):
            raise ValueError(
                f"Invalid company name '{name}'. "
                "Only alphanumeric characters, underscores, and hyphens are allowed."
            )

        try:
            cursor.execute(f"INSERT INTO {cls.TABLE_NAME} (Name) VALUES (?)", (name,))
            db.conn.commit()
            return cls(name=name)
        except Exception as e:
            if "UNIQUE constraint failed" in str(e):
                raise ValueError(f"Company '{name}' already exists")
            raise

    @classmethod
    def get_by_name(cls, db: Database, name: str) -> Optional["Company"]:
        """
        Retrieve a company by name.

        Args:
            db: Database instance
            name: Company name (PRIMARY KEY)

        Returns:
            Company instance or None
        """
        cls._ensure_table_exists(db)

        cursor = db.conn.cursor()
        cursor.execute(f"SELECT Name FROM {cls.TABLE_NAME} WHERE Name = ?", (name,))
        row = cursor.fetchone()

        if row:
            return cls(name=row[0])
        return None

    @classmethod
    def exists(cls, db: Database, name: str) -> bool:
        """
        Check if a company exists.

        Args:
            db: Database instance
            name: Company name

        Returns:
            True if company exists
        """
        return cls.get_by_name(db, name) is not None

    @classmethod
    def list_all(cls, db: Database) -> List["Company"]:
        """
        List all companies.

        Args:
            db: Database instance

        Returns:
            List of Company instances
        """
        cls._ensure_table_exists(db)

        cursor = db.conn.cursor()
        cursor.execute(f"SELECT Name FROM {cls.TABLE_NAME}")

        rows = cursor.fetchall()
        return [cls(name=row[0]) for row in rows]

    def delete(self, db: Database):
        """
        Delete this company.

        Warning: This will NOT automatically delete company-specific tables.
        You should delete those separately if needed.

        Args:
            db: Database instance
        """
        cursor = db.conn.cursor()
        cursor.execute(f"DELETE FROM {cls.TABLE_NAME} WHERE Name = ?", (self.name,))
        db.conn.commit()

    @staticmethod
    def _is_valid_company_name(name: str) -> bool:
        """
        Validate company name format.

        Only alphanumeric, underscore, and hyphen allowed.
        Must not be empty.
        """
        if not name:
            return False
        import re
        return bool(re.match(r'^[a-zA-Z0-9_-]+$', name))

    def to_dict(self) -> Dict[str, Any]:
        """Convert company to dictionary."""
        return {'name': self.name}

    def __repr__(self):
        """String representation of Company."""
        return f"<Company {self.name}>"

    def __eq__(self, other):
        """Check equality based on name."""
        if not isinstance(other, Company):
            return False
        return self.name == other.name

    def __hash__(self):
        """Hash based on name."""
        return hash(self.name)
