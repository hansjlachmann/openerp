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

    def insert(
        self,
        table_name: str,
        record: Dict[str, Any],
        company_id: Optional[int] = None
    ) -> Dict[str, Any]:
        """
        Insert a new record with ON_INSERT trigger execution.

        Args:
            table_name: Name of the table
            record: Dictionary of field_name: value
            company_id: Optional company ID

        Returns:
            Dictionary with insert result and the inserted record

        Example:
            result = crud.insert('customers', {
                'name': 'John Doe',
                'email': 'john@example.com'
            })
            print(result['id'])  # Auto-generated ID
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

        # Add company_id if specified
        if company_id is not None:
            modified_record['company_id'] = company_id

        # Build INSERT statement
        fields = list(modified_record.keys())
        placeholders = ', '.join(['?' for _ in fields])
        field_names = ', '.join(fields)

        sql = f"INSERT INTO {table_name} ({field_names}) VALUES ({placeholders})"
        values = tuple(modified_record[f] for f in fields)

        try:
            cursor = self.db.execute(sql, values)
            inserted_id = cursor.lastrowid

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
            table_name: Name of the table
            record_id: ID of the record to update
            updates: Dictionary of fields to update

        Returns:
            Dictionary with update result

        Example:
            crud.update('customers', 1, {'email': 'newemail@example.com'})
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

        # Build UPDATE statement
        set_clause = ', '.join([f"{field} = ?" for field in updates.keys()])
        set_clause += ", updated_at = ?"

        sql = f"UPDATE {table_name} SET {set_clause} WHERE id = ?"
        values = tuple(modified_record[f] for f in updates.keys())
        values += (modified_record['updated_at'], record_id)

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
            table_name: Name of the table
            record_id: ID of the record to delete

        Returns:
            Dictionary with delete result

        Example:
            crud.delete('customers', 1)
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
        sql = f"DELETE FROM {table_name} WHERE id = ?"

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
            table_name: Name of the table
            record_id: ID of the record

        Returns:
            Dictionary with the record or error
        """
        cursor = self.db.conn.cursor()
        cursor.execute(f"SELECT * FROM {table_name} WHERE id = ?", (record_id,))
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
        company_id: Optional[int] = None,
        limit: Optional[int] = None,
        offset: int = 0
    ) -> Dict[str, Any]:
        """
        Retrieve all records from a table.

        Args:
            table_name: Name of the table
            company_id: Optional company ID filter
            limit: Maximum number of records to return
            offset: Number of records to skip

        Returns:
            Dictionary with list of records
        """
        cursor = self.db.conn.cursor()

        sql = f"SELECT * FROM {table_name}"
        params = []

        if company_id is not None:
            sql += " WHERE company_id = ?"
            params.append(company_id)

        sql += f" LIMIT {limit or -1} OFFSET {offset}"

        cursor.execute(sql, tuple(params))
        rows = cursor.fetchall()

        return {
            'success': True,
            'records': [dict(row) for row in rows],
            'count': len(rows)
        }

    def search(
        self,
        table_name: str,
        conditions: Dict[str, Any],
        company_id: Optional[int] = None
    ) -> Dict[str, Any]:
        """
        Search for records matching conditions.

        Args:
            table_name: Name of the table
            conditions: Dictionary of field: value conditions
            company_id: Optional company ID filter

        Returns:
            Dictionary with matching records

        Example:
            crud.search('customers', {'name': 'John Doe'})
        """
        cursor = self.db.conn.cursor()

        where_clauses = [f"{field} = ?" for field in conditions.keys()]
        params = list(conditions.values())

        if company_id is not None:
            where_clauses.append("company_id = ?")
            params.append(company_id)

        sql = f"SELECT * FROM {table_name} WHERE {' AND '.join(where_clauses)}"

        cursor.execute(sql, tuple(params))
        rows = cursor.fetchall()

        return {
            'success': True,
            'records': [dict(row) for row in rows],
            'count': len(rows)
        }
