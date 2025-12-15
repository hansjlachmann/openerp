"""Internationalization (i18n) and Translation Management"""

from typing import Dict, List, Optional, Any
from openerp.core.database import Database


class TranslationContext:
    """Translation context types."""
    TABLE_NAME = "table_name"
    FIELD_NAME = "field_name"
    FORM_LABEL = "form_label"
    BUTTON = "button"
    MESSAGE = "message"
    REPORT = "report"
    DIALOG = "dialog"
    MENU = "menu"
    VALIDATION = "validation"
    HELP_TEXT = "help_text"


class Language:
    """Language model for multi-language support."""

    TABLE_NAME = "Language"  # Global table

    def __init__(
        self,
        code: str,
        name: str,
        native_name: str,
        is_default: bool = False
    ):
        """
        Initialize a Language instance.

        Args:
            code: ISO 639-1 language code (e.g., "en", "es", "fr", "nl")
            name: English name of the language
            native_name: Name in the language itself
            is_default: Whether this is the default system language
        """
        self.code = code
        self.name = name
        self.native_name = native_name
        self.is_default = is_default

    @classmethod
    def _ensure_table_exists(cls, db: Database):
        """Ensure the Language global table exists."""
        cursor = db.conn.cursor()
        cursor.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name=?",
            (cls.TABLE_NAME,)
        )

        if not cursor.fetchone():
            db.create_table(
                cls.TABLE_NAME,
                {
                    'code': 'TEXT NOT NULL UNIQUE',
                    'name': 'TEXT NOT NULL',
                    'native_name': 'TEXT NOT NULL',
                    'is_default': 'INTEGER DEFAULT 0'
                },
                is_global=True
            )

    @classmethod
    def create(
        cls,
        db: Database,
        code: str,
        name: str,
        native_name: str,
        is_default: bool = False
    ) -> "Language":
        """
        Create a new language.

        Args:
            db: Database instance
            code: ISO 639-1 language code
            name: English name
            native_name: Native name
            is_default: Set as default language

        Returns:
            Language instance

        Example:
            lang = Language.create(db, "en", "English", "English", is_default=True)
            lang = Language.create(db, "es", "Spanish", "Español")
            lang = Language.create(db, "nl", "Dutch", "Nederlands")
        """
        cls._ensure_table_exists(db)

        # Validate code format (2 or 3 letter code)
        if not code or len(code) not in [2, 3] or not code.isalpha():
            raise ValueError(f"Invalid language code '{code}'. Use ISO 639-1/2 format (e.g., 'en', 'es')")

        # If setting as default, unset other defaults
        if is_default:
            cursor = db.conn.cursor()
            cursor.execute(f'UPDATE "{cls.TABLE_NAME}" SET is_default = 0')
            db.conn.commit()

        # Insert language
        cursor = db.conn.cursor()
        cursor.execute(
            f'INSERT INTO "{cls.TABLE_NAME}" (code, name, native_name, is_default) VALUES (?, ?, ?, ?)',
            (code.lower(), name, native_name, 1 if is_default else 0)
        )
        db.conn.commit()

        return cls(code.lower(), name, native_name, is_default)

    @classmethod
    def get_by_code(cls, db: Database, code: str) -> Optional["Language"]:
        """Get language by code."""
        cls._ensure_table_exists(db)

        cursor = db.conn.cursor()
        cursor.execute(
            f'SELECT * FROM "{cls.TABLE_NAME}" WHERE code = ?',
            (code.lower(),)
        )
        row = cursor.fetchone()

        if row:
            row_dict = dict(row)
            return cls(
                row_dict['code'],
                row_dict['name'],
                row_dict['native_name'],
                bool(row_dict['is_default'])
            )
        return None

    @classmethod
    def get_default(cls, db: Database) -> Optional["Language"]:
        """Get the default language."""
        cls._ensure_table_exists(db)

        cursor = db.conn.cursor()
        cursor.execute(f'SELECT * FROM "{cls.TABLE_NAME}" WHERE is_default = 1')
        row = cursor.fetchone()

        if row:
            row_dict = dict(row)
            return cls(
                row_dict['code'],
                row_dict['name'],
                row_dict['native_name'],
                True
            )
        return None

    @classmethod
    def list_all(cls, db: Database) -> List["Language"]:
        """List all available languages."""
        cls._ensure_table_exists(db)

        cursor = db.conn.cursor()
        cursor.execute(f'SELECT * FROM "{cls.TABLE_NAME}" ORDER BY name')
        rows = cursor.fetchall()

        return [
            cls(
                dict(row)['code'],
                dict(row)['name'],
                dict(row)['native_name'],
                bool(dict(row)['is_default'])
            )
            for row in rows
        ]

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary."""
        return {
            'code': self.code,
            'name': self.name,
            'native_name': self.native_name,
            'is_default': self.is_default
        }

    def __repr__(self):
        return f"<Language {self.code}: {self.native_name}>"


class TranslationManager:
    """
    Manages translations for multi-language support.

    Supports translation of:
    - Table names
    - Field names
    - Form labels
    - Buttons
    - Messages
    - Reports
    - Dialogs
    - Menus
    - Validation messages
    - Help text
    """

    TABLE_NAME = "Translation"  # Global table

    def __init__(self, db: Database, current_language: str = "en"):
        """
        Initialize translation manager.

        Args:
            db: Database instance
            current_language: Current language code (default: "en")
        """
        self.db = db
        self.current_language = current_language
        self._ensure_table_exists()
        self._cache: Dict[str, Dict[str, str]] = {}

    def _ensure_table_exists(self):
        """Ensure the Translation global table exists."""
        cursor = self.db.conn.cursor()
        cursor.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name=?",
            (self.TABLE_NAME,)
        )

        if not cursor.fetchone():
            self.db.create_table(
                self.TABLE_NAME,
                {
                    'language_code': 'TEXT NOT NULL',
                    'context': 'TEXT NOT NULL',
                    'key': 'TEXT NOT NULL',
                    'value': 'TEXT NOT NULL'
                },
                is_global=True
            )

            # Create unique index
            cursor = self.db.conn.cursor()
            cursor.execute(f'''
                CREATE UNIQUE INDEX IF NOT EXISTS idx_translation_unique
                ON "{self.TABLE_NAME}" (language_code, context, key)
            ''')
            self.db.conn.commit()

    def add_translation(
        self,
        language_code: str,
        context: str,
        key: str,
        value: str
    ) -> bool:
        """
        Add or update a translation.

        Args:
            language_code: Language code (e.g., "en", "es", "nl")
            context: Translation context (e.g., "table_name", "field_name")
            key: Translation key (e.g., "customers", "email")
            value: Translated text

        Returns:
            True if successful

        Example:
            # Table name translations
            tm.add_translation("es", "table_name", "Customers", "Clientes")
            tm.add_translation("nl", "table_name", "Customers", "Klanten")

            # Field name translations
            tm.add_translation("es", "field_name", "name", "nombre")
            tm.add_translation("nl", "field_name", "name", "naam")

            # Button translations
            tm.add_translation("es", "button", "save", "Guardar")
            tm.add_translation("nl", "button", "save", "Opslaan")
        """
        cursor = self.db.conn.cursor()

        # Try to insert, or update if exists
        try:
            cursor.execute(f'''
                INSERT INTO "{self.TABLE_NAME}" (language_code, context, key, value)
                VALUES (?, ?, ?, ?)
                ON CONFLICT(language_code, context, key) DO UPDATE SET value = excluded.value
            ''', (language_code.lower(), context, key, value))
            self.db.conn.commit()

            # Clear cache for this language
            cache_key = f"{language_code}:{context}"
            if cache_key in self._cache:
                del self._cache[cache_key]

            return True
        except Exception as e:
            print(f"Error adding translation: {e}")
            return False

    def bulk_add_translations(self, translations: List[Dict[str, str]]) -> int:
        """
        Add multiple translations at once.

        Args:
            translations: List of dicts with keys: language_code, context, key, value

        Returns:
            Number of translations added

        Example:
            translations = [
                {"language_code": "es", "context": "table_name", "key": "Customers", "value": "Clientes"},
                {"language_code": "es", "context": "field_name", "key": "name", "value": "nombre"},
                {"language_code": "nl", "context": "table_name", "key": "Customers", "value": "Klanten"},
            ]
            tm.bulk_add_translations(translations)
        """
        count = 0
        for trans in translations:
            if self.add_translation(
                trans['language_code'],
                trans['context'],
                trans['key'],
                trans['value']
            ):
                count += 1
        return count

    def translate(
        self,
        key: str,
        context: str = TranslationContext.MESSAGE,
        language: Optional[str] = None,
        fallback: Optional[str] = None
    ) -> str:
        """
        Get translation for a key.

        Args:
            key: Translation key
            context: Translation context
            language: Language code (uses current_language if not specified)
            fallback: Fallback text if translation not found (uses key if not specified)

        Returns:
            Translated text or fallback

        Example:
            # Basic usage
            tm.current_language = "es"
            text = tm.translate("save", "button")  # "Guardar"

            # With explicit language
            text = tm.translate("save", "button", language="nl")  # "Opslaan"

            # With custom fallback
            text = tm.translate("unknown_key", fallback="Default Text")
        """
        lang = (language or self.current_language).lower()
        cache_key = f"{lang}:{context}"

        # Check cache
        if cache_key not in self._cache:
            self._load_translations_to_cache(lang, context)

        # Get translation from cache
        translation = self._cache[cache_key].get(key)

        if translation:
            return translation

        # Fallback to default language if not current language
        if lang != "en":
            default_cache_key = f"en:{context}"
            if default_cache_key not in self._cache:
                self._load_translations_to_cache("en", context)

            translation = self._cache[default_cache_key].get(key)
            if translation:
                return translation

        # Return fallback or key
        return fallback if fallback is not None else key

    def _load_translations_to_cache(self, language_code: str, context: str):
        """Load translations for a language and context into cache."""
        cache_key = f"{language_code}:{context}"

        cursor = self.db.conn.cursor()
        cursor.execute(f'''
            SELECT key, value FROM "{self.TABLE_NAME}"
            WHERE language_code = ? AND context = ?
        ''', (language_code, context))

        rows = cursor.fetchall()
        self._cache[cache_key] = {row[0]: row[1] for row in rows}

    def set_language(self, language_code: str):
        """Set the current language."""
        self.current_language = language_code.lower()

    def get_all_translations(
        self,
        context: Optional[str] = None,
        language: Optional[str] = None
    ) -> List[Dict[str, str]]:
        """
        Get all translations, optionally filtered by context and/or language.

        Args:
            context: Filter by context
            language: Filter by language

        Returns:
            List of translation dictionaries
        """
        cursor = self.db.conn.cursor()

        query = f'SELECT * FROM "{self.TABLE_NAME}" WHERE 1=1'
        params = []

        if language:
            query += ' AND language_code = ?'
            params.append(language.lower())

        if context:
            query += ' AND context = ?'
            params.append(context)

        query += ' ORDER BY language_code, context, key'

        cursor.execute(query, tuple(params))
        rows = cursor.fetchall()

        return [dict(row) for row in rows]

    # Convenience methods for specific contexts

    def translate_table_name(self, table_name: str, language: Optional[str] = None) -> str:
        """Translate a table name."""
        return self.translate(table_name, TranslationContext.TABLE_NAME, language, table_name)

    def translate_field_name(self, field_name: str, language: Optional[str] = None) -> str:
        """Translate a field name."""
        return self.translate(field_name, TranslationContext.FIELD_NAME, language, field_name)

    def translate_label(self, label: str, language: Optional[str] = None) -> str:
        """Translate a form label."""
        return self.translate(label, TranslationContext.FORM_LABEL, language, label)

    def translate_button(self, button: str, language: Optional[str] = None) -> str:
        """Translate a button text."""
        return self.translate(button, TranslationContext.BUTTON, language, button)

    def translate_message(self, message: str, language: Optional[str] = None) -> str:
        """Translate a message."""
        return self.translate(message, TranslationContext.MESSAGE, language, message)

    def translate_validation(self, message: str, language: Optional[str] = None) -> str:
        """Translate a validation message."""
        return self.translate(message, TranslationContext.VALIDATION, language, message)


# Global translation helper function
_default_tm: Optional[TranslationManager] = None


def init_i18n(db: Database, default_language: str = "en") -> TranslationManager:
    """
    Initialize the global translation manager.

    Args:
        db: Database instance
        default_language: Default language code

    Returns:
        TranslationManager instance

    Example:
        from openerp.core.i18n import init_i18n, t

        db = Database('myerp.db')
        tm = init_i18n(db, "en")

        # Now you can use the global t() function
        text = t("save", "button")
    """
    global _default_tm
    _default_tm = TranslationManager(db, default_language)
    return _default_tm


def t(
    key: str,
    context: str = TranslationContext.MESSAGE,
    language: Optional[str] = None,
    fallback: Optional[str] = None
) -> str:
    """
    Global translation function (shorthand).

    Must call init_i18n() first.

    Args:
        key: Translation key
        context: Translation context
        language: Language code (optional)
        fallback: Fallback text (optional)

    Returns:
        Translated text

    Example:
        text = t("save", "button")  # Get translation for "save" button
        text = t("Customers", "table_name", language="es")  # Spanish table name
    """
    if _default_tm is None:
        raise RuntimeError("Translation system not initialized. Call init_i18n() first.")
    return _default_tm.translate(key, context, language, fallback)
