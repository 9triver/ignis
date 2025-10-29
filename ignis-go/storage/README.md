# Ignis Storage Package

**English** | [中文](README_zh.md)

A lightweight, extensible interface for object storage operations in the ignis framework.

## Overview

The `storage` package provides a clean abstraction layer for object storage, making it easy to integrate with S3-compatible services, cloud storage providers, or custom storage backends. It is designed to work seamlessly with ignis's object system and streaming capabilities.

## Features

- 🎯 **Simple and Intuitive API** - Clean interfaces for common storage operations
- 🔄 **Streaming Support** - Efficient handling of large objects with ignis Stream integration
- 🔌 **Extensible** - Easy to implement for different storage backends
- 📦 **Multipart Upload** - Support for uploading very large objects
- 🔗 **Pre-signed URLs** - Generate temporary URLs for direct client access
- ⚡ **Context-aware** - Support for cancellation and timeouts
- 🛡️ **Type-safe Errors** - Comprehensive error handling with specific error codes

## Interfaces

### ObjectStorage (Core)

Basic object storage operations:

```go
type ObjectStorage interface {
    PutObject(ctx context.Context, req *PutObjectRequest) (*PutObjectResponse, error)
    GetObject(ctx context.Context, req *GetObjectRequest) (*GetObjectResponse, error)
    DeleteObject(ctx context.Context, req *DeleteObjectRequest) error
    ListObjects(ctx context.Context, req *ListObjectsRequest) (*ListObjectsResponse, error)
    HeadObject(ctx context.Context, req *HeadObjectRequest) (*ObjectMetadata, error)
    CopyObject(ctx context.Context, req *CopyObjectRequest) error
    ObjectExists(ctx context.Context, bucket, key string) (bool, error)
}
```

### StreamStorage

Extended streaming capabilities:

```go
type StreamStorage interface {
    ObjectStorage
    
    // Stream operations
    PutStream(ctx context.Context, req *PutStreamRequest) (*PutObjectResponse, error)
    GetStream(ctx context.Context, req *GetObjectRequest) (io.ReadCloser, *ObjectMetadata, error)
    
    // Ignis Stream integration
    PutIgnisStream(ctx context.Context, req *PutIgnisStreamRequest) (*PutObjectResponse, error)
    GetIgnisStream(ctx context.Context, req *GetObjectRequest) (objects.Interface, error)
}
```

### BucketManager

Bucket management operations:

```go
type BucketManager interface {
    CreateBucket(ctx context.Context, bucket string, opts *BucketOptions) error
    DeleteBucket(ctx context.Context, bucket string) error
    ListBuckets(ctx context.Context) ([]BucketInfo, error)
    BucketExists(ctx context.Context, bucket string) (bool, error)
    GetBucketInfo(ctx context.Context, bucket string) (*BucketInfo, error)
}
```

### MultipartUpload

Multipart upload for large files:

```go
type MultipartUpload interface {
    InitiateMultipartUpload(ctx context.Context, req *InitiateMultipartRequest) (*MultipartUploadInfo, error)
    UploadPart(ctx context.Context, req *UploadPartRequest) (*UploadPartResponse, error)
    CompleteMultipartUpload(ctx context.Context, req *CompleteMultipartRequest) (*PutObjectResponse, error)
    AbortMultipartUpload(ctx context.Context, req *AbortMultipartRequest) error
    ListParts(ctx context.Context, req *ListPartsRequest) (*ListPartsResponse, error)
}
```

## Usage Examples

### Basic Operations

```go
import (
    "context"
    "github.com/9triver/ignis/storage"
    "github.com/your-org/iarnet/internal/storage/s3"
)

// Initialize storage (implementation-specific)
storage, err := s3.NewS3Storage(&s3.S3Config{
    Endpoint:  "s3.amazonaws.com",
    Region:    "us-east-1",
    AccessKey: "YOUR_ACCESS_KEY",
    SecretKey: "YOUR_SECRET_KEY",
})

// Upload an object
resp, err := storage.PutObject(context.Background(), &storage.PutObjectRequest{
    Bucket:      "my-bucket",
    Key:         "data/file.txt",
    Data:        []byte("Hello, World!"),
    ContentType: "text/plain",
    Metadata: map[string]string{
        "author": "ignis",
    },
})

// Download an object
obj, err := storage.GetObject(context.Background(), &storage.GetObjectRequest{
    Bucket: "my-bucket",
    Key:    "data/file.txt",
})
fmt.Println(string(obj.Data))

// List objects
list, err := storage.ListObjects(context.Background(), &storage.ListObjectsRequest{
    Bucket: "my-bucket",
    Prefix: "data/",
})
for _, obj := range list.Objects {
    fmt.Printf("%s (%d bytes)\n", obj.Key, obj.Size)
}

// Delete an object
err = storage.DeleteObject(context.Background(), &storage.DeleteObjectRequest{
    Bucket: "my-bucket",
    Key:    "data/file.txt",
})
```

### Streaming Operations

```go
// Upload from io.Reader
file, _ := os.Open("large-file.bin")
defer file.Close()

streamStorage := storage.(storage.StreamStorage)
_, err := streamStorage.PutStream(context.Background(), &storage.PutStreamRequest{
    Bucket:      "my-bucket",
    Key:         "uploads/large-file.bin",
    Reader:      file,
    Size:        fileInfo.Size(),
    ContentType: "application/octet-stream",
})

// Download as stream
reader, metadata, err := streamStorage.GetStream(context.Background(), &storage.GetObjectRequest{
    Bucket: "my-bucket",
    Key:    "uploads/large-file.bin",
})
defer reader.Close()

// Process stream...
io.Copy(outputFile, reader)
```

