"""Utility functions and helpers"""

from datetime import datetime
import pytz


def now_utc() -> datetime:
    """Get current UTC timestamp."""
    return datetime.now(pytz.UTC)


def format_currency(amount: float, currency: str = "USD") -> str:
    """
    Format a currency amount.

    Args:
        amount: Amount to format
        currency: Currency code

    Returns:
        Formatted currency string
    """
    symbols = {
        'USD': '$',
        'EUR': '€',
        'GBP': '£',
        'JPY': '¥'
    }

    symbol = symbols.get(currency, currency)
    return f"{symbol}{amount:,.2f}"


def validate_email(email: str) -> bool:
    """
    Basic email validation.

    Args:
        email: Email address to validate

    Returns:
        True if valid format
    """
    import re
    pattern = r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'
    return bool(re.match(pattern, email))
