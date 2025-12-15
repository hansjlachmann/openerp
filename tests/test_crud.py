"""Tests for CRUD operations"""

import pytest
from openerp.core.database import Database
from openerp.core.crud import CRUDManager


class TestCRUD:
    """Test cases for CRUDManager class."""

    def setup_method(self):
        """Set up test database."""
        self.db = Database(":memory:")
        self.crud = CRUDManager(self.db)

        # Create a test table
        self.db.create_table(
            'products',
            {
                'name': 'TEXT NOT NULL',
                'price': 'REAL',
                'stock': 'INTEGER DEFAULT 0'
            }
        )

    def test_insert(self):
        """Test inserting a record."""
        result = self.crud.insert('products', {
            'name': 'Widget',
            'price': 9.99,
            'stock': 100
        })

        assert result['success'] is True
        assert result['id'] is not None
        assert result['record']['name'] == 'Widget'

    def test_insert_with_trigger(self):
        """Test insert with ON_INSERT trigger."""
        # Create table with trigger
        self.db.create_table(
            'orders',
            {'amount': 'REAL', 'status': 'TEXT'},
            on_insert="record['status'] = 'pending'"
        )

        # Reload triggers
        self.crud._load_triggers_from_metadata()

        result = self.crud.insert('orders', {'amount': 100.0})

        assert result['success'] is True
        assert result['trigger_executed'] is True

        # Verify the trigger modified the record
        fetched = self.crud.get_by_id('orders', result['id'])
        assert fetched['record']['status'] == 'pending'

    def test_get_by_id(self):
        """Test retrieving a record by ID."""
        insert_result = self.crud.insert('products', {
            'name': 'Gadget',
            'price': 19.99
        })

        result = self.crud.get_by_id('products', insert_result['id'])

        assert result['success'] is True
        assert result['record']['name'] == 'Gadget'
        assert result['record']['price'] == 19.99

    def test_update(self):
        """Test updating a record."""
        insert_result = self.crud.insert('products', {
            'name': 'Item',
            'price': 5.0,
            'stock': 50
        })

        update_result = self.crud.update(
            'products',
            insert_result['id'],
            {'price': 6.0, 'stock': 75}
        )

        assert update_result['success'] is True

        # Verify update
        fetched = self.crud.get_by_id('products', insert_result['id'])
        assert fetched['record']['price'] == 6.0
        assert fetched['record']['stock'] == 75

    def test_delete(self):
        """Test deleting a record."""
        insert_result = self.crud.insert('products', {
            'name': 'Temp Item',
            'price': 1.0
        })

        delete_result = self.crud.delete('products', insert_result['id'])

        assert delete_result['success'] is True

        # Verify deletion
        fetched = self.crud.get_by_id('products', insert_result['id'])
        assert fetched['success'] is False

    def test_get_all(self):
        """Test retrieving all records."""
        self.crud.insert('products', {'name': 'Product 1', 'price': 10.0})
        self.crud.insert('products', {'name': 'Product 2', 'price': 20.0})
        self.crud.insert('products', {'name': 'Product 3', 'price': 30.0})

        result = self.crud.get_all('products')

        assert result['success'] is True
        assert result['count'] == 3

    def test_get_all_with_limit(self):
        """Test retrieving records with limit."""
        for i in range(10):
            self.crud.insert('products', {'name': f'Product {i}', 'price': i * 10.0})

        result = self.crud.get_all('products', limit=5)

        assert result['success'] is True
        assert result['count'] == 5

    def test_search(self):
        """Test searching for records."""
        self.crud.insert('products', {'name': 'Widget', 'price': 10.0})
        self.crud.insert('products', {'name': 'Gadget', 'price': 20.0})
        self.crud.insert('products', {'name': 'Widget', 'price': 15.0})

        result = self.crud.search('products', {'name': 'Widget'})

        assert result['success'] is True
        assert result['count'] == 2

    def teardown_method(self):
        """Clean up test database."""
        self.db.close()
