#!/usr/bin/env python3
"""
Translation Demo - Demonstrating Multi-Language Support

This example shows how to use the JSON-based translation system
for table names and field names stored in metadata tables.
"""

import os
import sys

# Add parent directory to path
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))

from openerp import Database
from openerp.models.company import Company


def main():
    print("=" * 60)
    print("OpenERP Translation System Demo")
    print("=" * 60)

    # Create database
    db_path = "translation_demo.db"
    if os.path.exists(db_path):
        os.remove(db_path)

    db = Database(db_path)
    print(f"\n✓ Created database: {db_path}")

    # Step 1: Create a company
    print("\n" + "=" * 60)
    print("Step 1: Creating Company")
    print("=" * 60)
    company = Company.create(db, "ACME")
    print(f"✓ Created company: {company.name}")

    # Step 2: Create a company-specific table
    print("\n" + "=" * 60)
    print("Step 2: Creating Customers Table")
    print("=" * 60)

    customers_schema = {
        "Name": "TEXT NOT NULL",
        "Email": "TEXT NOT NULL",
        "Phone": "TEXT",
        "Address": "TEXT"
    }

    db.create_table("Customers", customers_schema, company_name="ACME")
    print(f"✓ Created table: ACME$Customers")
    print(f"  Fields: Name, Email, Phone, Address")

    # Step 3: Add table translations
    print("\n" + "=" * 60)
    print("Step 3: Adding Table Name Translations")
    print("=" * 60)

    table_name = "ACME$Customers"

    # Add translations for different languages
    translations = {
        "en": "Customers",
        "es": "Clientes",
        "nl": "Klanten",
        "fr": "Clients",
        "de": "Kunden"
    }

    for lang_code, translation in translations.items():
        db.set_table_translation(table_name, lang_code, translation)
        print(f"✓ Added {lang_code.upper()}: {translation}")

    # Step 4: Add field translations
    print("\n" + "=" * 60)
    print("Step 4: Adding Field Name Translations")
    print("=" * 60)

    field_translations = {
        "Name": {
            "en": "Name",
            "es": "Nombre",
            "nl": "Naam",
            "fr": "Nom",
            "de": "Name"
        },
        "Email": {
            "en": "Email",
            "es": "Correo electrónico",
            "nl": "E-mail",
            "fr": "Courriel",
            "de": "E-Mail"
        },
        "Phone": {
            "en": "Phone",
            "es": "Teléfono",
            "nl": "Telefoon",
            "fr": "Téléphone",
            "de": "Telefon"
        },
        "Address": {
            "en": "Address",
            "es": "Dirección",
            "nl": "Adres",
            "fr": "Adresse",
            "de": "Adresse"
        }
    }

    for field_name, translations in field_translations.items():
        print(f"\n  Field: {field_name}")
        for lang_code, translation in translations.items():
            db.set_field_translation(table_name, field_name, lang_code, translation)
            print(f"    ✓ {lang_code.upper()}: {translation}")

    # Step 5: Retrieve translations
    print("\n" + "=" * 60)
    print("Step 5: Retrieving Translations")
    print("=" * 60)

    # Get all table translations
    print(f"\nAll table translations for '{table_name}':")
    all_table_trans = db.get_table_translations(table_name)
    for lang, trans in all_table_trans.items():
        print(f"  {lang.upper()}: {trans}")

    # Get specific table translation
    print(f"\nTable name in Spanish:")
    spanish_table = db.get_table_translation(table_name, "es")
    print(f"  {spanish_table}")

    # Get all field translations for a specific field
    print(f"\nAll translations for 'Email' field:")
    email_translations = db.get_field_translations(table_name, "Email")
    for lang, trans in email_translations.items():
        print(f"  {lang.upper()}: {trans}")

    # Step 6: Simulate a multi-language form
    print("\n" + "=" * 60)
    print("Step 6: Multi-Language Form Simulation")
    print("=" * 60)

    languages = ["en", "es", "nl", "fr", "de"]

    for lang in languages:
        print(f"\n--- Form in {lang.upper()} ---")
        form_title = db.get_table_translation(table_name, lang)
        print(f"Form: {form_title}")
        print("-" * 30)

        for field_name in ["Name", "Email", "Phone", "Address"]:
            field_label = db.get_field_translation(table_name, field_name, lang)
            print(f"  {field_label}: [____________]")

    # Step 7: Test fallback behavior
    print("\n" + "=" * 60)
    print("Step 7: Testing Fallback Behavior")
    print("=" * 60)

    # Try to get translation for unsupported language
    print("\nRequesting translation for unsupported language (Japanese):")
    japanese_table = db.get_table_translation(table_name, "ja", fallback="お客様")
    print(f"  Result with fallback: {japanese_table}")

    japanese_table_no_fallback = db.get_table_translation(table_name, "ja")
    print(f"  Result without fallback: {japanese_table_no_fallback}")

    # Step 8: Create another table with translations
    print("\n" + "=" * 60)
    print("Step 8: Creating Products Table with Translations")
    print("=" * 60)

    products_schema = {
        "Code": "TEXT NOT NULL",
        "Description": "TEXT NOT NULL",
        "Price": "REAL NOT NULL",
        "Stock": "INTEGER"
    }

    db.create_table("Products", products_schema, company_name="ACME")
    products_table = "ACME$Products"
    print(f"✓ Created table: {products_table}")

    # Add table translations
    product_table_trans = {
        "en": "Products",
        "es": "Productos",
        "nl": "Producten",
        "fr": "Produits",
        "de": "Produkte"
    }

    for lang_code, translation in product_table_trans.items():
        db.set_table_translation(products_table, lang_code, translation)

    print("\n✓ Added table translations")

    # Add field translations
    product_field_trans = {
        "Code": {"en": "Code", "es": "Código", "nl": "Code", "fr": "Code", "de": "Code"},
        "Description": {"en": "Description", "es": "Descripción", "nl": "Beschrijving", "fr": "Description", "de": "Beschreibung"},
        "Price": {"en": "Price", "es": "Precio", "nl": "Prijs", "fr": "Prix", "de": "Preis"},
        "Stock": {"en": "Stock", "es": "Existencias", "nl": "Voorraad", "fr": "Stock", "de": "Bestand"}
    }

    for field_name, translations in product_field_trans.items():
        for lang_code, translation in translations.items():
            db.set_field_translation(products_table, field_name, lang_code, translation)

    print("✓ Added field translations")

    # Display both forms side by side in Spanish
    print("\n" + "=" * 60)
    print("Spanish Forms Side by Side")
    print("=" * 60)

    customers_es = db.get_table_translation(table_name, "es")
    products_es = db.get_table_translation(products_table, "es")

    print(f"\n{customers_es:30} | {products_es}")
    print("-" * 30 + " | " + "-" * 30)

    # Get max number of fields
    customer_fields = ["Name", "Email", "Phone", "Address"]
    product_fields = ["Code", "Description", "Price", "Stock"]

    for i in range(max(len(customer_fields), len(product_fields))):
        cust_field = customer_fields[i] if i < len(customer_fields) else ""
        prod_field = product_fields[i] if i < len(product_fields) else ""

        cust_label = db.get_field_translation(table_name, cust_field, "es") if cust_field else ""
        prod_label = db.get_field_translation(products_table, prod_field, "es") if prod_field else ""

        cust_line = f"{cust_label}: [________]" if cust_label else ""
        prod_line = f"{prod_label}: [________]" if prod_label else ""

        print(f"{cust_line:30} | {prod_line}")

    print("\n" + "=" * 60)
    print("Demo completed successfully!")
    print("=" * 60)
    print(f"\n✓ Database file: {db_path}")
    print("✓ Translation storage: JSON in metadata tables")
    print("✓ Supported languages: EN, ES, NL, FR, DE")
    print("\nThe translations are stored as JSON in:")
    print("  - __table_metadata.translations (for table names)")
    print("  - __field_metadata.translations (for field names)")


if __name__ == "__main__":
    main()
