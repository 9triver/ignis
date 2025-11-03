// Package storage provides a lightweight, extensible interface for object storage operations.
//
// This package defines a set of interfaces and types that abstract common object storage
// operations, making it easy to integrate with S3-compatible services, cloud storage providers,
// or custom storage backends.
//
// # Core Interfaces
//
// The package defines several interfaces with increasing capabilities:
//
//   - ObjectStorage: Basic object operations (Put, Get, Delete, List)
//   - StreamStorage: Streaming operations for large objects
//   - BucketManager: Bucket management operations
//   - PresignedURLGenerator: Generate pre-signed URLs for direct access
//   - MultipartUpload: Multipart upload support for very large objects
//   - FullStorage: Combines all interfaces
//
// # Integration with ignis
//
// The storage package is designed to work seamlessly with ignis's object system:
//
//   - Supports ignis Stream objects for efficient data transfer between actors
//   - Integrates with ignis error handling and logging
//   - Compatible with ignis's serialization formats (JSON, Go, Python)
//
// # Example Usage
//
// Basic object operations:
//
//	storage, _ := s3.NewS3Storage(&s3.S3Config{
//	    Endpoint: "s3.amazonaws.com",
//	    Region: "us-east-1",
//	})
//
//	// Upload an object
//	storage.PutObject(ctx, &storage.PutObjectRequest{
//	    Bucket: "my-bucket",
//	    Key: "path/to/object",
//	    Data: []byte("hello world"),
//	})
//
//	// Download an object
//	resp, _ := storage.GetObject(ctx, &storage.GetObjectRequest{
//	    Bucket: "my-bucket",
//	    Key: "path/to/object",
//	})
//	fmt.Println(string(resp.Data))
//
// Streaming operations:
//
//	// Upload from ignis Stream
//	stream := object.NewStream(dataChannel, object.LangGo)
//	streamStorage.(storage.StreamStorage).PutIgnisStream(ctx, &storage.PutIgnisStreamRequest{
//	    Bucket: "my-bucket",
//	    Key: "large-file.bin",
//	    Stream: stream,
//	})
//
//	// Download as ignis Stream
//	stream, _ := streamStorage.(storage.StreamStorage).GetIgnisStream(ctx, &storage.GetObjectRequest{
//	    Bucket: "my-bucket",
//	    Key: "large-file.bin",
//	})
//
// # Error Handling
//
// The package defines a comprehensive error system with specific error codes:
//
//	resp, err := storage.GetObject(ctx, req)
//	if err != nil {
//	    if storage.IsNotFoundError(err) {
//	        // Object doesn't exist
//	    } else if storage.IsAccessError(err) {
//	        // Permission denied
//	    } else if storage.IsRetryable(err) {
//	        // Can retry the operation
//	    }
//	}
//
// # Implementation Guidelines
//
// When implementing the storage interfaces:
//
//  1. Support at least the ObjectStorage interface
//  2. Implement StreamStorage for better performance with large objects
//  3. Use the predefined error types and codes
//  4. Validate inputs using the provided utility functions
//  5. Support context cancellation for long-running operations
//  6. Add appropriate logging using the ignis logging system
//
// # Thread Safety
//
// All operations should be thread-safe and support concurrent access.
// Implementations should handle concurrent requests properly.
package storage
