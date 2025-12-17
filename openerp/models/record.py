"""
NAV-style Record base class

Provides a familiar interface for NAV/C/AL developers:
- Record.get(id) - Like NAV's Rec.GET(PrimaryKey)
- record.insert() - Like NAV's Rec.INSERT
- record.modify() - Like NAV's Rec.MODIFY
- record.delete() - Like NAV's Rec.DELETE
- Record.find_set() - Like NAV's Rec.FINDSET

Example usage (NAV-style):
    customer = Customer.get(1)
    customer['balance'] = 5000
    customer.modify()
"""

from typing import Any, Dict, List, Optional, Type, TypeVar
from openerp.core.crud import CRUDManager
from openerp.core.database import Database


T = TypeVar('T', bound='Record')


class Record:
    """
    Base class for NAV-style record operations.

    Similar to NAV's Record type with Get, Insert, Modify, Delete.
    """

    # Subclasses must define these
    TABLE_NAME: str = None
    COMPANY_NAME: str = None

    def __init__(self, db: Database, data: Optional[Dict[str, Any]] = None):
        """
        Initialize a record.

        Args:
            db: Database instance
            data: Record data (optional, for new records)
        """
        self._db = db
        self._crud = CRUDManager(db)
        self._data = data or {}
        self._exists = data is not None and 'id' in data

    @property
    def table_name(self) -> str:
        """Get full table name with company prefix."""
        if self.COMPANY_NAME:
            return f"{self.COMPANY_NAME}${self.TABLE_NAME}"
        return self.TABLE_NAME

    def __getitem__(self, key: str) -> Any:
        """Get field value (like NAV: CustomerName := Rec."Name")."""
        return self._data.get(key)

    def __setitem__(self, key: str, value: Any):
        """Set field value (like NAV: Rec."Name" := 'New Name')."""
        self._data[key] = value

    def get(self, key: str, default: Any = None) -> Any:
        """Get field value with default."""
        return self._data.get(key, default)

    @classmethod
    def get(cls: Type[T], db: Database, *primary_key) -> Optional[T]:
        """
        Get a record by primary key (like NAV's GET).

        Args:
            db: Database instance
            *primary_key: Primary key value(s)

        Returns:
            Record instance or None if not found

        Example:
            customer = Customer.get(db, 1)
            orderline = OrderLine.get(db, order_no, line_no)
        """
        if len(primary_key) == 1:
            # Single primary key (most common case)
            record_id = primary_key[0]
            crud = CRUDManager(db)

            # Get full table name
            if cls.COMPANY_NAME:
                table_name = f"{cls.COMPANY_NAME}${cls.TABLE_NAME}"
            else:
                table_name = cls.TABLE_NAME

            result = crud.get_by_id(table_name, record_id)

            if result['success']:
                return cls(db, result['record'])
            return None
        else:
            # Composite primary key - not yet implemented
            raise NotImplementedError("Composite primary keys not yet supported")

    def insert(self) -> bool:
        """
        Insert record into database (like NAV's INSERT).

        Returns:
            True if successful

        Example:
            customer = Customer(db)
            customer['name'] = 'John Doe'
            customer['email'] = 'john@example.com'
            if customer.insert():
                print(f"Created customer ID: {customer['id']}")
        """
        result = self._crud.insert(self.table_name, self._data)

        if result['success']:
            self._data = result['record']
            self._exists = True
            return True
        return False

    def modify(self) -> bool:
        """
        Update existing record (like NAV's MODIFY).

        Returns:
            True if successful

        Example:
            customer = Customer.get(db, 1)
            customer['balance'] = 5000
            customer.modify()
        """
        if not self._exists or 'id' not in self._data:
            raise ValueError("Cannot modify a record that hasn't been inserted")

        # Get only the fields that should be updated (exclude id, created_at)
        update_fields = {k: v for k, v in self._data.items()
                        if k not in ('id', 'created_at', 'updated_at')}

        result = self._crud.update(self.table_name, self._data['id'], update_fields)

        if result['success']:
            self._data = result['record']
            return True
        return False

    def delete(self) -> bool:
        """
        Delete record from database (like NAV's DELETE).

        Returns:
            True if successful

        Example:
            customer = Customer.get(db, 1)
            customer.delete()
        """
        if not self._exists or 'id' not in self._data:
            raise ValueError("Cannot delete a record that hasn't been inserted")

        result = self._crud.delete(self.table_name, self._data['id'])

        if result['success']:
            self._exists = False
            return True
        return False

    @classmethod
    def find_set(cls: Type[T], db: Database, filters: Optional[Dict[str, Any]] = None) -> List[T]:
        """
        Find multiple records (like NAV's FINDSET).

        Args:
            db: Database instance
            filters: Optional filter criteria

        Returns:
            List of record instances

        Example:
            # Get all customers
            customers = Customer.find_set(db)

            # Get customers with filter
            customers = Customer.find_set(db, {'name': 'John Doe'})
        """
        crud = CRUDManager(db)

        # Get full table name
        if cls.COMPANY_NAME:
            table_name = f"{cls.COMPANY_NAME}${cls.TABLE_NAME}"
        else:
            table_name = cls.TABLE_NAME

        if filters:
            result = crud.search(table_name, filters)
        else:
            result = crud.get_all(table_name)

        if result['success']:
            return [cls(db, record_data) for record_data in result['records']]
        return []

    @classmethod
    def find_first(cls: Type[T], db: Database, filters: Dict[str, Any]) -> Optional[T]:
        """
        Find first matching record (like NAV's FINDFIRST).

        Args:
            db: Database instance
            filters: Filter criteria

        Returns:
            First matching record or None

        Example:
            customer = Customer.find_first(db, {'email': 'john@example.com'})
        """
        records = cls.find_set(db, filters)
        return records[0] if records else None

    def __repr__(self) -> str:
        """String representation."""
        return f"<{self.__class__.__name__} {self._data}>"
