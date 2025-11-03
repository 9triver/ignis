package storage

import (
	"context"
	"io"

	"github.com/9triver/ignis/object"
)

// ObjectStorage defines the core interface for object storage operations.
// Implementations should support S3-compatible protocols or other cloud storage services.
type ObjectStorage interface {
	// PutObject uploads an object to storage
	PutObject(ctx context.Context, req *PutObjectRequest) (*PutObjectResponse, error)

	// GetObject downloads an object from storage
	GetObject(ctx context.Context, req *GetObjectRequest) (*GetObjectResponse, error)

	// DeleteObject removes an object from storage
	DeleteObject(ctx context.Context, req *DeleteObjectRequest) error

	// ListObjects lists objects with a given prefix
	ListObjects(ctx context.Context, req *ListObjectsRequest) (*ListObjectsResponse, error)

	// HeadObject retrieves metadata of an object without downloading its content
	HeadObject(ctx context.Context, req *HeadObjectRequest) (*ObjectMetadata, error)

	// CopyObject copies an object from source to destination
	CopyObject(ctx context.Context, req *CopyObjectRequest) error

	// ObjectExists checks if an object exists
	ObjectExists(ctx context.Context, bucket, key string) (bool, error)
}

// StreamStorage extends ObjectStorage with streaming capabilities.
// This is useful for handling large objects that don't fit in memory.
type StreamStorage interface {
	ObjectStorage

	// PutStream uploads data from an io.Reader
	// Useful for streaming large files or data from ignis objects
	PutStream(ctx context.Context, req *PutStreamRequest) (*PutObjectResponse, error)

	// GetStream downloads an object and returns an io.ReadCloser
	// Caller is responsible for closing the reader
	GetStream(ctx context.Context, req *GetObjectRequest) (io.ReadCloser, *ObjectMetadata, error)

	// PutIgnisStream uploads data from an ignis Stream object
	// Integrates directly with ignis's streaming system
	PutIgnisStream(ctx context.Context, req *PutIgnisStreamRequest) (*PutObjectResponse, error)

	// GetIgnisStream downloads an object and returns an ignis Stream
	// Useful for passing data between actors
	GetIgnisStream(ctx context.Context, req *GetObjectRequest) (object.Interface, error)
}

// BucketManager defines operations for managing storage buckets.
type BucketManager interface {
	// CreateBucket creates a new bucket
	CreateBucket(ctx context.Context, bucket string, opts *BucketOptions) error

	// DeleteBucket deletes an empty bucket
	DeleteBucket(ctx context.Context, bucket string) error

	// ListBuckets lists all buckets
	ListBuckets(ctx context.Context) ([]BucketInfo, error)

	// BucketExists checks if a bucket exists
	BucketExists(ctx context.Context, bucket string) (bool, error)

	// GetBucketInfo retrieves bucket information
	GetBucketInfo(ctx context.Context, bucket string) (*BucketInfo, error)
}

// PresignedURLGenerator generates pre-signed URLs for direct client access.
// This is optional and may not be supported by all implementations.
type PresignedURLGenerator interface {
	// GeneratePresignedGetURL generates a URL for downloading an object
	GeneratePresignedGetURL(ctx context.Context, req *PresignedURLRequest) (string, error)

	// GeneratePresignedPutURL generates a URL for uploading an object
	GeneratePresignedPutURL(ctx context.Context, req *PresignedURLRequest) (string, error)
}

// MultipartUpload defines operations for multipart upload.
// This is useful for uploading very large objects in parts.
type MultipartUpload interface {
	// InitiateMultipartUpload starts a multipart upload
	InitiateMultipartUpload(ctx context.Context, req *InitiateMultipartRequest) (*MultipartUploadInfo, error)

	// UploadPart uploads a part of a multipart upload
	UploadPart(ctx context.Context, req *UploadPartRequest) (*UploadPartResponse, error)

	// CompleteMultipartUpload completes a multipart upload
	CompleteMultipartUpload(ctx context.Context, req *CompleteMultipartRequest) (*PutObjectResponse, error)

	// AbortMultipartUpload aborts a multipart upload
	AbortMultipartUpload(ctx context.Context, req *AbortMultipartRequest) error

	// ListParts lists uploaded parts
	ListParts(ctx context.Context, req *ListPartsRequest) (*ListPartsResponse, error)
}

// FullStorage combines all storage interfaces.
// Implementations can choose which interfaces to support.
type FullStorage interface {
	ObjectStorage
	StreamStorage
	BucketManager
	PresignedURLGenerator
	MultipartUpload
}
