"""OpenERP - Lightweight ERP System"""

__version__ = "0.1.0"

from openerp.core.database import Database
from openerp.models.company import Company
from openerp.core.i18n import (
    Language,
    TranslationManager,
    TranslationContext,
    init_i18n,
    t
)

__all__ = [
    "Database",
    "Company",
    "Language",
    "TranslationManager",
    "TranslationContext",
    "init_i18n",
    "t"
]
