# Multi-Language Translation System

OpenERP includes a built-in multi-language translation system for table names and field names. Translations are stored as JSON directly in the metadata tables, making them efficient and easy to manage.

## Overview

The translation system allows you to:
- Store translations for table names in multiple languages
- Store translations for field names in multiple languages
- Retrieve translations with fallback support
- Build multi-language user interfaces

## Storage Architecture

Translations are stored as JSON strings in two metadata tables:

### Table Name Translations

Stored in `__table_metadata.translations` column:

```json
{
  "en": "Customers",
  "es": "Clientes",
  "nl": "Klanten",
  "fr": "Clients",
  "de": "Kunden"
}
```

### Field Name Translations

Stored in `__field_metadata.translations` column:

```json
{
  "en": "Email",
  "es": "Correo electrónico",
  "nl": "E-mail",
  "fr": "Courriel",
  "de": "E-Mail"
}
```

## API Methods

### Table Translation Methods

#### Set Table Translation

```python
db.set_table_translation(table_name: str, language_code: str, translation: str)
```

**Parameters:**
- `table_name`: Full table name (e.g., "ACME$Customers" or "Company")
- `language_code`: ISO language code (e.g., "en", "es", "nl", "fr", "de")
- `translation`: Translated table name

**Example:**
```python
db.set_table_translation("ACME$Customers", "es", "Clientes")
db.set_table_translation("ACME$Customers", "fr", "Clients")
```

#### Get Table Translation

```python
db.get_table_translation(table_name: str, language_code: str, fallback: Optional[str] = None) -> str
```

**Parameters:**
- `table_name`: Full table name
- `language_code`: ISO language code
- `fallback`: Optional fallback value if translation not found (defaults to table_name)

**Returns:** Translated table name or fallback

**Example:**
```python
# Get Spanish translation
spanish = db.get_table_translation("ACME$Customers", "es")
# Returns: "Clientes"

# With fallback for unsupported language
japanese = db.get_table_translation("ACME$Customers", "ja", fallback="お客様")
# Returns: "お客様"

# Without fallback returns original table name
chinese = db.get_table_translation("ACME$Customers", "zh")
# Returns: "ACME$Customers"
```

#### Get All Table Translations

```python
db.get_table_translations(table_name: str) -> Dict[str, str]
```

**Parameters:**
- `table_name`: Full table name

**Returns:** Dictionary of all translations

**Example:**
```python
translations = db.get_table_translations("ACME$Customers")
# Returns: {"en": "Customers", "es": "Clientes", "nl": "Klanten", ...}
```

### Field Translation Methods

#### Set Field Translation

```python
db.set_field_translation(table_name: str, field_name: str, language_code: str, translation: str)
```

**Parameters:**
- `table_name`: Full table name
- `field_name`: Field name
- `language_code`: ISO language code
- `translation`: Translated field name

**Example:**
```python
db.set_field_translation("ACME$Customers", "Email", "es", "Correo electrónico")
db.set_field_translation("ACME$Customers", "Email", "fr", "Courriel")
```

#### Get Field Translation

```python
db.get_field_translation(table_name: str, field_name: str, language_code: str, fallback: Optional[str] = None) -> str
```

**Parameters:**
- `table_name`: Full table name
- `field_name`: Field name
- `language_code`: ISO language code
- `fallback`: Optional fallback value (defaults to field_name)

**Returns:** Translated field name or fallback

**Example:**
```python
# Get Spanish translation
email_es = db.get_field_translation("ACME$Customers", "Email", "es")
# Returns: "Correo electrónico"

# With fallback
email_ja = db.get_field_translation("ACME$Customers", "Email", "ja", fallback="メール")
# Returns: "メール"
```

#### Get All Field Translations

```python
db.get_field_translations(table_name: str, field_name: str) -> Dict[str, str]
```

**Parameters:**
- `table_name`: Full table name
- `field_name`: Field name

**Returns:** Dictionary of all translations

**Example:**
```python
translations = db.get_field_translations("ACME$Customers", "Email")
# Returns: {"en": "Email", "es": "Correo electrónico", "nl": "E-mail", ...}
```

## Complete Example

```python
from openerp import Database
from openerp.models.company import Company

# Create database
db = Database("myapp.db")

# Create company and table
company = Company.create(db, "ACME")
db.create_table("Customers", {
    "Name": "TEXT NOT NULL",
    "Email": "TEXT NOT NULL",
    "Phone": "TEXT"
}, company_name="ACME")

# Add table translations
table_name = "ACME$Customers"
db.set_table_translation(table_name, "en", "Customers")
db.set_table_translation(table_name, "es", "Clientes")
db.set_table_translation(table_name, "nl", "Klanten")
db.set_table_translation(table_name, "fr", "Clients")
db.set_table_translation(table_name, "de", "Kunden")

# Add field translations
fields = {
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
    }
}

for field_name, translations in fields.items():
    for lang_code, translation in translations.items():
        db.set_field_translation(table_name, field_name, lang_code, translation)

# Build a multi-language form
def render_form(db, table_name, language):
    form_title = db.get_table_translation(table_name, language)
    print(f"Form: {form_title}")
    print("-" * 40)

    # Get fields from table
    field_names = ["Name", "Email", "Phone"]

    for field_name in field_names:
        field_label = db.get_field_translation(table_name, field_name, language)
        print(f"  {field_label}: [____________]")

# Render in different languages
render_form(db, table_name, "en")  # English
render_form(db, table_name, "es")  # Spanish
render_form(db, table_name, "nl")  # Dutch
```

