# Multi-Language Support (i18n)

OpenERP includes comprehensive multi-language support for internationalizing your ERP application.

## Overview

The translation system supports:
- ✅ **Table names**
- ✅ **Field names**
- ✅ **Form labels**
- ✅ **Buttons**
- ✅ **Messages**
- ✅ **Reports**
- ✅ **Dialogs**
- ✅ **Menus**
- ✅ **Validation messages**
- ✅ **Help text**

## Global Tables

The multi-language system uses two global tables:

### 1. `Language` Table

Stores available languages in the system.

```sql
CREATE TABLE Language (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT NOT NULL UNIQUE,         -- ISO 639-1 code (e.g., "en", "es", "nl")
    name TEXT NOT NULL,                 -- English name
    native_name TEXT NOT NULL,          -- Native name (e.g., "Español", "Nederlands")
    is_default INTEGER DEFAULT 0,       -- 1 if default language
    created_at TIMESTAMP,
    updated_at TIMESTAMP
)
```

### 2. `Translation` Table

Stores all translations.

```sql
CREATE TABLE Translation (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    language_code TEXT NOT NULL,        -- Language code
    context TEXT NOT NULL,              -- Translation context (table_name, field_name, etc.)
    key TEXT NOT NULL,                  -- Translation key
    value TEXT NOT NULL,                -- Translated text
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    UNIQUE(language_code, context, key)
)
```

## Translation Contexts

The system supports different translation contexts:

| Context | Description | Example Key | Example Value |
|---------|-------------|-------------|---------------|
| `table_name` | Table names | "Customers" | "Clientes" (es) |
| `field_name` | Field names | "email" | "correo electrónico" (es) |
| `form_label` | Form labels | "customer_form" | "Formulario de Cliente" (es) |
| `button` | Button text | "save" | "Guardar" (es) |
| `message` | User messages | "record_saved" | "Registro guardado" (es) |
| `report` | Report names | "sales_report" | "Informe de Ventas" (es) |
| `dialog` | Dialog titles | "confirm_delete" | "Confirmar eliminación" (es) |
| `menu` | Menu items | "file_menu" | "Archivo" (es) |
| `validation` | Validation messages | "email_required" | "El correo es obligatorio" (es) |
| `help_text` | Help text | "email_help" | "Ingrese un correo válido" (es) |

## Quick Start

### 1. Initialize the Translation System

```python
from openerp import Database, Language, init_i18n

db = Database('myerp.db')

# Initialize translation system with default language
tm = init_i18n(db, default_language="en")
```

### 2. Create Languages

```python
# Create languages
en = Language.create(db, "en", "English", "English", is_default=True)
es = Language.create(db, "es", "Spanish", "Español")
nl = Language.create(db, "nl", "Dutch", "Nederlands")
fr = Language.create(db, "fr", "French", "Français")
de = Language.create(db, "de", "German", "Deutsch")

# List all languages
languages = Language.list_all(db)
for lang in languages:
    print(f"{lang.code}: {lang.native_name}")
```

### 3. Add Translations

```python
# Single translation
tm.add_translation("es", "table_name", "Customers", "Clientes")
tm.add_translation("nl", "table_name", "Customers", "Klanten")

# Bulk translations
translations = [
    {"language_code": "es", "context": "button", "key": "save", "value": "Guardar"},
    {"language_code": "es", "context": "button", "key": "cancel", "value": "Cancelar"},
    {"language_code": "nl", "context": "button", "key": "save", "value": "Opslaan"},
    {"language_code": "nl", "context": "button", "key": "cancel", "value": "Annuleren"},
]
tm.bulk_add_translations(translations)
```

### 4. Use Translations

```python
# Set current language
tm.set_language("es")

# Translate using context-specific methods
table_name = tm.translate_table_name("Customers")  # "Clientes"
field_name = tm.translate_field_name("email")      # "correo electrónico"
button_text = tm.translate_button("save")           # "Guardar"
message = tm.translate_message("record_saved")      # "Registro guardado"

# Or use the generic translate method
text = tm.translate("save", "button")  # "Guardar"

# Translate with explicit language (overrides current language)
text = tm.translate("save", "button", language="nl")  # "Opslaan"

# With custom fallback
text = tm.translate("unknown_key", fallback="Default Text")
```

### 5. Using the Global `t()` Function

After calling `init_i18n()`, you can use the global `t()` function:

```python
from openerp import t

# Simple usage (uses current language)
text = t("save", "button")

# With explicit language
text = t("Customers", "table_name", language="es")

# With fallback
text = t("unknown", fallback="Not found")
```

