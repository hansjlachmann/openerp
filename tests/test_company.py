"""Tests for company management"""

import pytest
from openerp.core.database import Database
from openerp.models.company import Company


class TestCompany:
    """Test cases for Company class."""

    def setup_method(self):
        """Set up test database."""
        self.db = Database(":memory:")

    def test_create_company(self):
        """Test creating a company."""
        company = Company.create(
            self.db,
            code="ACME",
            name="Acme Corporation",
            tax_id="12-3456789"
        )

        assert company is not None
        assert company.code == "ACME"
        assert company.name == "Acme Corporation"
        assert company.tax_id == "12-3456789"

    def test_company_code_uppercase(self):
        """Test that company code is automatically uppercased."""
        company = Company.create(
            self.db,
            code="test",
            name="Test Company"
        )

        assert company.code == "TEST"

    def test_get_by_id(self):
        """Test retrieving company by ID."""
        created = Company.create(
            self.db,
            code="TEST",
            name="Test Company"
        )

        fetched = Company.get_by_id(self.db, created.id)

        assert fetched is not None
        assert fetched.id == created.id
        assert fetched.name == created.name

    def test_get_by_code(self):
        """Test retrieving company by code."""
        Company.create(
            self.db,
            code="MYCO",
            name="My Company"
        )

        fetched = Company.get_by_code(self.db, "MYCO")

        assert fetched is not None
        assert fetched.code == "MYCO"

    def test_list_all(self):
        """Test listing all companies."""
        Company.create(self.db, code="CO1", name="Company 1")
        Company.create(self.db, code="CO2", name="Company 2")
        Company.create(self.db, code="CO3", name="Company 3")

        companies = Company.list_all(self.db)

        assert len(companies) == 3

    def test_update_company(self):
        """Test updating company fields."""
        company = Company.create(
            self.db,
            code="UPD",
            name="Original Name"
        )

        company.update(self.db, name="Updated Name")

        fetched = Company.get_by_id(self.db, company.id)
        assert fetched.name == "Updated Name"

    def test_deactivate_company(self):
        """Test deactivating a company."""
        company = Company.create(
            self.db,
            code="DEACT",
            name="To Be Deactivated"
        )

        assert company.active is True

        company.deactivate(self.db)

        # Verify deactivation
        all_companies = Company.list_all(self.db, active_only=False)
        inactive = [c for c in all_companies if c.code == "DEACT"][0]
        assert inactive.active is False

    def test_company_to_dict(self):
        """Test converting company to dictionary."""
        company = Company.create(
            self.db,
            code="DICT",
            name="Dictionary Test",
            currency="EUR"
        )

        data = company.to_dict()

        assert data['code'] == "DICT"
        assert data['name'] == "Dictionary Test"
        assert data['currency'] == "EUR"

    def test_parent_company(self):
        """Test creating subsidiary company."""
        parent = Company.create(
            self.db,
            code="PARENT",
            name="Parent Company"
        )

        subsidiary = Company.create(
            self.db,
            code="SUB",
            name="Subsidiary",
            parent_id=parent.id
        )

        assert subsidiary.parent_id == parent.id

    def teardown_method(self):
        """Clean up test database."""
        self.db.close()
