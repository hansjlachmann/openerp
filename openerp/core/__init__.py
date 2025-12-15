"""Core ERP functionality"""

from openerp.core.database import Database
from openerp.core.executor import CodeExecutor
from openerp.core.triggers import TriggerManager
from openerp.core.crud import CRUDManager

__all__ = ["Database", "CodeExecutor", "TriggerManager", "CRUDManager"]
