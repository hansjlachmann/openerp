"""CRUD operations with trigger support"""

from typing import Any, Dict, List, Optional
from datetime import datetime
from openerp.core.database import Database
from openerp.core.triggers import TriggerManager, TriggerType


class CRUDManager:
    """
    Manages CRUD operations with automatic trigger execution.

    Provides a high-level interface for data manipulation with
    built-in trigger support and validation.
    """

    def __init__(self, database: Database, trigger_manager: Optional[TriggerManager] = None):
        """
        Initialize CRUD manager.

        Args:
            database: Database instance
            trigger_manager: TriggerManager instance
        """
        self.db = database
        self.trigger_manager = trigger_manager or TriggerManager()
        self._load_triggers_from_metadata()

    def _load_triggers_from_metadata(self):
        """Load triggers from table metadata into trigger manager."""
        tables = self.db.list_tables()
        for table_name in tables:
            metadata = self.db.get_table_metadata(table_name)
            if metadata:
                if metadata['on_insert_trigger']:
                    self.trigger_manager.register_trigger(
                        table_name,
                        TriggerType.ON_INSERT,
                        metadata['on_insert_trigger']
                    )
                if metadata['on_update_trigger']:
                    self.trigger_manager.register_trigger(
                        table_name,
                        TriggerType.ON_UPDATE,
                        metadata['on_update_trigger']
                    )
                if metadata['on_delete_trigger']:
                    self.trigger_manager.register_trigger(
                        table_name,
                        TriggerType.ON_DELETE,
                        metadata['on_delete_trigger']
                    )

    def reload_triggers(self):
        """
        Reload all triggers from database metadata.

        Useful when new tables with triggers are created after
        the CRUDManager was initialized.
        """
        self._load_triggers_from_metadata()

    def insert(
        self,
        table_name: str,
        record: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Insert a new record with ON_INSERT trigger execution.

        Args:
            table_name: Full table name (may include company prefix like "ACME$Customers")
            record: Dictionary of field_name: value

        Returns:
            Dictionary with insert result and the inserted record

        Example:
            # Company-specific table
            result = crud.insert('ACME$Customers', {
                'name': 'John Doe',
                'email': 'john@example.com'
            })

            # Global table
            result = crud.insert('SystemSettings', {
                'key': 'theme',
                'value': 'dark'
            })
        """
        # Execute ON_INSERT trigger
        trigger_result = self.trigger_manager.execute_trigger(
            table_name,
            TriggerType.ON_INSERT,
            record
        )

        if not trigger_result['success']:
            return {
                'success': False,
                'errors': trigger_result['errors'],
                'id': None
            }

        # Use the modified record from trigger
        modified_record = trigger_result['record']

        # Build INSERT statement
        fields = list(modified_record.keys())
        placeholders = ', '.join(['?' for _ in fields])
        field_names = ', '.join(fields)

        # Use quotes around table name to handle $ character
        sql = f'INSERT INTO "{table_name}" ({field_names}) VALUES ({placeholders})'
        values = tuple(modified_record[f] for f in fields)

        try:
            cursor = self.db.execute(sql, values)
            inserted_id = cursor.lastrowid

            # Add the generated id to the record
            modified_record['id'] = inserted_id

            return {
                'success': True,
                'id': inserted_id,
                'record': modified_record,
                'trigger_executed': trigger_result['executed']
            }

        except Exception as e:
            return {
                'success': False,
                'errors': [str(e)],
                'id': None
            }

    def update(
        self,
        table_name: str,
        record_id: int,
        updates: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Update a record with ON_UPDATE trigger execution.

        Args:
            table_name: Full table name (may include company prefix)
            record_id: ID of the record to update
            updates: Dictionary of fields to update

        Returns:
            Dictionary with update result

        Example:
            crud.update('ACME$Customers', 1, {'email': 'newemail@example.com'})
        """
        # Fetch the old record
        old_record = self.get_by_id(table_name, record_id)
        if not old_record['success']:
            return old_record

        # Merge updates with old record
        new_record = old_record['record'].copy()
        new_record.update(updates)

        # Update the updated_at timestamp
        new_record['updated_at'] = datetime.now().isoformat()

        # Execute ON_UPDATE trigger
        trigger_result = self.trigger_manager.execute_trigger(
            table_name,
            TriggerType.ON_UPDATE,
            new_record,
            old_record['record']
        )

        if not trigger_result['success']:
            return {
                'success': False,
                'errors': trigger_result['errors']
            }

        # Use the modified record from trigger
        modified_record = trigger_result['record']

        # Find all fields that changed (excluding id and created_at)
        fields_to_update = {k: v for k, v in modified_record.items()
                          if k not in ('id', 'created_at') and
                          (k not in old_record['record'] or old_record['record'][k] != v)}

        # Build UPDATE statement with all modified fields
        set_clause = ', '.join([f"{field} = ?" for field in fields_to_update.keys()])

        # Use quotes around table name to handle $ character
        sql = f'UPDATE "{table_name}" SET {set_clause} WHERE id = ?'
        values = tuple(fields_to_update[f] for f in fields_to_update.keys())
        values += (record_id,)

        try:
            self.db.execute(sql, values)
            return {
                'success': True,
                'record': modified_record,
                'trigger_executed': trigger_result['executed']
            }

        except Exception as e:
            return {
                'success': False,
                'errors': [str(e)]
            }

    def delete(self, table_name: str, record_id: int) -> Dict[str, Any]:
        """
        Delete a record with ON_DELETE trigger execution.

        Args:
            table_name: Full table name (may include company prefix)
            record_id: ID of the record to delete

        Returns:
            Dictionary with delete result

        Example:
            crud.delete('ACME$Customers', 1)
        """
        # Fetch the record before deletion
        old_record = self.get_by_id(table_name, record_id)
        if not old_record['success']:
            return old_record

        # Execute ON_DELETE trigger
        trigger_result = self.trigger_manager.execute_trigger(
            table_name,
            TriggerType.ON_DELETE,
            old_record['record'],
            old_record['record']
        )

        if not trigger_result['success']:
            return {
                'success': False,
                'errors': trigger_result['errors']
            }

        # Perform the deletion
        # Use quotes around table name to handle $ character
        sql = f'DELETE FROM "{table_name}" WHERE id = ?'

        try:
            self.db.execute(sql, (record_id,))
            return {
                'success': True,
                'deleted_record': old_record['record'],
                'trigger_executed': trigger_result['executed']
            }

        except Exception as e:
            return {
                'success': False,
                'errors': [str(e)]
            }

    def get_by_id(self, table_name: str, record_id: int) -> Dict[str, Any]:
        """
        Retrieve a single record by ID.

        Args:
            table_name: Full table name (may include company prefix)
            record_id: ID of the record

        Returns:
            Dictionary with the record or error
        """
        cursor = self.db.conn.cursor()
        # Use quotes around table name to handle $ character
        cursor.execute(f'SELECT * FROM "{table_name}" WHERE id = ?', (record_id,))
        row = cursor.fetchone()

        if row:
            return {
                'success': True,
                'record': dict(row)
            }
        else:
            return {
                'success': False,
                'errors': [f"Record with id {record_id} not found"]
            }

    def get_all(
        self,
        table_name: str,
        limit: Optional[int] = None,
        offset: int = 0
    ) -> Dict[str, Any]:
        """
        Retrieve all records from a table.

        Args:
            table_name: Full table name (may include company prefix)
            limit: Maximum number of records to return
            offset: Number of records to skip

        Returns:
            Dictionary with list of records
        """
        cursor = self.db.conn.cursor()

        # Use quotes around table name to handle $ character
        sql = f'SELECT * FROM "{table_name}"'
        sql += f" LIMIT {limit or -1} OFFSET {offset}"

        cursor.execute(sql)
        rows = cursor.fetchall()

        return {
            'success': True,
            'records': [dict(row) for row in rows],
            'count': len(rows)
        }

    def search(
        self,
        table_name: str,
        conditions: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Search for records matching conditions.

        Args:
            table_name: Full table name (may include company prefix)
            conditions: Dictionary of field: value conditions

        Returns:
            Dictionary with matching records

        Example:
            crud.search('ACME$Customers', {'name': 'John Doe'})
        """
        cursor = self.db.conn.cursor()

        where_clauses = [f"{field} = ?" for field in conditions.keys()]
        params = list(conditions.values())

        # Use quotes around table name to handle $ character
        sql = f'SELECT * FROM "{table_name}" WHERE {" AND ".join(where_clauses)}'

        cursor.execute(sql, tuple(params))
        rows = cursor.fetchall()

        return {
            'success': True,
            'records': [dict(row) for row in rows],
            'count': len(rows)
        }
