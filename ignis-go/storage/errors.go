package storage

import (
	"fmt"
)

// Error codes for storage operations
const (
	// Object errors
	ErrCodeObjectNotFound      = "ObjectNotFound"
	ErrCodeObjectAlreadyExists = "ObjectAlreadyExists"
	ErrCodeInvalidObjectKey    = "InvalidObjectKey"
	ErrCodeObjectTooLarge      = "ObjectTooLarge"

	// Bucket errors
	ErrCodeBucketNotFound      = "BucketNotFound"
	ErrCodeBucketAlreadyExists = "BucketAlreadyExists"
	ErrCodeBucketNotEmpty      = "BucketNotEmpty"
	ErrCodeInvalidBucketName   = "InvalidBucketName"

	// Access errors
	ErrCodePermissionDenied = "PermissionDenied"
	ErrCodeAccessDenied     = "AccessDenied"
	ErrCodeUnauthorized     = "Unauthorized"

	// Network errors
	ErrCodeNetworkError     = "NetworkError"
	ErrCodeConnectionFailed = "ConnectionFailed"
	ErrCodeTimeout          = "Timeout"

	// Operation errors
	ErrCodeInvalidRequest   = "InvalidRequest"
	ErrCodeInvalidParameter = "InvalidParameter"
	ErrCodeOperationFailed  = "OperationFailed"
	ErrCodeNotImplemented   = "NotImplemented"
	ErrCodeInternalError    = "InternalError"

	// Stream errors
	ErrCodeStreamReadError  = "StreamReadError"
	ErrCodeStreamWriteError = "StreamWriteError"
	ErrCodeStreamClosed     = "StreamClosed"

	// Multipart upload errors
	ErrCodeInvalidUploadID   = "InvalidUploadID"
	ErrCodeInvalidPartNumber = "InvalidPartNumber"
	ErrCodeInvalidPartOrder  = "InvalidPartOrder"
	ErrCodeMultipartNotFound = "MultipartNotFound"
)

// StorageError represents a storage operation error.
type StorageError struct {
	Code    string // Error code (see constants above)
	Message string // Human-readable error message
	Bucket  string // Bucket name (if applicable)
	Key     string // Object key (if applicable)
	Cause   error  // Underlying error
}

