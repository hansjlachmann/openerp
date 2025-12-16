"""OpenERP - Lightweight ERP System"""

__version__ = "0.1.0"

from openerp.core.database import Database
from openerp.models.company import Company

__all__ = ["Database", "Company"]
