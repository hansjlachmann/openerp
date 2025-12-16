"""Python code execution engine with security restrictions"""

from typing import Any, Dict, Optional
from datetime import datetime
import pytz
from RestrictedPython import compile_restricted, safe_globals
from RestrictedPython.Guards import guarded_iter_unpack_sequence, guarded_unpack_sequence


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
        safe_builtins = safe_globals.copy()

        # Create a safe print function that actually prints
        def safe_print(*args, **kwargs):
            """Safe print implementation for RestrictedPython."""
            print(*args, **kwargs)
            # Return empty string to satisfy RestrictedPython's 'printed' check
            return ''

        # Add safe utilities
        safe_builtins.update({
            '_getiter_': iter,
            '_iter_unpack_sequence_': guarded_iter_unpack_sequence,
            '_unpack_sequence_': guarded_unpack_sequence,
            '_print_': safe_print,  # RestrictedPython uses _print_
            '_getattr_': getattr,   # Add getattr support
            'datetime': datetime,
            'pytz': pytz,
            # Safe built-in functions
            'len': len,
            'str': str,
            'int': int,
            'float': float,
            'bool': bool,
            'list': list,
            'dict': dict,
            'tuple': tuple,
            'set': set,
            'min': min,
            'max': max,
            'sum': sum,
            'abs': abs,
            'round': round,
            'sorted': sorted,
            'enumerate': enumerate,
            'zip': zip,
            'range': range,
            'print': safe_print,  # Also add as 'print' for direct calls
        })

        return safe_builtins

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
