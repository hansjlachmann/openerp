from setuptools import setup, find_packages

setup(
    name="openerp",
    version="0.1.0",
    description="Lightweight ERP system built from scratch",
    author="OpenERP Team",
    packages=find_packages(),
    python_requires=">=3.9",
    install_requires=[
        "sqlalchemy>=2.0.0",
        "pydantic>=2.0.0",
        "RestrictedPython>=6.0",
        "python-dateutil>=2.8.0",
        "pytz>=2023.3",
    ],
    extras_require={
        "dev": [
            "pytest>=7.4.0",
            "pytest-cov>=4.1.0",
            "black>=23.0.0",
            "flake8>=6.0.0",
            "mypy>=1.5.0",
        ]
    },
    classifiers=[
        "Development Status :: 3 - Alpha",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: MIT License",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
    ],
)
