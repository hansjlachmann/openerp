"""Tests for code executor module"""

import pytest
from openerp.core.executor import CodeExecutor


class TestCodeExecutor:
    """Test cases for CodeExecutor class."""

    def test_simple_execution(self):
        """Test simple code execution."""
        executor = CodeExecutor()

        result = executor.execute(
            "total = price * quantity",
            {"price": 10, "quantity": 5}
        )

        assert result['success'] is True
        assert result['context']['total'] == 50

    def test_datetime_access(self):
        """Test that datetime is available in execution context."""
        executor = CodeExecutor()

        result = executor.execute(
            "from datetime import datetime\nnow = datetime.now()",
            {}
        )

        assert result['success'] is True
        assert 'now' in result['context']

    def test_safe_builtins(self):
        """Test that safe builtins are available."""
        executor = CodeExecutor()

        result = executor.execute(
            "result = sum([1, 2, 3, 4, 5])",
            {}
        )

        assert result['success'] is True
        assert result['context']['result'] == 15

    def test_restricted_imports(self):
        """Test that dangerous imports are restricted."""
        executor = CodeExecutor()

        result = executor.execute(
            "import os\nos.system('echo test')",
            {}
        )

        assert result['success'] is False
        assert len(result['errors']) > 0

    def test_eval_mode(self):
        """Test evaluation mode."""
        executor = CodeExecutor()

        result = executor.execute(
            "10 + 20",
            mode='eval'
        )

        assert result['success'] is True
        assert result['context']['result'] == 30

    def test_code_validation(self):
        """Test code validation without execution."""
        executor = CodeExecutor()

        # Valid code
        result = executor.validate_code("x = 10")
        assert result['valid'] is True

        # Invalid syntax
        result = executor.validate_code("x = ")
        assert result['valid'] is False

    def test_execution_error(self):
        """Test that execution errors are caught."""
        executor = CodeExecutor()

        result = executor.execute(
            "result = 1 / 0",
            {}
        )

        assert result['success'] is False
        assert 'division by zero' in str(result['errors']).lower()

    def test_record_modification(self):
        """Test modifying record dictionary."""
        executor = CodeExecutor()

        record = {'name': 'John', 'age': 30}
        result = executor.execute(
            "record['email'] = 'john@example.com'",
            {'record': record}
        )

        assert result['success'] is True
        assert result['context']['record']['email'] == 'john@example.com'
