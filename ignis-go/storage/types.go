package storage

import (
	"io"
	"time"

	"github.com/9triver/ignis/objects"
)

// PutObjectRequest represents a request to upload an object.
type PutObjectRequest struct {
	Bucket      string            // Bucket name
	Key         string            // Object key (path)
	Data        []byte            // Object data
	ContentType string            // MIME type (e.g., "application/json")
	Metadata    map[string]string // Custom metadata
	Tags        map[string]string // Object tags
}

// PutObjectResponse represents the response from uploading an object.
type PutObjectResponse struct {
	Bucket     string    // Bucket name
	Key        string    // Object key
	ETag       string    // Entity tag (MD5 hash)
	VersionID  string    // Version ID (if versioning is enabled)
	Size       int64     // Size of uploaded object in bytes
	UploadedAt time.Time // Upload timestamp
}

// GetObjectRequest represents a request to download an object.
type GetObjectRequest struct {
	Bucket    string // Bucket name
	Key       string // Object key
	VersionID string // Optional: specific version to retrieve
}

// GetObjectResponse represents the response from downloading an object.
type GetObjectResponse struct {
	Bucket       string            // Bucket name
	Key          string            // Object key
	Data         []byte            // Object data
	ContentType  string            // MIME type
	Size         int64             // Size in bytes
	ETag         string            // Entity tag
	LastModified time.Time         // Last modification time
	Metadata     map[string]string // Custom metadata
	VersionID    string            // Version ID (if versioning is enabled)
}

// DeleteObjectRequest represents a request to delete an object.
type DeleteObjectRequest struct {
	Bucket    string // Bucket name
	Key       string // Object key
	VersionID string // Optional: specific version to delete
}

// ListObjectsRequest represents a request to list objects.
type ListObjectsRequest struct {
	Bucket            string // Bucket name
	Prefix            string // Filter objects by prefix
	Delimiter         string // Delimiter for grouping (e.g., "/")
	MaxKeys           int    // Maximum number of keys to return (default: 1000)
	StartAfter        string // Start listing after this key
	ContinuationToken string // Token for pagination
}

// ListObjectsResponse represents the response from listing objects.
type ListObjectsResponse struct {
	Bucket                string       // Bucket name
	Prefix                string       // Prefix used for filtering
	Objects               []ObjectInfo // List of objects
	CommonPrefixes        []string     // Common prefixes (folders)
	IsTruncated           bool         // Whether there are more results
	NextContinuationToken string       // Token for next page
	KeyCount              int          // Number of keys in this response
}

// ObjectInfo represents basic information about an object.
type ObjectInfo struct {
	Key          string    // Object key
	Size         int64     // Size in bytes
	ETag         string    // Entity tag
	LastModified time.Time // Last modification time
	StorageClass string    // Storage class (e.g., "STANDARD", "GLACIER")
	Owner        string    // Owner ID
}

// HeadObjectRequest represents a request to get object metadata.
type HeadObjectRequest struct {
	Bucket    string // Bucket name
	Key       string // Object key
	VersionID string // Optional: specific version
}

// ObjectMetadata represents metadata of an object.
type ObjectMetadata struct {
	Bucket       string            // Bucket name
	Key          string            // Object key
	Size         int64             // Size in bytes
	ContentType  string            // MIME type
	ETag         string            // Entity tag
	LastModified time.Time         // Last modification time
	VersionID    string            // Version ID
	Metadata     map[string]string // Custom metadata
	Tags         map[string]string // Object tags
	StorageClass string            // Storage class
}

// CopyObjectRequest represents a request to copy an object.
type CopyObjectRequest struct {
	SourceBucket      string            // Source bucket
	SourceKey         string            // Source key
	DestinationBucket string            // Destination bucket
	DestinationKey    string            // Destination key
	Metadata          map[string]string // Optional: new metadata (replaces source)
	MetadataDirective string            // "COPY" or "REPLACE"
}