## Complete Example

```python
from openerp import Database, Language, TranslationManager, TranslationContext, init_i18n, t

# Initialize
db = Database('myerp.db')
tm = init_i18n(db, "en")

# Set up languages
Language.create(db, "en", "English", "English", is_default=True)
Language.create(db, "es", "Spanish", "Español")
Language.create(db, "nl", "Dutch", "Nederlands")

# Add translations for a customer form
translations = [
    # Table name
    {"language_code": "es", "context": "table_name", "key": "Customers", "value": "Clientes"},
    {"language_code": "nl", "context": "table_name", "key": "Customers", "value": "Klanten"},

    # Field names
    {"language_code": "es", "context": "field_name", "key": "name", "value": "Nombre"},
    {"language_code": "es", "context": "field_name", "key": "email", "value": "Correo electrónico"},
    {"language_code": "nl", "context": "field_name", "key": "name", "value": "Naam"},
    {"language_code": "nl", "context": "field_name", "key": "email", "value": "E-mail"},

    # Buttons
    {"language_code": "es", "context": "button", "key": "save", "value": "Guardar"},
    {"language_code": "es", "context": "button", "key": "cancel", "value": "Cancelar"},
    {"language_code": "nl", "context": "button", "key": "save", "value": "Opslaan"},
    {"language_code": "nl", "context": "button", "key": "cancel", "value": "Annuleren"},
]

tm.bulk_add_translations(translations)

# Display form in Spanish
tm.set_language("es")
print(f"{t('Customers', 'table_name')} Form")  # "Clientes Form"
print(f"  {t('name', 'field_name')}: _____")   # "Nombre: _____"
print(f"  {t('email', 'field_name')}: _____")  # "Correo electrónico: _____"
print(f"  [{t('save', 'button')}] [{t('cancel', 'button')}]")  # "[Guardar] [Cancelar]"

# Display form in Dutch
tm.set_language("nl")
print(f"{t('Customers', 'table_name')} Form")  # "Klanten Form"
print(f"  {t('name', 'field_name')}: _____")   # "Naam: _____"
print(f"  {t('email', 'field_name')}: _____")  # "E-mail: _____"
print(f"  [{t('save', 'button')}] [{t('cancel', 'button')}]")  # "[Opslaan] [Annuleren]"
```

## Translation Workflow

### 1. Extract Translatable Strings

Identify all strings that need translation:
- Table names in `db.create_table()`
- Field names in table schemas
- UI labels and buttons
- Messages to users
- Report titles
- Dialog text

### 2. Create Translation Keys

Use consistent, descriptive keys:

```python
# Good keys (descriptive, consistent)
"customer_form_title"
"email_field_label"
"save_button"
"record_saved_message"

# Bad keys (generic, unclear)
"form1"
"label"
"btn"
"msg"
```

### 3. Add Base Language (English)

Always add English translations as a base:

```python
tm.add_translation("en", "form_label", "customer_form_title", "Customer Form")
tm.add_translation("en", "field_name", "email", "Email Address")
```

### 4. Add Other Languages

Add translations for each supported language:

```python
# Spanish
tm.add_translation("es", "form_label", "customer_form_title", "Formulario de Cliente")
tm.add_translation("es", "field_name", "email", "Dirección de Correo")

# Dutch
tm.add_translation("nl", "form_label", "customer_form_title", "Klantformulier")
tm.add_translation("nl", "field_name", "email", "E-mailadres")
```

### 5. Use Translations in Code

Replace hard-coded strings with translation calls:

```python
# Before
print("Customer Form")
print(f"  Email: {customer.email}")
print("  [Save] [Cancel]")

# After
tm.set_language(user_language)
print(t("customer_form_title", "form_label"))
print(f"  {t('email', 'field_name')}: {customer.email}")
print(f"  [{t('save', 'button')}] [{t('cancel', 'button')}]")
```

## Best Practices

### 1. Use Consistent Keys

```python
# Good - consistent naming
"customer_form"
"customer_list"
"customer_report"

# Bad - inconsistent
"custForm"
"CustomersList"
"report_customer"
```

### 2. Provide Context

Use appropriate translation contexts:

```python
# Field name vs Form label
tm.add_translation("es", "field_name", "email", "correo")
tm.add_translation("es", "form_label", "email", "Dirección de Correo Electrónico")
```

### 3. Use Fallbacks

Always provide fallback text:

```python
# With fallback
text = t("new_feature", fallback="New Feature")

# Or use the key as fallback (default behavior)
text = t("new_feature")  # Returns "new_feature" if not found
```

