"""Python code execution engine with security restrictions"""

from typing import Any, Dict, Optional
from datetime import datetime
import pytz
from RestrictedPython import compile_restricted, safe_globals
from RestrictedPython.Guards import (
    guarded_iter_unpack_sequence,
    guarded_unpack_sequence,
    safe_builtins,
    safer_getattr
)
from RestrictedPython.PrintCollector import PrintCollector

# Save reference to the real __import__ before it's restricted
_real_import = __import__


class CodeExecutor:
    """
    Safe Python code execution engine for business logic.

    Uses RestrictedPython to safely execute user-defined code with
    limited access to system resources.
    """

    def __init__(self):
        """Initialize the code executor with safe globals."""
        self.safe_builtins = self._build_safe_builtins()

    def _build_safe_builtins(self) -> Dict[str, Any]:
        """
        Build a safe set of builtins for code execution.

        Returns:
            Dictionary of safe builtin functions and modules
        """
        # Define safe import function
        def safe_import(name, *args, **kwargs):
            """Allow only safe modules to be imported."""
            safe_modules = {'re', 'datetime', 'json', 'math', 'decimal', 'uuid', 'time'}
            if name in safe_modules:
                return _real_import(name, *args, **kwargs)
            raise ImportError(f"Import of module '{name}' is not allowed")

        # Start with RestrictedPython's safe_builtins
        builtins_dict = safe_builtins.copy()

        # Add safe_globals for additional safety
        builtins_dict.update(safe_globals)

        # Create custom __builtins__ with safe_import
        custom_builtins = safe_builtins.copy()
        custom_builtins['__import__'] = safe_import

        # Define safe container write function
        def safe_write(obj):
            """Allow writing to containers like dicts and lists."""
            return obj

        # Define safe getitem function for dictionary/list access
        def safe_getitem(obj, key):
            """Allow getting items from dicts and lists."""
            return obj[key]

        # Add safe utilities
        builtins_dict.update({
            '_getiter_': iter,
            '_iter_unpack_sequence_': guarded_iter_unpack_sequence,
            '_unpack_sequence_': guarded_unpack_sequence,
            '_getattr_': safer_getattr,  # Guarded attribute access
            '_getitem_': safe_getitem,  # Allow dict/list item access
            '_write_': safe_write,  # Allow container writes
            '_print_': PrintCollector,  # RestrictedPython's print collector
            '__builtins__': custom_builtins,  # Custom builtins with safe_import
            '__import__': safe_import,  # Also add to global namespace
            'datetime': datetime,
            'pytz': pytz,
        })

        return builtins_dict

    def execute(
        self,
        code: str,
        context: Optional[Dict[str, Any]] = None,
        mode: str = 'exec'
    ) -> Dict[str, Any]:
        """
        Execute Python code in a restricted environment.

        Args:
            code: Python code to execute
            context: Dictionary of variables to make available to the code
            mode: Execution mode ('exec' or 'eval')

        Returns:
            Dictionary containing the execution result and any errors

        Example:
            executor = CodeExecutor()
            result = executor.execute(
                "total = price * quantity",
                {"price": 10, "quantity": 5}
            )
            print(result['context']['total'])  # 50
        """
        if context is None:
            context = {}

        # Create execution namespace
        exec_globals = self.safe_builtins.copy()
        exec_globals.update(context)

        try:
            # Compile with restrictions
            byte_code = compile_restricted(code, '<string>', mode)

            # Check if compile_restricted returned errors
            if hasattr(byte_code, 'errors') and byte_code.errors:
                return {
                    'success': False,
                    'errors': byte_code.errors,
                    'context': context
                }

            # Execute the code
            if mode == 'exec':
                exec(byte_code, exec_globals)

                # Handle print output from PrintCollector
                # RestrictedPython creates a _print instance (without trailing underscore)
                if '_print' in exec_globals:
                    printer = exec_globals['_print']
                    if hasattr(printer, '__call__'):
                        printed = printer()
                        if printed:
                            print(printed, end='')

                # Extract modified context (excluding builtins)
                result_context = {
                    k: v for k, v in exec_globals.items()
                    if k not in self.safe_builtins and not k.startswith('_')
                }
            else:  # eval
                result = eval(byte_code, exec_globals)
                result_context = {'result': result}

            return {
                'success': True,
                'errors': [],
                'context': result_context
            }

        except Exception as e:
            return {
                'success': False,
                'errors': [str(e)],
                'context': context,
                'exception': e
            }

    def validate_code(self, code: str) -> Dict[str, Any]:
        """
        Validate code without executing it.

        Args:
            code: Python code to validate

        Returns:
            Dictionary with validation result
        """
        try:
            byte_code = compile_restricted(code, '<string>', 'exec')

            # Check if compile_restricted returned errors
            if hasattr(byte_code, 'errors') and byte_code.errors:
                return {
                    'valid': False,
                    'errors': byte_code.errors
                }

            return {
                'valid': True,
                'errors': []
            }

        except Exception as e:
            return {
                'valid': False,
                'errors': [str(e)]
            }
