"""
Storage client for interacting with ignis storage service.
"""

import os
import mimetypes
from pathlib import Path
from typing import Optional, Dict, BinaryIO, Iterator
import grpc

from .errors import (
    StorageError,
    NotFoundError,
    PermissionError as StoragePermissionError,
    NetworkError,
    InvalidParameterError,
)

# TODO: Import actual storage proto when available
# from ..protos import storage_pb2, storage_pb2_grpc


class StorageClient:
    """
    Client for uploading and downloading files to/from ignis storage.
    
    This client provides a simple interface for common storage operations,
    supporting both regular files and streaming for large objects.
    
    Example:
        ```python
        from actorc.storage import StorageClient
        
        # Initialize client
        client = StorageClient(endpoint="storage.ignis.local:9000")
        
        # Upload a file
        client.upload_file(
            bucket="my-bucket",
            key="data/file.txt",
            file_path="/path/to/local/file.txt"
        )
        
        # Download a file
        client.download_file(
            bucket="my-bucket",
            key="data/file.txt",
            file_path="/path/to/save/file.txt"
        )
        
        # Upload from bytes
        client.put_object(
            bucket="my-bucket",
            key="data/content.txt",
            data=b"Hello, Storage!"
        )
        
        # Download to bytes
        data = client.get_object(bucket="my-bucket", key="data/content.txt")
        print(data.decode())
        ```
    """

    DEFAULT_CHUNK_SIZE = 5 * 1024 * 1024  # 5MB

    def __init__(
        self,
        endpoint: str = None,
        access_key: str = None,
        secret_key: str = None,
        use_ssl: bool = False,
    ):
        """
        Initialize storage client.
        
        Args:
            endpoint: Storage service endpoint (e.g., "storage.ignis.local:9000")
                     If not provided, will try to read from STORAGE_ENDPOINT env var
            access_key: Access key for authentication (optional)
            secret_key: Secret key for authentication (optional)
            use_ssl: Whether to use SSL/TLS connection
        """
        self.endpoint = endpoint or os.getenv("STORAGE_ENDPOINT", "localhost:9000")
        self.access_key = access_key or os.getenv("STORAGE_ACCESS_KEY")
        self.secret_key = secret_key or os.getenv("STORAGE_SECRET_KEY")
        self.use_ssl = use_ssl

        # TODO: Initialize gRPC channel when proto is ready
        # if self.use_ssl:
        #     credentials = grpc.ssl_channel_credentials()
        #     self._channel = grpc.secure_channel(self.endpoint, credentials)
        # else:
        #     self._channel = grpc.insecure_channel(self.endpoint)
        # self._stub = storage_pb2_grpc.StorageServiceStub(self._channel)

    def put_object(
        self,
        bucket: str,
        key: str,
        data: bytes,
        content_type: str = None,
        metadata: Dict[str, str] = None,
    ) -> Dict:
        """
        Upload an object to storage.
        
        Args:
            bucket: Bucket name
            key: Object key (path)
            data: Object data as bytes
            content_type: MIME type (auto-detected if not provided)
            metadata: Custom metadata dictionary
            
        Returns:
            dict: Response with upload details (bucket, key, size, etag, etc.)
            
        Raises:
            StorageError: If upload fails
            InvalidParameterError: If parameters are invalid
        """
        self._validate_bucket_name(bucket)
        self._validate_object_key(key)

        if content_type is None:
            content_type = self._guess_content_type(key)

        # TODO: Implement actual gRPC call
        # For now, return mock response
        return {
            "bucket": bucket,
            "key": key,
            "size": len(data),
            "content_type": content_type,
            "etag": "mock-etag",
        }

    def get_object(self, bucket: str, key: str) -> bytes:
        """
        Download an object from storage.
        
        Args:
            bucket: Bucket name
            key: Object key (path)
            
        Returns:
            bytes: Object data
            
        Raises:
            NotFoundError: If object doesn't exist
            StorageError: If download fails
        """
        self._validate_bucket_name(bucket)
        self._validate_object_key(key)

        # TODO: Implement actual gRPC call
        raise NotFoundError(f"Object not found", bucket=bucket, key=key)

    def delete_object(self, bucket: str, key: str) -> None:
        """
        Delete an object from storage.
        
        Args:
            bucket: Bucket name
            key: Object key (path)
            
        Raises:
            StorageError: If deletion fails
        """
        self._validate_bucket_name(bucket)
        self._validate_object_key(key)

        # TODO: Implement actual gRPC call
        pass

    def list_objects(
        self, bucket: str, prefix: str = "", max_keys: int = 1000
    ) -> Dict:
        """
        List objects in a bucket.
        
        Args:
            bucket: Bucket name
            prefix: Filter objects by prefix
            max_keys: Maximum number of keys to return
            
        Returns:
            dict: Response with list of objects
            
        Raises:
            StorageError: If listing fails
        """
        self._validate_bucket_name(bucket)

        # TODO: Implement actual gRPC call
        return {
            "bucket": bucket,
            "prefix": prefix,
            "objects": [],
            "is_truncated": False,
        }

    def object_exists(self, bucket: str, key: str) -> bool:
        """
        Check if an object exists.
        
        Args:
            bucket: Bucket name
            key: Object key (path)
            
        Returns:
            bool: True if object exists, False otherwise
        """
        try:
            self.head_object(bucket, key)
            return True
        except NotFoundError:
            return False
        except StorageError:
            return False

    def head_object(self, bucket: str, key: str) -> Dict:
        """
        Get object metadata without downloading content.
        
        Args:
            bucket: Bucket name
            key: Object key (path)
            
        Returns:
            dict: Object metadata (size, content_type, etag, etc.)
            
        Raises:
            NotFoundError: If object doesn't exist
            StorageError: If request fails
        """
        self._validate_bucket_name(bucket)
        self._validate_object_key(key)

        # TODO: Implement actual gRPC call
        raise NotFoundError(f"Object not found", bucket=bucket, key=key)

    def upload_file(
        self,
        bucket: str,
        key: str,
        file_path: str,
        content_type: str = None,
        metadata: Dict[str, str] = None,
        chunk_size: int = None,
    ) -> Dict:
        """
        Upload a file from local filesystem.
        
        Args:
            bucket: Bucket name
            key: Object key (destination path in storage)
            file_path: Local file path to upload
            content_type: MIME type (auto-detected if not provided)
            metadata: Custom metadata dictionary
            chunk_size: Chunk size for streaming (default: 5MB)
            
        Returns:
            dict: Response with upload details
            
        Raises:
            FileNotFoundError: If local file doesn't exist
            StorageError: If upload fails
            
        Example:
            ```python
            client.upload_file(
                bucket="my-bucket",
                key="uploads/document.pdf",
                file_path="/home/user/document.pdf",
                metadata={"author": "John Doe"}
            )
            ```
        """
        file_path = Path(file_path)
        if not file_path.exists():
            raise FileNotFoundError(f"File not found: {file_path}")

        if not file_path.is_file():
            raise InvalidParameterError(f"Not a file: {file_path}")

        # Auto-detect content type if not provided
        if content_type is None:
            content_type = self._guess_content_type(str(file_path))

        file_size = file_path.stat().st_size

        # Use streaming for large files
        if file_size > 100 * 1024 * 1024:  # > 100MB
            return self._upload_file_streaming(
                bucket, key, file_path, content_type, metadata, chunk_size
            )
        else:
            # Read entire file for small files
            with open(file_path, "rb") as f:
                data = f.read()
            return self.put_object(bucket, key, data, content_type, metadata)

    def download_file(
        self, bucket: str, key: str, file_path: str, overwrite: bool = False
    ) -> Dict:
        """
        Download an object to local filesystem.
        
        Args:
            bucket: Bucket name
            key: Object key (source path in storage)
            file_path: Local file path to save
            overwrite: Whether to overwrite existing file
            
        Returns:
            dict: Response with download details
            
        Raises:
            FileExistsError: If file exists and overwrite is False
            NotFoundError: If object doesn't exist
            StorageError: If download fails
            
        Example:
            ```python
            client.download_file(
                bucket="my-bucket",
                key="uploads/document.pdf",
                file_path="/home/user/downloaded.pdf",
                overwrite=True
            )
            ```
        """
        file_path = Path(file_path)

        if file_path.exists() and not overwrite:
            raise FileExistsError(
                f"File already exists: {file_path}. Use overwrite=True to replace."
            )

        # Create parent directories if needed
        file_path.parent.mkdir(parents=True, exist_ok=True)

        data = self.get_object(bucket, key)

        with open(file_path, "wb") as f:
            f.write(data)

        return {
            "bucket": bucket,
            "key": key,
            "file_path": str(file_path),
            "size": len(data),
        }

    def upload_from_stream(
        self,
        bucket: str,
        key: str,
        stream: BinaryIO,
        content_type: str = None,
        metadata: Dict[str, str] = None,
        chunk_size: int = None,
    ) -> Dict:
        """
        Upload data from a stream (file-like object).
        
        Args:
            bucket: Bucket name
            key: Object key (path)
            stream: File-like object to read from
            content_type: MIME type
            metadata: Custom metadata dictionary
            chunk_size: Chunk size for reading (default: 5MB)
            
        Returns:
            dict: Response with upload details
            
        Raises:
            StorageError: If upload fails
            
        Example:
            ```python
            with open("large_file.bin", "rb") as f:
                client.upload_from_stream(
                    bucket="my-bucket",
                    key="data/large_file.bin",
                    stream=f
                )
            ```
        """
        self._validate_bucket_name(bucket)
        self._validate_object_key(key)

        chunk_size = chunk_size or self.DEFAULT_CHUNK_SIZE

        # TODO: Implement actual streaming upload via gRPC
        # For now, read all data and upload
        data = stream.read()
        return self.put_object(bucket, key, data, content_type, metadata)

    def _upload_file_streaming(
        self,
        bucket: str,
        key: str,
        file_path: Path,
        content_type: str,
        metadata: Dict[str, str],
        chunk_size: int = None,
    ) -> Dict:
        """Internal method for streaming upload of large files."""
        chunk_size = chunk_size or self.DEFAULT_CHUNK_SIZE

        with open(file_path, "rb") as f:
            return self.upload_from_stream(
                bucket, key, f, content_type, metadata, chunk_size
            )

    def _validate_bucket_name(self, bucket: str) -> None:
        """Validate bucket name according to naming rules."""
        if not bucket:
            raise InvalidParameterError("Bucket name cannot be empty")

        if len(bucket) < 3 or len(bucket) > 63:
            raise InvalidParameterError(
                "Bucket name must be between 3 and 63 characters"
            )

        if not bucket[0].isalnum() or not bucket[-1].isalnum():
            raise InvalidParameterError(
                "Bucket name must start and end with a letter or number"
            )

        # Simple validation - full regex validation can be added if needed
        if not all(c.islower() or c.isdigit() or c == "-" for c in bucket):
            raise InvalidParameterError(
                "Bucket name can only contain lowercase letters, numbers, and hyphens"
            )

    def _validate_object_key(self, key: str) -> None:
        """Validate object key."""
        if not key:
            raise InvalidParameterError("Object key cannot be empty")

        if len(key) > 1024:
            raise InvalidParameterError("Object key exceeds maximum length of 1024")

        if key.startswith("/"):
            raise InvalidParameterError("Object key should not start with /")

    def _guess_content_type(self, filename: str) -> str:
        """Guess content type from filename."""
        content_type, _ = mimetypes.guess_type(filename)
        return content_type or "application/octet-stream"

    def close(self):
        """Close the gRPC channel."""
        # TODO: Close channel when implemented
        # if hasattr(self, '_channel'):
        #     self._channel.close()
        pass

    def __enter__(self):
        """Context manager entry."""
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit."""
        self.close()

