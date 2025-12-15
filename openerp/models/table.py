"""Table definition and management"""

from typing import Dict, Optional, Any
from openerp.core.database import Database


class TableDefinition:
    """
    Represents a table definition in the ERP system.

    This class provides a high-level interface for working with
    dynamically created tables.
    """

    def __init__(
        self,
        name: str,
        fields: Dict[str, str],
        company_id: Optional[int] = None,
        on_insert: Optional[str] = None,
        on_update: Optional[str] = None,
        on_delete: Optional[str] = None
    ):
        """
        Initialize a table definition.

        Args:
            name: Table name
            fields: Dictionary of field_name: field_type
            company_id: Optional company ID for multi-tenancy
            on_insert: Python code for ON_INSERT trigger
            on_update: Python code for ON_UPDATE trigger
            on_delete: Python code for ON_DELETE trigger
        """
        self.name = name
        self.fields = fields
        self.company_id = company_id
        self.on_insert = on_insert
        self.on_update = on_update
        self.on_delete = on_delete

    def create(self, db: Database) -> bool:
        """
        Create the table in the database.

        Args:
            db: Database instance

        Returns:
            True if successful
        """
        return db.create_table(
            self.name,
            self.fields,
            self.company_id,
            self.on_insert,
            self.on_update,
            self.on_delete
        )

    @classmethod
    def from_metadata(cls, db: Database, table_name: str) -> Optional["TableDefinition"]:
        """
        Load a table definition from database metadata.

        Args:
            db: Database instance
            table_name: Name of the table

        Returns:
            TableDefinition instance or None
        """
        metadata = db.get_table_metadata(table_name)
        if not metadata:
            return None

        import json
        fields = json.loads(metadata['schema_definition'])

        return cls(
            name=table_name,
            fields=fields,
            company_id=metadata.get('company_id'),
            on_insert=metadata.get('on_insert_trigger'),
            on_update=metadata.get('on_update_trigger'),
            on_delete=metadata.get('on_delete_trigger')
        )

    def to_dict(self) -> Dict[str, Any]:
        """Convert table definition to dictionary."""
        return {
            'name': self.name,
            'fields': self.fields,
            'company_id': self.company_id,
            'on_insert': self.on_insert,
            'on_update': self.on_update,
            'on_delete': self.on_delete
        }

    def __repr__(self):
        """String representation of TableDefinition."""
        field_count = len(self.fields)
        return f"<TableDefinition {self.name} ({field_count} fields)>"