// Error implements the error interface.
func (e *StorageError) Error() string {
	if e.Bucket != "" && e.Key != "" {
		return fmt.Sprintf("storage error [%s]: %s (bucket=%s, key=%s)", e.Code, e.Message, e.Bucket, e.Key)
	} else if e.Bucket != "" {
		return fmt.Sprintf("storage error [%s]: %s (bucket=%s)", e.Code, e.Message, e.Bucket)
	}
	return fmt.Sprintf("storage error [%s]: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error.
func (e *StorageError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target error code.
func (e *StorageError) Is(target error) bool {
	if t, ok := target.(*StorageError); ok {
		return e.Code == t.Code
	}
	return false
}

// NewStorageError creates a new storage error.
func NewStorageError(code, message string) *StorageError {
	return &StorageError{
		Code:    code,
		Message: message,
	}
}

// NewStorageErrorWithBucket creates a new storage error with bucket context.
func NewStorageErrorWithBucket(code, message, bucket string) *StorageError {
	return &StorageError{
		Code:    code,
		Message: message,
		Bucket:  bucket,
	}
}

// NewStorageErrorWithObject creates a new storage error with object context.
func NewStorageErrorWithObject(code, message, bucket, key string) *StorageError {
	return &StorageError{
		Code:    code,
		Message: message,
		Bucket:  bucket,
		Key:     key,
	}
}

// WrapStorageError wraps an existing error with storage error context.
func WrapStorageError(code, message string, cause error) *StorageError {
	return &StorageError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// WrapStorageErrorWithObject wraps an error with object context.
func WrapStorageErrorWithObject(code, message, bucket, key string, cause error) *StorageError {
	return &StorageError{
		Code:    code,
		Message: message,
		Bucket:  bucket,
		Key:     key,
		Cause:   cause,
	}
}

// Predefined errors for common cases
var (
	ErrObjectNotFound      = NewStorageError(ErrCodeObjectNotFound, "object not found")
	ErrObjectAlreadyExists = NewStorageError(ErrCodeObjectAlreadyExists, "object already exists")
	ErrInvalidObjectKey    = NewStorageError(ErrCodeInvalidObjectKey, "invalid object key")
	ErrObjectTooLarge      = NewStorageError(ErrCodeObjectTooLarge, "object size exceeds limit")

	ErrBucketNotFound      = NewStorageError(ErrCodeBucketNotFound, "bucket not found")
	ErrBucketAlreadyExists = NewStorageError(ErrCodeBucketAlreadyExists, "bucket already exists")
	ErrBucketNotEmpty      = NewStorageError(ErrCodeBucketNotEmpty, "bucket is not empty")
	ErrInvalidBucketName   = NewStorageError(ErrCodeInvalidBucketName, "invalid bucket name")

	ErrPermissionDenied = NewStorageError(ErrCodePermissionDenied, "permission denied")
	ErrAccessDenied     = NewStorageError(ErrCodeAccessDenied, "access denied")
	ErrUnauthorized     = NewStorageError(ErrCodeUnauthorized, "unauthorized")

	ErrNetworkError     = NewStorageError(ErrCodeNetworkError, "network error")
	ErrConnectionFailed = NewStorageError(ErrCodeConnectionFailed, "connection failed")
	ErrTimeout          = NewStorageError(ErrCodeTimeout, "operation timeout")

	ErrInvalidRequest   = NewStorageError(ErrCodeInvalidRequest, "invalid request")
	ErrInvalidParameter = NewStorageError(ErrCodeInvalidParameter, "invalid parameter")
	ErrOperationFailed  = NewStorageError(ErrCodeOperationFailed, "operation failed")
	ErrNotImplemented   = NewStorageError(ErrCodeNotImplemented, "operation not implemented")
	ErrInternalError    = NewStorageError(ErrCodeInternalError, "internal error")

	ErrStreamReadError  = NewStorageError(ErrCodeStreamReadError, "stream read error")
	ErrStreamWriteError = NewStorageError(ErrCodeStreamWriteError, "stream write error")
	ErrStreamClosed     = NewStorageError(ErrCodeStreamClosed, "stream already closed")

	ErrInvalidUploadID   = NewStorageError(ErrCodeInvalidUploadID, "invalid upload ID")
	ErrInvalidPartNumber = NewStorageError(ErrCodeInvalidPartNumber, "invalid part number")
	ErrInvalidPartOrder  = NewStorageError(ErrCodeInvalidPartOrder, "invalid part order")
	ErrMultipartNotFound = NewStorageError(ErrCodeMultipartNotFound, "multipart upload not found")
)

// IsNotFoundError checks if the error is a "not found" error.
func IsNotFoundError(err error) bool {
	if se, ok := err.(*StorageError); ok {
		return se.Code == ErrCodeObjectNotFound || se.Code == ErrCodeBucketNotFound || se.Code == ErrCodeMultipartNotFound
	}
	return false
}

// IsAccessError checks if the error is an access-related error.
func IsAccessError(err error) bool {
	if se, ok := err.(*StorageError); ok {
		return se.Code == ErrCodePermissionDenied || se.Code == ErrCodeAccessDenied || se.Code == ErrCodeUnauthorized
	}
	return false
}

// IsNetworkError checks if the error is a network-related error.
func IsNetworkError(err error) bool {
	if se, ok := err.(*StorageError); ok {
		return se.Code == ErrCodeNetworkError || se.Code == ErrCodeConnectionFailed || se.Code == ErrCodeTimeout
	}
	return false
}

// IsRetryable checks if the error is retryable.
func IsRetryable(err error) bool {
	if se, ok := err.(*StorageError); ok {
		switch se.Code {
		case ErrCodeNetworkError, ErrCodeConnectionFailed, ErrCodeTimeout, ErrCodeInternalError:
			return true
		}
	}
	return false
}

// ConvertToStorageError converts a generic error to a StorageError.
// If the error is already a StorageError, it returns it as-is.
// Otherwise, it wraps it as an internal error.
func ConvertToStorageError(err error) *StorageError {
	if err == nil {
		return nil
	}

	if se, ok := err.(*StorageError); ok {
		return se
	}

	return WrapStorageError(ErrCodeInternalError, err.Error(), err)
}
