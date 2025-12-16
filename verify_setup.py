#!/usr/bin/env python3
"""
OpenERP Setup Verification Script

This script verifies that OpenERP is correctly installed and all
dependencies are available.
"""

import sys
import os

def print_status(message, success=True):
    """Print status message with color."""
    symbol = "✓" if success else "✗"
    color = "\033[92m" if success else "\033[91m"
    reset = "\033[0m"
    print(f"{color}{symbol}{reset} {message}")

def main():
    print("=" * 60)
    print("OpenERP Setup Verification")
    print("=" * 60)

    errors = []

    # Check Python version
    print("\n1. Checking Python version...")
    py_version = sys.version_info
    if py_version.major >= 3 and py_version.minor >= 8:
        print_status(f"Python {py_version.major}.{py_version.minor}.{py_version.micro}", True)
    else:
        print_status(f"Python {py_version.major}.{py_version.minor}.{py_version.micro} (3.8+ required)", False)
        errors.append("Python version too old")

    # Check dependencies
    print("\n2. Checking dependencies...")

    try:
        import RestrictedPython
        version = getattr(RestrictedPython, '__version__', 'installed')
        print_status(f"RestrictedPython {version}", True)
    except ImportError as e:
        print_status("RestrictedPython not found", False)
        errors.append("RestrictedPython missing")

    try:
        import pytz
        print_status(f"pytz {pytz.__version__}", True)
    except ImportError:
        print_status("pytz not found", False)
        errors.append("pytz missing")

    # Check OpenERP imports
    print("\n3. Checking OpenERP modules...")

    try:
        from openerp import Database, Company
        print_status("openerp.Database", True)
        print_status("openerp.Company", True)
    except ImportError as e:
        print_status(f"OpenERP import failed: {e}", False)
        errors.append("OpenERP import failed")
        return

    try:
        from openerp.core.crud import CRUDManager
        print_status("openerp.core.crud.CRUDManager", True)
    except ImportError as e:
        print_status(f"CRUDManager import failed: {e}", False)
        errors.append("CRUDManager import failed")

    try:
        from openerp.core.executor import CodeExecutor
        print_status("openerp.core.executor.CodeExecutor", True)
    except ImportError as e:
        print_status(f"CodeExecutor import failed: {e}", False)
        errors.append("CodeExecutor import failed")

    try:
        from openerp.core.triggers import TriggerManager
        print_status("openerp.core.triggers.TriggerManager", True)
    except ImportError as e:
        print_status(f"TriggerManager import failed: {e}", False)
        errors.append("TriggerManager import failed")

    # Quick functional test
    print("\n4. Running functional test...")

    try:
        # Create test database
        test_db = "test_verification.db"
        if os.path.exists(test_db):
            os.remove(test_db)

        db = Database(test_db)
        print_status("Database creation", True)

        # Create company
        company = Company.create(db, "TestCompany")
        print_status(f"Company creation: {company.name}", True)

        # Create table
        db.create_table("TestTable", {
            "name": "TEXT NOT NULL",
            "value": "TEXT"
        }, company_name="TestCompany")
        print_status("Table creation: TestCompany$TestTable", True)

        # Add translation
        db.set_table_translation("TestCompany$TestTable", "es", "TablaDeTest")
        translation = db.get_table_translation("TestCompany$TestTable", "es")
        if translation == "TablaDeTest":
            print_status(f"Translation system: {translation}", True)
        else:
            print_status("Translation system failed", False)
            errors.append("Translation failed")

        # CRUD operations
        crud = CRUDManager(db)
        result = crud.insert("TestCompany$TestTable", {
            "name": "Test Item",
            "value": "Test Value"
        })
        if result.get("success"):
            print_status("Insert operation", True)
        else:
            print_status("Insert operation failed", False)
            errors.append("Insert failed")

        query_result = crud.get_all("TestCompany$TestTable")
        if query_result.get("success") and query_result.get("count") == 1:
            records = query_result.get("records", [])
            if records and records[0]["name"] == "Test Item":
                print_status(f"Query operation: found {len(records)} record(s)", True)
            else:
                print_status("Query operation: data mismatch", False)
                errors.append("Query data mismatch")
        else:
            print_status("Query operation failed", False)
            errors.append("Query failed")

        # Cleanup
        os.remove(test_db)
        print_status("Cleanup", True)

    except Exception as e:
        print_status(f"Functional test failed: {e}", False)
        errors.append(f"Functional test error: {e}")
        if os.path.exists(test_db):
            os.remove(test_db)

    # Summary
    print("\n" + "=" * 60)
    if not errors:
        print("\033[92m✓ All checks passed! OpenERP is ready to use.\033[0m")
        print("\nNext steps:")
        print("  1. Run examples: python3 examples/translation_demo.py")
        print("  2. Read QUICKSTART.md for detailed instructions")
        print("  3. Check README.md for API documentation")
    else:
        print("\033[91m✗ Setup verification failed:\033[0m")
        for error in errors:
            print(f"  - {error}")
        print("\nPlease fix the errors above and run this script again.")
        print("Install missing dependencies with: pip install -r requirements.txt")
    print("=" * 60)

    return len(errors) == 0

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