// PutStreamRequest represents a request to upload from a stream.
type PutStreamRequest struct {
	Bucket      string            // Bucket name
	Key         string            // Object key
	Reader      io.Reader         // Data source
	Size        int64             // Total size (optional, -1 for unknown)
	ContentType string            // MIME type
	Metadata    map[string]string // Custom metadata
}

// PutIgnisStreamRequest represents a request to upload from an ignis Stream.
type PutIgnisStreamRequest struct {
	Bucket      string            // Bucket name
	Key         string            // Object key
	Stream      objects.Interface // ignis Stream object
	ContentType string            // MIME type
	Metadata    map[string]string // Custom metadata
	ChunkSize   int               // Size of each chunk (default: 5MB)
}

// BucketOptions represents options for creating a bucket.
type BucketOptions struct {
	Region     string            // Region (e.g., "us-east-1")
	ACL        string            // Access control list (e.g., "private", "public-read")
	Versioning bool              // Enable versioning
	Tags       map[string]string // Bucket tags
}

// BucketInfo represents information about a bucket.
type BucketInfo struct {
	Name         string    // Bucket name
	CreationDate time.Time // Creation timestamp
	Region       string    // Region
	Owner        string    // Owner ID
	Versioning   bool      // Whether versioning is enabled
}

// PresignedURLRequest represents a request to generate a pre-signed URL.
type PresignedURLRequest struct {
	Bucket      string        // Bucket name
	Key         string        // Object key
	Expiration  time.Duration // URL expiration duration
	Method      string        // HTTP method ("GET", "PUT", etc.)
	ContentType string        // Content type (for PUT requests)
}

// InitiateMultipartRequest represents a request to initiate a multipart upload.
type InitiateMultipartRequest struct {
	Bucket      string            // Bucket name
	Key         string            // Object key
	ContentType string            // MIME type
	Metadata    map[string]string // Custom metadata
}

// MultipartUploadInfo represents information about an initiated multipart upload.
type MultipartUploadInfo struct {
	Bucket   string // Bucket name
	Key      string // Object key
	UploadID string // Upload ID for tracking
}

// UploadPartRequest represents a request to upload a part.
type UploadPartRequest struct {
	Bucket     string    // Bucket name
	Key        string    // Object key
	UploadID   string    // Upload ID
	PartNumber int       // Part number (1-10000)
	Data       []byte    // Part data
	Size       int64     // Part size
	Reader     io.Reader // Alternative to Data: stream reader
}

// UploadPartResponse represents the response from uploading a part.
type UploadPartResponse struct {
	PartNumber int    // Part number
	ETag       string // Entity tag for this part
	Size       int64  // Part size
}

// CompleteMultipartRequest represents a request to complete a multipart upload.
type CompleteMultipartRequest struct {
	Bucket   string          // Bucket name
	Key      string          // Object key
	UploadID string          // Upload ID
	Parts    []CompletedPart // List of completed parts
}

// CompletedPart represents a completed part in a multipart upload.
type CompletedPart struct {
	PartNumber int    // Part number
	ETag       string // Entity tag
}

// AbortMultipartRequest represents a request to abort a multipart upload.
type AbortMultipartRequest struct {
	Bucket   string // Bucket name
	Key      string // Object key
	UploadID string // Upload ID
}

// ListPartsRequest represents a request to list uploaded parts.
type ListPartsRequest struct {
	Bucket           string // Bucket name
	Key              string // Object key
	UploadID         string // Upload ID
	MaxParts         int    // Maximum parts to return
	PartNumberMarker int    // Start after this part number
}

// ListPartsResponse represents the response from listing parts.
type ListPartsResponse struct {
	Bucket               string     // Bucket name
	Key                  string     // Object key
	UploadID             string     // Upload ID
	Parts                []PartInfo // List of parts
	IsTruncated          bool       // Whether there are more parts
	NextPartNumberMarker int        // Marker for next page
}

// PartInfo represents information about an uploaded part.
type PartInfo struct {
	PartNumber   int       // Part number
	Size         int64     // Part size
	ETag         string    // Entity tag
	LastModified time.Time // Upload time
}
