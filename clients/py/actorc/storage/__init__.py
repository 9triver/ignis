"""
Storage client module for uploading/downloading files to/from ignis storage.

This module provides a Python client for interacting with ignis storage services,
supporting basic object operations and streaming for large files.
"""

from .client import StorageClient
from .errors import StorageError, NotFoundError, PermissionError as StoragePermissionError

__all__ = [
    "StorageClient",
    "StorageError",
    "NotFoundError",
    "StoragePermissionError",
]