## Language Codes

The system accepts standard ISO 639-1 language codes. Common examples:

- `en` - English
- `es` - Spanish
- `nl` - Dutch
- `fr` - French
- `de` - German
- `it` - Italian
- `pt` - Portuguese
- `ja` - Japanese
- `zh` - Chinese
- `ru` - Russian

Language codes are case-insensitive (internally converted to lowercase).

## Best Practices

### 1. Set English as Default

Always provide English translations first, as they serve as the base language:

```python
db.set_table_translation(table_name, "en", "Customers")
```

### 2. Use Consistent Naming

Use the same language codes consistently across your application.

### 3. Provide Fallbacks

When retrieving translations in user-facing code, always provide appropriate fallbacks:

```python
# Good - provides fallback
label = db.get_field_translation(table_name, field_name, user_language, fallback=field_name)

# Less ideal - might return technical names to users
label = db.get_field_translation(table_name, field_name, user_language)
```

### 4. Batch Operations

When setting up translations for a new table, set all translations in a batch:

```python
def setup_translations(db, table_name):
    # Table translations
    table_trans = {
        "en": "Customers",
        "es": "Clientes",
        "nl": "Klanten"
    }
    for lang, trans in table_trans.items():
        db.set_table_translation(table_name, lang, trans)

    # Field translations
    field_trans = {
        "Name": {"en": "Name", "es": "Nombre", "nl": "Naam"},
        "Email": {"en": "Email", "es": "Correo", "nl": "E-mail"}
    }
    for field, langs in field_trans.items():
        for lang, trans in langs.items():
            db.set_field_translation(table_name, field, lang, trans)
```

### 5. Handle Missing Translations Gracefully

Check for missing translations and log warnings:

```python
translation = db.get_table_translation(table_name, language)
if translation == table_name:
    logger.warning(f"Missing {language} translation for table {table_name}")
```

## Future Extensions

The current system supports table and field name translations. Future versions may include:

- Form title translations
- Button label translations
- Report header translations
- Dialog message translations
- Help text translations
- Validation message translations

These would likely be stored in similar JSON fields in their respective metadata structures.

## Technical Details

### Storage Format

Translations are stored as JSON text in SQLite TEXT columns. Example structure:

```sql
-- In __table_metadata
CREATE TABLE __table_metadata (
    ...
    translations TEXT,  -- JSON: {"en": "...", "es": "...", ...}
    ...
);

-- In __field_metadata
CREATE TABLE __field_metadata (
    ...
    translations TEXT,  -- JSON: {"en": "...", "es": "...", ...}
    ...
);
```

### JSON Structure

The JSON structure is a simple key-value object:

```json
{
  "language_code": "translation",
  "language_code": "translation",
  ...
}
```

Language codes are stored in lowercase for consistency.

### Performance Considerations

- Translations are loaded from the database on each request
- For high-performance applications, consider caching translations in memory
- JSON parsing is fast for small translation dictionaries (< 100 languages)
- Indexes on `table_name` in metadata tables ensure fast lookups

### Database Schema

The translation fields are automatically created when initializing a Database instance:

```python
db = Database("myapp.db")
# __table_metadata.translations and __field_metadata.translations are created
```

No additional setup is required.

## Troubleshooting

### Translation Not Found

**Problem:** `get_table_translation()` returns the table name instead of translation

**Solutions:**
1. Check the table name is correct (including company prefix for company-specific tables)
2. Verify the translation was set: `db.get_table_translations(table_name)`
3. Check the language code is correct (case-insensitive)

### Cannot Set Translation

**Problem:** Error when calling `set_table_translation()` or `set_field_translation()`

**Solutions:**
1. Ensure the table exists in `__table_metadata`
2. For field translations, ensure the field exists in `__field_metadata`
3. Check database permissions (write access required)

### JSON Encoding Issues

**Problem:** Special characters not displaying correctly

**Solutions:**
1. Ensure UTF-8 encoding throughout your application
2. Python's `json` module handles Unicode correctly by default
3. Verify your database connection uses UTF-8: `Database(db_path)` sets this automatically

## See Also

- [Company Architecture](COMPANY_ARCHITECTURE.md) - Multi-company table structure
- [examples/translation_demo.py](examples/translation_demo.py) - Complete working example
- [Database API Reference](README.md#database-api) - Full database API documentation
