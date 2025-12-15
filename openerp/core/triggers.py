"""Trigger management system for database operations"""

from typing import Any, Dict, Optional, Callable
from enum import Enum
from openerp.core.executor import CodeExecutor


class TriggerType(Enum):
    """Types of triggers supported."""
    ON_INSERT = "on_insert"
    ON_UPDATE = "on_update"
    ON_DELETE = "on_delete"


class TriggerManager:
    """
    Manages triggers for database operations.

    Triggers are Python code snippets that execute automatically
    when certain database operations occur.
    """

    def __init__(self, executor: Optional[CodeExecutor] = None):
        """
        Initialize trigger manager.

        Args:
            executor: CodeExecutor instance for running trigger code
        """
        self.executor = executor or CodeExecutor()
        self._triggers: Dict[str, Dict[TriggerType, str]] = {}

    def register_trigger(
        self,
        table_name: str,
        trigger_type: TriggerType,
        code: str
    ):
        """
        Register a trigger for a table.

        Args:
            table_name: Name of the table
            trigger_type: Type of trigger (ON_INSERT, ON_UPDATE, ON_DELETE)
            code: Python code to execute

        Example:
            manager.register_trigger(
                'customers',
                TriggerType.ON_INSERT,
                "record['created_at'] = datetime.now()"
            )
        """
        if table_name not in self._triggers:
            self._triggers[table_name] = {}

        # Validate the code before storing
        validation = self.executor.validate_code(code)
        if not validation['valid']:
            raise ValueError(f"Invalid trigger code: {validation['errors']}")

        self._triggers[table_name][trigger_type] = code

    def execute_trigger(
        self,
        table_name: str,
        trigger_type: TriggerType,
        record: Dict[str, Any],
        old_record: Optional[Dict[str, Any]] = None
    ) -> Dict[str, Any]:
        """
        Execute a trigger for a specific operation.

        Args:
            table_name: Name of the table
            trigger_type: Type of trigger to execute
            record: The record being modified (NEW record)
            old_record: The previous record state (for updates/deletes)

        Returns:
            Dictionary containing execution result and modified record

        The trigger code has access to:
        - record: The new/current record (mutable for ON_INSERT/ON_UPDATE)
        - old_record: The previous record (for ON_UPDATE/ON_DELETE)
        - datetime: datetime module
        """
        if table_name not in self._triggers:
            return {'success': True, 'record': record, 'executed': False}

        if trigger_type not in self._triggers[table_name]:
            return {'success': True, 'record': record, 'executed': False}

        code = self._triggers[table_name][trigger_type]

        # Build execution context
        context = {
            'record': record.copy(),
            'old_record': old_record.copy() if old_record else None,
        }

        # Execute the trigger
        result = self.executor.execute(code, context)

        if not result['success']:
            return {
                'success': False,
                'errors': result['errors'],
                'record': record,
                'executed': True
            }

        # Extract the modified record
        modified_record = result['context'].get('record', record)

        return {
            'success': True,
            'record': modified_record,
            'errors': [],
            'executed': True
        }

    def has_trigger(self, table_name: str, trigger_type: TriggerType) -> bool:
        """Check if a table has a specific trigger."""
        return (
            table_name in self._triggers and
            trigger_type in self._triggers[table_name]
        )

    def get_trigger(self, table_name: str, trigger_type: TriggerType) -> Optional[str]:
        """Get the code for a specific trigger."""
        if not self.has_trigger(table_name, trigger_type):
            return None
        return self._triggers[table_name][trigger_type]

    def remove_trigger(self, table_name: str, trigger_type: TriggerType):
        """Remove a trigger from a table."""
        if table_name in self._triggers:
            self._triggers[table_name].pop(trigger_type, None)

    def remove_all_triggers(self, table_name: str):
        """Remove all triggers for a table."""
        self._triggers.pop(table_name, None)
