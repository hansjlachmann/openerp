"""Company management module"""

from typing import Dict, List, Optional, Any
from datetime import datetime
from openerp.core.database import Database


class Company:
    """
    Company management for multi-tenant ERP system.

    Each company represents a separate business entity with its own
    data isolation within the system.
    """

    TABLE_NAME = "companies"

    def __init__(
        self,
        id: Optional[int] = None,
        code: str = "",
        name: str = "",
        legal_name: Optional[str] = None,
        tax_id: Optional[str] = None,
        currency: str = "USD",
        active: bool = True,
        parent_id: Optional[int] = None,
        created_at: Optional[datetime] = None,
        updated_at: Optional[datetime] = None
    ):
        """Initialize a Company instance."""
        self.id = id
        self.code = code
        self.name = name
        self.legal_name = legal_name or name
        self.tax_id = tax_id
        self.currency = currency
        self.active = active
        self.parent_id = parent_id
        self.created_at = created_at
        self.updated_at = updated_at

    @classmethod
    def _ensure_table_exists(cls, db: Database):
        """Ensure the companies table exists in the database."""
        cursor = db.conn.cursor()
        cursor.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name=?",
            (cls.TABLE_NAME,)
        )

        if not cursor.fetchone():
            # Create companies table
            db.create_table(
                cls.TABLE_NAME,
                {
                    'code': 'TEXT NOT NULL UNIQUE',
                    'name': 'TEXT NOT NULL',
                    'legal_name': 'TEXT',
                    'tax_id': 'TEXT',
                    'currency': 'TEXT DEFAULT "USD"',
                    'active': 'INTEGER DEFAULT 1',
                    'parent_id': 'INTEGER',
                },
                on_insert="""
# Validate company code
if not record.get('code'):
    raise ValueError("Company code is required")

# Uppercase the code
record['code'] = record['code'].upper()

# Set legal_name to name if not provided
if not record.get('legal_name'):
    record['legal_name'] = record['name']

print(f"Creating company: {record['name']} ({record['code']})")
"""
            )

    @classmethod
    def create(
        cls,
        db: Database,
        code: str,
        name: str,
        legal_name: Optional[str] = None,
        tax_id: Optional[str] = None,
        currency: str = "USD",
        parent_id: Optional[int] = None
    ) -> "Company":
        """
        Create a new company.

        Args:
            db: Database instance
            code: Unique company code (e.g., "ACME")
            name: Company display name
            legal_name: Legal company name (defaults to name)
            tax_id: Tax identification number
            currency: Default currency code
            parent_id: Parent company ID for subsidiaries

        Returns:
            Company instance

        Example:
            company = Company.create(
                db,
                code="ACME",
                name="Acme Corporation",
                tax_id="12-3456789"
            )
        """
        cls._ensure_table_exists(db)

        cursor = db.conn.cursor()

        # Import here to avoid circular dependency
        from openerp.core.crud import CRUDManager
        crud = CRUDManager(db)

        record = {
            'code': code.upper(),
            'name': name,
            'legal_name': legal_name or name,
            'tax_id': tax_id,
            'currency': currency,
            'active': 1,
            'parent_id': parent_id
        }

        result = crud.insert(cls.TABLE_NAME, record)

        if not result['success']:
            raise ValueError(f"Failed to create company: {result['errors']}")

        return cls.get_by_id(db, result['id'])

    @classmethod
    def get_by_id(cls, db: Database, company_id: int) -> Optional["Company"]:
        """
        Retrieve a company by ID.

        Args:
            db: Database instance
            company_id: Company ID

        Returns:
            Company instance or None
        """
        cls._ensure_table_exists(db)

        cursor = db.conn.cursor()
        cursor.execute(f"SELECT * FROM {cls.TABLE_NAME} WHERE id = ?", (company_id,))
        row = cursor.fetchone()

        if row:
            return cls._from_db_row(dict(row))
        return None

    @classmethod
    def get_by_code(cls, db: Database, code: str) -> Optional["Company"]:
        """
        Retrieve a company by code.

        Args:
            db: Database instance
            code: Company code

        Returns:
            Company instance or None
        """
        cls._ensure_table_exists(db)

        cursor = db.conn.cursor()
        cursor.execute(
            f"SELECT * FROM {cls.TABLE_NAME} WHERE code = ?",
            (code.upper(),)
        )
        row = cursor.fetchone()

        if row:
            return cls._from_db_row(dict(row))
        return None

    @classmethod
    def list_all(cls, db: Database, active_only: bool = True) -> List["Company"]:
        """
        List all companies.

        Args:
            db: Database instance
            active_only: If True, only return active companies

        Returns:
            List of Company instances
        """
        cls._ensure_table_exists(db)

        cursor = db.conn.cursor()

        if active_only:
            cursor.execute(f"SELECT * FROM {cls.TABLE_NAME} WHERE active = 1")
        else:
            cursor.execute(f"SELECT * FROM {cls.TABLE_NAME}")

        rows = cursor.fetchall()
        return [cls._from_db_row(dict(row)) for row in rows]

    def update(self, db: Database, **kwargs):
        """
        Update company fields.

        Args:
            db: Database instance
            **kwargs: Fields to update
        """
        from openerp.core.crud import CRUDManager
        crud = CRUDManager(db)

        result = crud.update(self.TABLE_NAME, self.id, kwargs)

        if not result['success']:
            raise ValueError(f"Failed to update company: {result['errors']}")

        # Update instance attributes
        for key, value in kwargs.items():
            if hasattr(self, key):
                setattr(self, key, value)

    def deactivate(self, db: Database):
        """Deactivate the company."""
        self.update(db, active=False)

    def to_dict(self) -> Dict[str, Any]:
        """Convert company to dictionary."""
        return {
            'id': self.id,
            'code': self.code,
            'name': self.name,
            'legal_name': self.legal_name,
            'tax_id': self.tax_id,
            'currency': self.currency,
            'active': self.active,
            'parent_id': self.parent_id,
            'created_at': self.created_at.isoformat() if self.created_at else None,
            'updated_at': self.updated_at.isoformat() if self.updated_at else None,
        }

    @classmethod
    def _from_db_row(cls, row: Dict[str, Any]) -> "Company":
        """Create Company instance from database row."""
        return cls(
            id=row['id'],
            code=row['code'],
            name=row['name'],
            legal_name=row.get('legal_name'),
            tax_id=row.get('tax_id'),
            currency=row.get('currency', 'USD'),
            active=bool(row.get('active', 1)),
            parent_id=row.get('parent_id'),
            created_at=row.get('created_at'),
            updated_at=row.get('updated_at')
        )

    def __repr__(self):
        """String representation of Company."""
        return f"<Company {self.code}: {self.name}>"
