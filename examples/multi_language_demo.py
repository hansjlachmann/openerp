"""
Multi-Language Support Demo

Demonstrates translation of:
- Table names
- Field names
- Form labels
- Buttons
- Messages
- Reports
- Dialogs
"""

from openerp import Database, Company
from openerp.core.crud import CRUDManager
from openerp.core.i18n import (
    Language,
    TranslationManager,
    TranslationContext,
    init_i18n,
    t
)


def main():
    print("=== OpenERP Multi-Language Support Demo ===\n")

    # 1. Initialize Database
    db = Database(':memory:')
    crud = CRUDManager(db)

    # 2. Set up Languages
    print("1. Setting up Languages")
    print("-" * 60)

    # Create languages
    en = Language.create(db, "en", "English", "English", is_default=True)
    es = Language.create(db, "es", "Spanish", "Español")
    nl = Language.create(db, "nl", "Dutch", "Nederlands")
    fr = Language.create(db, "fr", "French", "Français")

    print(f"✓ Created languages:")
    for lang in Language.list_all(db):
        default = " (default)" if lang.is_default else ""
        print(f"  - {lang.code}: {lang.native_name}{default}")

    # 3. Initialize Translation Manager
    print("\n2. Initializing Translation System")
    print("-" * 60)

    tm = init_i18n(db, "en")
    print("✓ Translation system initialized")

    # 4. Add Table Name Translations
    print("\n3. Adding Table Name Translations")
    print("-" * 60)

    table_translations = [
        # English (base)
        {"language_code": "en", "context": "table_name", "key": "Customers", "value": "Customers"},
        {"language_code": "en", "context": "table_name", "key": "Orders", "value": "Orders"},
        {"language_code": "en", "context": "table_name", "key": "Products", "value": "Products"},

        # Spanish
        {"language_code": "es", "context": "table_name", "key": "Customers", "value": "Clientes"},
        {"language_code": "es", "context": "table_name", "key": "Orders", "value": "Pedidos"},
        {"language_code": "es", "context": "table_name", "key": "Products", "value": "Productos"},

        # Dutch
        {"language_code": "nl", "context": "table_name", "key": "Customers", "value": "Klanten"},
        {"language_code": "nl", "context": "table_name", "key": "Orders", "value": "Bestellingen"},
        {"language_code": "nl", "context": "table_name", "key": "Products", "value": "Producten"},

        # French
        {"language_code": "fr", "context": "table_name", "key": "Customers", "value": "Clients"},
        {"language_code": "fr", "context": "table_name", "key": "Orders", "value": "Commandes"},
        {"language_code": "fr", "context": "table_name", "key": "Products", "value": "Produits"},
    ]

    count = tm.bulk_add_translations(table_translations)
    print(f"✓ Added {count} table name translations")

    # 5. Add Field Name Translations
    print("\n4. Adding Field Name Translations")
    print("-" * 60)

    field_translations = [
        # English
        {"language_code": "en", "context": "field_name", "key": "name", "value": "Name"},
        {"language_code": "en", "context": "field_name", "key": "email", "value": "Email"},
        {"language_code": "en", "context": "field_name", "key": "phone", "value": "Phone"},
        {"language_code": "en", "context": "field_name", "key": "address", "value": "Address"},

        # Spanish
        {"language_code": "es", "context": "field_name", "key": "name", "value": "Nombre"},
        {"language_code": "es", "context": "field_name", "key": "email", "value": "Correo electrónico"},
        {"language_code": "es", "context": "field_name", "key": "phone", "value": "Teléfono"},
        {"language_code": "es", "context": "field_name", "key": "address", "value": "Dirección"},

        # Dutch
        {"language_code": "nl", "context": "field_name", "key": "name", "value": "Naam"},
        {"language_code": "nl", "context": "field_name", "key": "email", "value": "E-mail"},
        {"language_code": "nl", "context": "field_name", "key": "phone", "value": "Telefoon"},
        {"language_code": "nl", "context": "field_name", "key": "address", "value": "Adres"},

        # French
        {"language_code": "fr", "context": "field_name", "key": "name", "value": "Nom"},
        {"language_code": "fr", "context": "field_name", "key": "email", "value": "E-mail"},
        {"language_code": "fr", "context": "field_name", "key": "phone", "value": "Téléphone"},
        {"language_code": "fr", "context": "field_name", "key": "address", "value": "Adresse"},
    ]

    count = tm.bulk_add_translations(field_translations)
    print(f"✓ Added {count} field name translations")

    # 6. Add Button Translations
    print("\n5. Adding Button Translations")
    print("-" * 60)

    button_translations = [
        # English
        {"language_code": "en", "context": "button", "key": "save", "value": "Save"},
        {"language_code": "en", "context": "button", "key": "cancel", "value": "Cancel"},
        {"language_code": "en", "context": "button", "key": "delete", "value": "Delete"},
        {"language_code": "en", "context": "button", "key": "new", "value": "New"},

        # Spanish
        {"language_code": "es", "context": "button", "key": "save", "value": "Guardar"},
        {"language_code": "es", "context": "button", "key": "cancel", "value": "Cancelar"},
        {"language_code": "es", "context": "button", "key": "delete", "value": "Eliminar"},
        {"language_code": "es", "context": "button", "key": "new", "value": "Nuevo"},

        # Dutch
        {"language_code": "nl", "context": "button", "key": "save", "value": "Opslaan"},
        {"language_code": "nl", "context": "button", "key": "cancel", "value": "Annuleren"},
        {"language_code": "nl", "context": "button", "key": "delete", "value": "Verwijderen"},
        {"language_code": "nl", "context": "button", "key": "new", "value": "Nieuw"},

        # French
        {"language_code": "fr", "context": "button", "key": "save", "value": "Enregistrer"},
        {"language_code": "fr", "context": "button", "key": "cancel", "value": "Annuler"},
        {"language_code": "fr", "context": "button", "key": "delete", "value": "Supprimer"},
        {"language_code": "fr", "context": "button", "key": "new", "value": "Nouveau"},
    ]

    count = tm.bulk_add_translations(button_translations)
    print(f"✓ Added {count} button translations")

    # 7. Add Message Translations
    print("\n6. Adding Message Translations")
    print("-" * 60)

    message_translations = [
        # English
        {"language_code": "en", "context": "message", "key": "record_saved", "value": "Record saved successfully"},
        {"language_code": "en", "context": "message", "key": "record_deleted", "value": "Record deleted successfully"},
        {"language_code": "en", "context": "message", "key": "confirm_delete", "value": "Are you sure you want to delete this record?"},

        # Spanish
        {"language_code": "es", "context": "message", "key": "record_saved", "value": "Registro guardado exitosamente"},
        {"language_code": "es", "context": "message", "key": "record_deleted", "value": "Registro eliminado exitosamente"},
        {"language_code": "es", "context": "message", "key": "confirm_delete", "value": "¿Está seguro de que desea eliminar este registro?"},

        # Dutch
        {"language_code": "nl", "context": "message", "key": "record_saved", "value": "Record succesvol opgeslagen"},
        {"language_code": "nl", "context": "message", "key": "record_deleted", "value": "Record succesvol verwijderd"},
        {"language_code": "nl", "context": "message", "key": "confirm_delete", "value": "Weet u zeker dat u dit record wilt verwijderen?"},

        # French
        {"language_code": "fr", "context": "message", "key": "record_saved", "value": "Enregistrement sauvegardé avec succès"},
        {"language_code": "fr", "context": "message", "key": "record_deleted", "value": "Enregistrement supprimé avec succès"},
        {"language_code": "fr", "context": "message", "key": "confirm_delete", "value": "Êtes-vous sûr de vouloir supprimer cet enregistrement?"},
    ]

    count = tm.bulk_add_translations(message_translations)
    print(f"✓ Added {count} message translations")

    # 8. Demonstrate Translations
    print("\n7. Demonstrating Translations")
    print("-" * 60)

    # Test table name translations
    print("\nTable Name: 'Customers'")
    tm.set_language("en")
    print(f"  English: {tm.translate_table_name('Customers')}")
    tm.set_language("es")
    print(f"  Spanish: {tm.translate_table_name('Customers')}")
    tm.set_language("nl")
    print(f"  Dutch: {tm.translate_table_name('Customers')}")
    tm.set_language("fr")
    print(f"  French: {tm.translate_table_name('Customers')}")

    # Test field name translations
    print("\nField Name: 'email'")
    print(f"  English: {tm.translate_field_name('email', 'en')}")
    print(f"  Spanish: {tm.translate_field_name('email', 'es')}")
    print(f"  Dutch: {tm.translate_field_name('email', 'nl')}")
    print(f"  French: {tm.translate_field_name('email', 'fr')}")

    # Test button translations
    print("\nButton: 'save'")
    print(f"  English: {tm.translate_button('save', 'en')}")
    print(f"  Spanish: {tm.translate_button('save', 'es')}")
    print(f"  Dutch: {tm.translate_button('save', 'nl')}")
    print(f"  French: {tm.translate_button('save', 'fr')}")

    # Test message translations
    print("\nMessage: 'record_saved'")
    print(f"  English: {tm.translate_message('record_saved', 'en')}")
    print(f"  Spanish: {tm.translate_message('record_saved', 'es')}")
    print(f"  Dutch: {tm.translate_message('record_saved', 'nl')}")
    print(f"  French: {tm.translate_message('record_saved', 'fr')}")

    # 9. Using the global t() function
    print("\n8. Using Global t() Function")
    print("-" * 60)

    tm.set_language("es")
    print(f"Current language: Spanish")
    print(f"  t('save', 'button'): {t('save', 'button')}")
    print(f"  t('Customers', 'table_name'): {t('Customers', 'table_name')}")
    print(f"  t('record_saved'): {t('record_saved')}")

    # 10. Simulating a Multi-Language Form
    print("\n9. Multi-Language Customer Form")
    print("-" * 60)

    def display_customer_form(language: str):
        """Display a customer form in the specified language."""
        tm.set_language(language)

        print(f"\n--- {Language.get_by_code(db, language).native_name} ---")
        print(f"{t('Customers', 'table_name')} Form")
        print(f"")
        print(f"  {t('name', 'field_name')}: ___________________")
        print(f"  {t('email', 'field_name')}: ___________________")
        print(f"  {t('phone', 'field_name')}: ___________________")
        print(f"  {t('address', 'field_name')}: ___________________")
        print(f"")
        print(f"  [{t('save', 'button')}]  [{t('cancel', 'button')}]")

    # Display form in different languages
    for lang_code in ['en', 'es', 'nl', 'fr']:
        display_customer_form(lang_code)

    # 11. Translation Statistics
    print("\n10. Translation Statistics")
    print("-" * 60)

    all_translations = tm.get_all_translations()
    by_language = {}
    by_context = {}

    for trans in all_translations:
        lang = trans['language_code']
        context = trans['context']

        by_language[lang] = by_language.get(lang, 0) + 1
        by_context[context] = by_context.get(context, 0) + 1

    print("\nTranslations by Language:")
    for lang, count in sorted(by_language.items()):
        lang_obj = Language.get_by_code(db, lang)
        print(f"  {lang_obj.native_name}: {count} translations")

    print("\nTranslations by Context:")
    for context, count in sorted(by_context.items()):
        print(f"  {context}: {count} translations")

    print(f"\nTotal translations: {len(all_translations)}")

    print("\n=== Demo Complete ===")
    db.close()


if __name__ == '__main__':
    main()
