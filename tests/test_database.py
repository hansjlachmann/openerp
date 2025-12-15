"""Tests for database module"""

import pytest
from openerp.core.database import Database


class TestDatabase:
    """Test cases for Database class."""

    def test_create_database(self):
        """Test database initialization."""
        db = Database(":memory:")
        assert db.conn is not None

        # Check metadata tables exist
        cursor = db.conn.cursor()
        cursor.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name='__table_metadata'"
        )
        assert cursor.fetchone() is not None

    def test_create_table(self):
        """Test table creation."""
        db = Database(":memory:")

        result = db.create_table(
            'customers',
            {
                'name': 'TEXT NOT NULL',
                'email': 'TEXT UNIQUE',
                'balance': 'REAL DEFAULT 0'
            }
        )

        assert result is True

        # Verify table exists
        cursor = db.conn.cursor()
        cursor.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name='customers'"
        )
        assert cursor.fetchone() is not None

    def test_create_table_with_trigger(self):
        """Test table creation with trigger."""
        db = Database(":memory:")

        result = db.create_table(
            'orders',
            {'amount': 'REAL', 'status': 'TEXT'},
            on_insert="record['status'] = 'pending'"
        )

        assert result is True

        # Verify trigger is stored
        metadata = db.get_table_metadata('orders')
        assert metadata is not None
        assert metadata['on_insert_trigger'] is not None

    def test_list_tables(self):
        """Test listing tables."""
        db = Database(":memory:")

        db.create_table('table1', {'field1': 'TEXT'})
        db.create_table('table2', {'field2': 'INTEGER'})

        tables = db.list_tables()
        assert 'table1' in tables
        assert 'table2' in tables

    def test_drop_table(self):
        """Test dropping a table."""
        db = Database(":memory:")

        db.create_table('temp_table', {'data': 'TEXT'})
        assert 'temp_table' in db.list_tables()

        db.drop_table('temp_table')
        assert 'temp_table' not in db.list_tables()

    def test_duplicate_table_error(self):
        """Test that creating duplicate table raises error."""
        db = Database(":memory:")

        db.create_table('users', {'name': 'TEXT'})

        with pytest.raises(ValueError, match="already exists"):
            db.create_table('users', {'email': 'TEXT'})

    def test_context_manager(self):
        """Test database context manager."""
        with Database(":memory:") as db:
            db.create_table('test', {'data': 'TEXT'})
            assert 'test' in db.list_tables()
