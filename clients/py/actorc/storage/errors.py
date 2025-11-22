"""
Error types for storage operations.
"""


class StorageError(Exception):
    """Base exception for all storage errors."""

    def __init__(self, message: str, code: str = None, bucket: str = None, key: str = None):
        self.message = message
        self.code = code or "StorageError"
        self.bucket = bucket
        self.key = key
        super().__init__(self._format_message())

    def _format_message(self) -> str:
        msg = f"[{self.code}] {self.message}"
        if self.bucket and self.key:
            msg += f" (bucket={self.bucket}, key={self.key})"
        elif self.bucket:
            msg += f" (bucket={self.bucket})"
        return msg


class NotFoundError(StorageError):
    """Exception raised when an object or bucket is not found."""

    def __init__(self, message: str, bucket: str = None, key: str = None):
        super().__init__(message, code="NotFound", bucket=bucket, key=key)


class PermissionError(StorageError):
    """Exception raised when access is denied."""

    def __init__(self, message: str, bucket: str = None, key: str = None):
        super().__init__(message, code="PermissionDenied", bucket=bucket, key=key)


class NetworkError(StorageError):
    """Exception raised for network-related errors."""

    def __init__(self, message: str):
        super().__init__(message, code="NetworkError")


class InvalidParameterError(StorageError):
    """Exception raised for invalid parameters."""

    def __init__(self, message: str):
        super().__init__(message, code="InvalidParameter")