### Ignis Stream Integration

```go
import "github.com/9triver/ignis/objects"

// Create ignis Stream from data source
dataChannel := make(chan objects.Interface, 10)
go func() {
    defer close(dataChannel)
    for i := 0; i < 100; i++ {
        data := fmt.Sprintf("chunk-%d", i)
        dataChannel <- objects.NewLocal([]byte(data), objects.LangGo)
    }
}()

stream := objects.NewStream(dataChannel, objects.LangGo)

// Upload ignis Stream
streamStorage := storage.(storage.StreamStorage)
_, err := streamStorage.PutIgnisStream(context.Background(), &storage.PutIgnisStreamRequest{
    Bucket: "my-bucket",
    Key:    "streams/data.bin",
    Stream: stream,
})

// Download as ignis Stream
stream, err := streamStorage.GetIgnisStream(context.Background(), &storage.GetObjectRequest{
    Bucket: "my-bucket",
    Key:    "streams/data.bin",
})

// Use in actor function
result := myActorFunction(stream)
```

### Error Handling

```go
obj, err := storage.GetObject(ctx, req)
if err != nil {
    if storage.IsNotFoundError(err) {
        log.Println("Object doesn't exist")
    } else if storage.IsAccessError(err) {
        log.Println("Permission denied")
    } else if storage.IsNetworkError(err) {
        log.Println("Network error, retrying...")
        // Retry logic
    } else if storage.IsRetryable(err) {
        log.Println("Temporary error, can retry")
    } else {
        log.Printf("Fatal error: %v", err)
    }
}

// Access error details
if se, ok := err.(*storage.StorageError); ok {
    log.Printf("Error code: %s, bucket: %s, key: %s", se.Code, se.Bucket, se.Key)
}
```

### Utility Functions

```go
import "github.com/9triver/ignis/storage"

// Validate bucket name
err := storage.ValidateBucketName("my-bucket")

// Validate object key
err = storage.ValidateObjectKey("path/to/file.txt")

// Normalize key
key := storage.NormalizeKey("//path///to//file.txt") // "path/to/file.txt"

// Join path segments
key = storage.JoinKey("folder", "subfolder", "file.txt") // "folder/subfolder/file.txt"

// Guess content type
contentType := storage.GuessContentType("image.jpg") // "image/jpeg"

// Calculate optimal part size for multipart upload
partSize := storage.CalculatePartSize(5 * 1024 * 1024 * 1024) // For 5GB file

// Check if should use multipart
useMultipart := storage.ShouldUseMultipart(150 * 1024 * 1024) // true for 150MB
```

## Implementation Guidelines

When implementing the storage interfaces for a specific backend:

1. **Start with ObjectStorage** - Implement the core interface first
2. **Add StreamStorage** - For better performance with large objects
3. **Use predefined errors** - Return appropriate StorageError types
4. **Validate inputs** - Use utility functions like `ValidateBucketName`
5. **Support context** - Handle cancellation properly
6. **Add logging** - Use ignis logging system
7. **Thread safety** - Ensure concurrent access is safe

### Example Implementation Structure

```go
package s3

import "github.com/9triver/ignis/storage"

type S3Storage struct {
    client *s3.Client
    // ... config fields
}

func NewS3Storage(config *S3Config) (*S3Storage, error) {
    // Initialize S3 client
}

// Implement storage.ObjectStorage
func (s *S3Storage) PutObject(ctx context.Context, req *storage.PutObjectRequest) (*storage.PutObjectResponse, error) {
    // Validate inputs
    if err := storage.ValidateBucketName(req.Bucket); err != nil {
        return nil, err
    }
    if err := storage.ValidateObjectKey(req.Key); err != nil {
        return nil, err
    }
    
    // Call S3 API
    // Handle errors with storage.WrapStorageError
    // Return response
}

// Implement other interfaces...
```

## Error Codes Reference

| Code | Description | Retryable |
|------|-------------|-----------|
| `ObjectNotFound` | Object doesn't exist | No |
| `ObjectAlreadyExists` | Object already exists | No |
| `BucketNotFound` | Bucket doesn't exist | No |
| `PermissionDenied` | Access denied | No |
| `NetworkError` | Network failure | Yes |
| `Timeout` | Operation timeout | Yes |
| `InternalError` | Internal error | Yes |

See `errors.go` for the complete list of error codes.

## Best Practices

1. **Always use context** - Pass proper context for timeout/cancellation
2. **Validate inputs early** - Use utility functions before making API calls
3. **Handle errors properly** - Check error types and provide meaningful messages
4. **Use streaming for large files** - Don't load large files entirely into memory
5. **Set appropriate content types** - Use `GuessContentType` or set explicitly
6. **Add metadata** - Include useful metadata for debugging and tracking
7. **Clean up resources** - Close readers/streams when done

## Thread Safety

All interfaces are designed to be thread-safe. Implementations should support concurrent operations.

## Performance Considerations

- Use `StreamStorage` for objects > 100MB
- Use multipart upload for objects > 100MB (automatic in most implementations)
- Set appropriate chunk sizes based on network and object size
- Consider pre-signed URLs for direct client uploads/downloads
- Use connection pooling in implementations

## License

Part of the ignis framework.

