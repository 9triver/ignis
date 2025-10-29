package storage

import (
	"context"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"

	"github.com/9triver/ignis/objects"
)

const (
	// DefaultChunkSize is the default chunk size for streaming operations (5MB)
	DefaultChunkSize = 5 * 1024 * 1024

	// MaxObjectKeyLength is the maximum allowed length for object keys
	MaxObjectKeyLength = 1024

	// MinMultipartChunkSize is the minimum chunk size for multipart uploads (5MB)
	MinMultipartChunkSize = 5 * 1024 * 1024

	// MaxMultipartChunkSize is the maximum chunk size for multipart uploads (5GB)
	MaxMultipartChunkSize = 5 * 1024 * 1024 * 1024

	// MaxMultipartParts is the maximum number of parts in a multipart upload
	MaxMultipartParts = 10000
)

var (
	// bucketNameRegex validates bucket names
	// Rules: 3-63 characters, lowercase letters, numbers, hyphens
	bucketNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)

	// invalidKeyChars contains characters that are not allowed in object keys
	invalidKeyChars = []string{"\x00", "\n", "\r"}
)

// ValidateBucketName validates a bucket name according to S3 naming rules.
func ValidateBucketName(bucket string) error {
	if bucket == "" {
		return NewStorageError(ErrCodeInvalidBucketName, "bucket name cannot be empty")
	}

	if len(bucket) < 3 || len(bucket) > 63 {
		return NewStorageError(ErrCodeInvalidBucketName, "bucket name must be between 3 and 63 characters")
	}

	if !bucketNameRegex.MatchString(bucket) {
		return NewStorageError(ErrCodeInvalidBucketName, "bucket name must contain only lowercase letters, numbers, and hyphens")
	}

	// Cannot start or end with a hyphen
	if strings.HasPrefix(bucket, "-") || strings.HasSuffix(bucket, "-") {
		return NewStorageError(ErrCodeInvalidBucketName, "bucket name cannot start or end with a hyphen")
	}

	// Cannot contain consecutive hyphens
	if strings.Contains(bucket, "--") {
		return NewStorageError(ErrCodeInvalidBucketName, "bucket name cannot contain consecutive hyphens")
	}

	// Cannot be formatted as an IP address
	if regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`).MatchString(bucket) {
		return NewStorageError(ErrCodeInvalidBucketName, "bucket name cannot be formatted as an IP address")
	}

	return nil
}

// ValidateObjectKey validates an object key.
func ValidateObjectKey(key string) error {
	if key == "" {
		return NewStorageError(ErrCodeInvalidObjectKey, "object key cannot be empty")
	}

	if len(key) > MaxObjectKeyLength {
		return NewStorageError(ErrCodeInvalidObjectKey, fmt.Sprintf("object key exceeds maximum length of %d", MaxObjectKeyLength))
	}

	// Check for invalid characters
	for _, char := range invalidKeyChars {
		if strings.Contains(key, char) {
			return NewStorageError(ErrCodeInvalidObjectKey, fmt.Sprintf("object key contains invalid character: %q", char))
		}
	}

	// Keys should not start with /
	if strings.HasPrefix(key, "/") {
		return NewStorageError(ErrCodeInvalidObjectKey, "object key should not start with /")
	}

	return nil
}

// NormalizeKey normalizes an object key by removing redundant slashes.
func NormalizeKey(key string) string {
	// Remove leading slash
	key = strings.TrimPrefix(key, "/")

	// Replace multiple consecutive slashes with a single slash
	for strings.Contains(key, "//") {
		key = strings.ReplaceAll(key, "//", "/")
	}

	// Clean the path
	key = path.Clean(key)

	// path.Clean converts "." to ".", so we need to handle that
	if key == "." {
		return ""
	}

	return key
}

// JoinKey joins path segments into a single object key.
func JoinKey(segments ...string) string {
	key := path.Join(segments...)
	return NormalizeKey(key)
}

// SplitKey splits an object key into directory and filename.
func SplitKey(key string) (dir, file string) {
	dir, file = path.Split(key)
	return strings.TrimSuffix(dir, "/"), file
}

// GetExtension returns the file extension from an object key.
func GetExtension(key string) string {
	return path.Ext(key)
}

// GetBasename returns the filename without extension from an object key.
func GetBasename(key string) string {
	_, file := path.Split(key)
	ext := path.Ext(file)
	return strings.TrimSuffix(file, ext)
}

// IsDirectory checks if a key represents a directory (ends with /).
func IsDirectory(key string) bool {
	return strings.HasSuffix(key, "/")
}

// EnsureDirectoryKey ensures a key ends with / to represent a directory.
func EnsureDirectoryKey(key string) string {
	if !IsDirectory(key) {
		return key + "/"
	}
	return key
}

// GuessContentType attempts to guess the content type based on file extension.
func GuessContentType(key string) string {
	ext := strings.ToLower(GetExtension(key))

	contentTypes := map[string]string{
		".txt":  "text/plain",
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",
		".pdf":  "application/pdf",
		".zip":  "application/zip",
		".tar":  "application/x-tar",
		".gz":   "application/gzip",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
		".mp4":  "video/mp4",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
	}

	if contentType, ok := contentTypes[ext]; ok {
		return contentType
	}

	return "application/octet-stream"
}

// CalculatePartSize calculates an appropriate part size for multipart upload.
// It ensures the part size is between min and max limits and results in a reasonable number of parts.
func CalculatePartSize(totalSize int64) int64 {
	if totalSize <= 0 {
		return DefaultChunkSize
	}

	// Start with default chunk size
	partSize := int64(DefaultChunkSize)

	// Calculate number of parts with current part size
	numParts := (totalSize + partSize - 1) / partSize

	// If we would need more than max parts, increase part size
	if numParts > MaxMultipartParts {
		partSize = (totalSize + MaxMultipartParts - 1) / MaxMultipartParts
	}

	// Ensure part size is at least the minimum
	if partSize < MinMultipartChunkSize {
		partSize = MinMultipartChunkSize
	}

	// Ensure part size doesn't exceed the maximum
	if partSize > MaxMultipartChunkSize {
		partSize = MaxMultipartChunkSize
	}

	return partSize
}

// ShouldUseMultipart determines if multipart upload should be used based on object size.
func ShouldUseMultipart(size int64) bool {
	// Use multipart for objects larger than 100MB
	const multipartThreshold = 100 * 1024 * 1024
	return size > multipartThreshold
}

// StreamToReader converts an ignis Stream to an io.Reader.
// This is useful for integrating ignis streams with standard Go I/O operations.
func StreamToReader(stream objects.Interface) (io.Reader, error) {
	if stream == nil {
		return nil, NewStorageError(ErrCodeInvalidParameter, "stream is nil")
	}

	// Check if it's actually a stream
	value, err := stream.Value()
	if err != nil {
		return nil, WrapStorageError(ErrCodeStreamReadError, "failed to get stream value", err)
	}

	// If the value is already a channel, create a reader from it
	ch, ok := value.(<-chan objects.Interface)
	if !ok {
		return nil, NewStorageError(ErrCodeInvalidParameter, "stream value is not a channel")
	}

	return &streamReader{ch: ch}, nil
}

// streamReader implements io.Reader for ignis streams.
type streamReader struct {
	ch      <-chan objects.Interface
	current []byte
	offset  int
	closed  bool
}

func (r *streamReader) Read(p []byte) (n int, err error) {
	if r.closed {
		return 0, io.EOF
	}

	for {
		// If we have data in current buffer, copy it
		if r.offset < len(r.current) {
			n = copy(p, r.current[r.offset:])
			r.offset += n
			return n, nil
		}

		// Try to read next chunk from stream
		obj, ok := <-r.ch
		if !ok {
			r.closed = true
			return 0, io.EOF
		}

		// Extract data from object
		value, err := obj.Value()
		if err != nil {
			return 0, WrapStorageError(ErrCodeStreamReadError, "failed to read from stream", err)
		}

		// Convert to bytes
		var data []byte
		switch v := value.(type) {
		case []byte:
			data = v
		case string:
			data = []byte(v)
		default:
			return 0, NewStorageError(ErrCodeStreamReadError, fmt.Sprintf("unsupported stream data type: %T", value))
		}

		r.current = data
		r.offset = 0
	}
}

// ReaderToIgnisStream converts an io.Reader to an ignis Stream.
// It reads from the reader in chunks and creates a stream of objects.
func ReaderToIgnisStream(ctx context.Context, reader io.Reader, chunkSize int, language objects.Language) objects.Interface {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}

	ch := make(chan objects.Interface, 10)

	go func() {
		defer close(ch)

		buffer := make([]byte, chunkSize)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := reader.Read(buffer)
				if n > 0 {
					// Create a copy of the data
					data := make([]byte, n)
					copy(data, buffer[:n])

					// Send as ignis object
					obj := objects.NewLocal(data, language)
					select {
					case ch <- obj:
					case <-ctx.Done():
						return
					}
				}

				if err == io.EOF {
					return
				}
				if err != nil {
					// TODO: handle error properly
					return
				}
			}
		}
	}()

	return objects.NewStream(ch, language)
}

// CopyObject is a helper function to copy an object within the same storage.
func CopyObject(ctx context.Context, storage ObjectStorage, req *CopyObjectRequest) error {
	if req.SourceBucket == "" || req.SourceKey == "" {
		return NewStorageError(ErrCodeInvalidParameter, "source bucket and key are required")
	}
	if req.DestinationBucket == "" || req.DestinationKey == "" {
		return NewStorageError(ErrCodeInvalidParameter, "destination bucket and key are required")
	}

	return storage.CopyObject(ctx, req)
}

// MoveObject moves an object by copying and then deleting the source.
func MoveObject(ctx context.Context, storage ObjectStorage, req *CopyObjectRequest) error {
	// First copy the object
	if err := CopyObject(ctx, storage, req); err != nil {
		return err
	}

	// Then delete the source
	return storage.DeleteObject(ctx, &DeleteObjectRequest{
		Bucket: req.SourceBucket,
		Key:    req.SourceKey,
	})
}