### 4. Cache Translations

The TranslationManager automatically caches translations per language and context for performance.

### 5. Handle Plurals

For languages with complex plural rules, use separate keys:

```python
tm.add_translation("en", "message", "items_one", "1 item")
tm.add_translation("en", "message", "items_many", "{count} items")

# In code
count = 5
if count == 1:
    msg = t("items_one")
else:
    msg = t("items_many").format(count=count)
```

### 6. Format Parameters

Use Python string formatting for dynamic values:

```python
# Add translation with placeholder
tm.add_translation("es", "message", "welcome_user", "Bienvenido, {username}!")

# Use with formatting
msg = t("welcome_user").format(username="Juan")  # "Bienvenido, Juan!"
```

## Language Management

### Get Default Language

```python
default_lang = Language.get_default(db)
print(f"Default language: {default_lang.native_name}")
```

### Check if Language Exists

```python
lang = Language.get_by_code(db, "es")
if lang:
    print(f"Spanish is available: {lang.native_name}")
```

### List All Languages

```python
languages = Language.list_all(db)
for lang in languages:
    default = " (default)" if lang.is_default else ""
    print(f"{lang.code}: {lang.native_name}{default}")
```

## Query Translations

### Get All Translations

```python
# All translations
all_trans = tm.get_all_translations()

# By language
spanish_trans = tm.get_all_translations(language="es")

# By context
button_trans = tm.get_all_translations(context="button")

# By language and context
spanish_buttons = tm.get_all_translations(language="es", context="button")
```

## Integration with Company-Specific Tables

Translations work seamlessly with the `CompanyName$TableName` architecture:

```python
# The actual table name in database
actual_table = "ACME$Customers"

# Parse to get base name
company, base_table = Database.parse_table_name(actual_table)

# Translate the base table name
translated = tm.translate_table_name(base_table)  # "Clientes" in Spanish

# Display
print(f"{company} - {translated}")  # "ACME - Clientes"
```

## Example: Multi-Language Report

```python
def generate_customer_report(db, tm, language="en"):
    """Generate customer report in specified language."""
    tm.set_language(language)

    # Report title
    print(f"=== {t('customer_report', 'report')} ===\n")

    # Column headers
    print(f"{t('name', 'field_name'):20} {t('email', 'field_name'):30} {t('phone', 'field_name'):15}")
    print("-" * 65)

    # Data rows
    crud = CRUDManager(db)
    customers = crud.get_all('ACME$Customers')

    for customer in customers['records']:
        print(f"{customer['name']:20} {customer['email']:30} {customer['phone']:15}")

    print(f"\n{t('total_records', 'message')}: {customers['count']}")

# Generate in different languages
generate_customer_report(db, tm, "en")
generate_customer_report(db, tm, "es")
generate_customer_report(db, tm, "nl")
```

## Common Language Codes (ISO 639-1)

| Code | English Name | Native Name |
|------|--------------|-------------|
| `en` | English | English |
| `es` | Spanish | Español |
| `fr` | French | Français |
| `de` | German | Deutsch |
| `it` | Italian | Italiano |
| `pt` | Portuguese | Português |
| `nl` | Dutch | Nederlands |
| `ru` | Russian | Русский |
| `ja` | Japanese | 日本語 |
| `zh` | Chinese | 中文 |
| `ar` | Arabic | العربية |
| `hi` | Hindi | हिन्दी |

## Performance Considerations

1. **Caching**: Translations are cached per language and context
2. **Bulk Operations**: Use `bulk_add_translations()` for adding multiple translations
3. **Lazy Loading**: Translations are loaded on-demand by language and context
4. **Index**: Unique index on (language_code, context, key) for fast lookups

## Troubleshooting

### Translation Not Found

```python
# Check if translation exists
trans = tm.get_all_translations(language="es", context="button")
print([t for t in trans if t['key'] == 'save'])

# Add missing translation
tm.add_translation("es", "button", "save", "Guardar")
```

### Wrong Language Displayed

```python
# Check current language
print(f"Current language: {tm.current_language}")

# Set correct language
tm.set_language("es")
```

### Fallback Not Working

```python
# Ensure default language (English) has translations
tm.add_translation("en", "button", "save", "Save")

# Or provide explicit fallback
text = t("save", "button", fallback="Save")
```

## See Also

- [COMPANY_ARCHITECTURE.md](COMPANY_ARCHITECTURE.md) - Multi-company setup
- [examples/multi_language_demo.py](examples/multi_language_demo.py) - Complete working example
