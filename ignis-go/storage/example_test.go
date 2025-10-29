package storage_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/9triver/ignis/objects"
	"github.com/9triver/ignis/storage"
)

// MockStorage is a simple in-memory implementation for testing
type MockStorage struct {
	data map[string]map[string][]byte // bucket -> key -> data
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		data: make(map[string]map[string][]byte),
	}
}

func (m *MockStorage) PutObject(ctx context.Context, req *storage.PutObjectRequest) (*storage.PutObjectResponse, error) {
	if m.data[req.Bucket] == nil {
		m.data[req.Bucket] = make(map[string][]byte)
	}
	m.data[req.Bucket][req.Key] = req.Data

	return &storage.PutObjectResponse{
		Bucket: req.Bucket,
		Key:    req.Key,
		Size:   int64(len(req.Data)),
	}, nil
}

func (m *MockStorage) GetObject(ctx context.Context, req *storage.GetObjectRequest) (*storage.GetObjectResponse, error) {
	bucket, ok := m.data[req.Bucket]
	if !ok {
		return nil, storage.NewStorageErrorWithBucket(storage.ErrCodeBucketNotFound, "bucket not found", req.Bucket)
	}

	data, ok := bucket[req.Key]
	if !ok {
		return nil, storage.NewStorageErrorWithObject(storage.ErrCodeObjectNotFound, "object not found", req.Bucket, req.Key)
	}

	return &storage.GetObjectResponse{
		Bucket: req.Bucket,
		Key:    req.Key,
		Data:   data,
		Size:   int64(len(data)),
	}, nil
}

func (m *MockStorage) DeleteObject(ctx context.Context, req *storage.DeleteObjectRequest) error {
	if bucket, ok := m.data[req.Bucket]; ok {
		delete(bucket, req.Key)
	}
	return nil
}

func (m *MockStorage) ListObjects(ctx context.Context, req *storage.ListObjectsRequest) (*storage.ListObjectsResponse, error) {
	bucket, ok := m.data[req.Bucket]
	if !ok {
		return nil, storage.NewStorageErrorWithBucket(storage.ErrCodeBucketNotFound, "bucket not found", req.Bucket)
	}

	var objects []storage.ObjectInfo
	for key, data := range bucket {
		if strings.HasPrefix(key, req.Prefix) {
			objects = append(objects, storage.ObjectInfo{
				Key:  key,
				Size: int64(len(data)),
			})
		}
	}

	return &storage.ListObjectsResponse{
		Bucket:   req.Bucket,
		Objects:  objects,
		KeyCount: len(objects),
	}, nil
}

func (m *MockStorage) HeadObject(ctx context.Context, req *storage.HeadObjectRequest) (*storage.ObjectMetadata, error) {
	bucket, ok := m.data[req.Bucket]
	if !ok {
		return nil, storage.NewStorageErrorWithBucket(storage.ErrCodeBucketNotFound, "bucket not found", req.Bucket)
	}

	data, ok := bucket[req.Key]
	if !ok {
		return nil, storage.NewStorageErrorWithObject(storage.ErrCodeObjectNotFound, "object not found", req.Bucket, req.Key)
	}

	return &storage.ObjectMetadata{
		Bucket: req.Bucket,
		Key:    req.Key,
		Size:   int64(len(data)),
	}, nil
}

func (m *MockStorage) CopyObject(ctx context.Context, req *storage.CopyObjectRequest) error {
	srcBucket, ok := m.data[req.SourceBucket]
	if !ok {
		return storage.NewStorageErrorWithBucket(storage.ErrCodeBucketNotFound, "source bucket not found", req.SourceBucket)
	}

	data, ok := srcBucket[req.SourceKey]
	if !ok {
		return storage.NewStorageErrorWithObject(storage.ErrCodeObjectNotFound, "source object not found", req.SourceBucket, req.SourceKey)
	}

	if m.data[req.DestinationBucket] == nil {
		m.data[req.DestinationBucket] = make(map[string][]byte)
	}
	m.data[req.DestinationBucket][req.DestinationKey] = data

	return nil
}

func (m *MockStorage) ObjectExists(ctx context.Context, bucket, key string) (bool, error) {
	if b, ok := m.data[bucket]; ok {
		_, exists := b[key]
		return exists, nil
	}
	return false, nil
}

// Example demonstrates basic storage operations
func Example() {
	ctx := context.Background()
	store := NewMockStorage()

	// Upload an object
	_, err := store.PutObject(ctx, &storage.PutObjectRequest{
		Bucket: "my-bucket",
		Key:    "data/file.txt",
		Data:   []byte("Hello, Storage!"),
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Download an object
	obj, err := store.GetObject(ctx, &storage.GetObjectRequest{
		Bucket: "my-bucket",
		Key:    "data/file.txt",
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println(string(obj.Data))
	// Output: Hello, Storage!
}

// Example_errorHandling demonstrates error handling
func Example_errorHandling() {
	ctx := context.Background()
	store := NewMockStorage()

	// Try to get non-existent object
	_, err := store.GetObject(ctx, &storage.GetObjectRequest{
		Bucket: "my-bucket",
		Key:    "missing.txt",
	})

	if err != nil {
		if storage.IsNotFoundError(err) {
			fmt.Println("Object not found")
		}

		if se, ok := err.(*storage.StorageError); ok {
			fmt.Printf("Error code: %s\n", se.Code)
		}
	}
	// Output:
	// Object not found
	// Error code: BucketNotFound
}

// Example_listObjects demonstrates listing objects
func Example_listObjects() {
	ctx := context.Background()
	store := NewMockStorage()

	// Upload some test objects
	store.PutObject(ctx, &storage.PutObjectRequest{
		Bucket: "my-bucket",
		Key:    "data/file1.txt",
		Data:   []byte("content1"),
	})
	store.PutObject(ctx, &storage.PutObjectRequest{
		Bucket: "my-bucket",
		Key:    "data/file2.txt",
		Data:   []byte("content2"),
	})
	store.PutObject(ctx, &storage.PutObjectRequest{
		Bucket: "my-bucket",
		Key:    "other/file3.txt",
		Data:   []byte("content3"),
	})

	// List objects with prefix
	resp, err := store.ListObjects(ctx, &storage.ListObjectsRequest{
		Bucket: "my-bucket",
		Prefix: "data/",
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Found %d objects\n", resp.KeyCount)
	for _, obj := range resp.Objects {
		fmt.Printf("- %s (%d bytes)\n", obj.Key, obj.Size)
	}
	// Output:
	// Found 2 objects
	// - data/file1.txt (8 bytes)
	// - data/file2.txt (8 bytes)
}

// Example_utilityFunctions demonstrates utility functions
func Example_utilityFunctions() {
	// Validate bucket name
	err := storage.ValidateBucketName("my-bucket")
	fmt.Printf("Valid bucket: %v\n", err == nil)

	// Validate object key
	err = storage.ValidateObjectKey("path/to/file.txt")
	fmt.Printf("Valid key: %v\n", err == nil)

	// Normalize key
	key := storage.NormalizeKey("//path///to//file.txt")
	fmt.Printf("Normalized: %s\n", key)

	// Join key segments
	key = storage.JoinKey("folder", "subfolder", "file.txt")
	fmt.Printf("Joined: %s\n", key)

	// Guess content type
	contentType := storage.GuessContentType("image.jpg")
	fmt.Printf("Content type: %s\n", contentType)

	// Output:
	// Valid bucket: true
	// Valid key: true
	// Normalized: path/to/file.txt
	// Joined: folder/subfolder/file.txt
	// Content type: image/jpeg
}

// Example_streamOperations demonstrates stream conversion
func Example_streamOperations() {
	ctx := context.Background()

	// Convert io.Reader to ignis Stream
	reader := strings.NewReader("Hello from stream")
	stream := storage.ReaderToIgnisStream(ctx, reader, 1024, objects.LangGo)

	fmt.Printf("Stream created: %v\n", stream != nil)
	// Output: Stream created: true
}

// Example_copyObject demonstrates copying objects
func Example_copyObject() {
	ctx := context.Background()
	store := NewMockStorage()

	// Create source object
	store.PutObject(ctx, &storage.PutObjectRequest{
		Bucket: "source-bucket",
		Key:    "original.txt",
		Data:   []byte("Original content"),
	})

	// Copy object
	err := store.CopyObject(ctx, &storage.CopyObjectRequest{
		SourceBucket:      "source-bucket",
		SourceKey:         "original.txt",
		DestinationBucket: "dest-bucket",
		DestinationKey:    "copy.txt",
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Verify copy
	obj, _ := store.GetObject(ctx, &storage.GetObjectRequest{
		Bucket: "dest-bucket",
		Key:    "copy.txt",
	})

	fmt.Println(string(obj.Data))
	// Output: Original content
}

// Example_validation demonstrates input validation
func Example_validation() {
	// Invalid bucket names
	err := storage.ValidateBucketName("My-Bucket") // uppercase not allowed
	fmt.Printf("Uppercase error: %v\n", err != nil)

	err = storage.ValidateBucketName("ab") // too short
	fmt.Printf("Too short error: %v\n", err != nil)

	// Invalid object keys
	err = storage.ValidateObjectKey("") // empty
	fmt.Printf("Empty key error: %v\n", err != nil)

	err = storage.ValidateObjectKey("/leading-slash") // starts with /
	fmt.Printf("Leading slash error: %v\n", err != nil)

	// Output:
	// Uppercase error: true
	// Too short error: true
	// Empty key error: true
	// Leading slash error: true
}

// Example_calculatePartSize demonstrates multipart calculations
func Example_calculatePartSize() {
	// Small file
	size := storage.CalculatePartSize(10 * 1024 * 1024) // 10MB
	fmt.Printf("10MB file: %dMB per part\n", size/(1024*1024))

	// Large file
	size = storage.CalculatePartSize(100 * 1024 * 1024 * 1024) // 100GB
	fmt.Printf("100GB file: %dMB per part\n", size/(1024*1024))

	// Should use multipart?
	useMultipart := storage.ShouldUseMultipart(150 * 1024 * 1024) // 150MB
	fmt.Printf("150MB uses multipart: %v\n", useMultipart)

	// Output:
	// 10MB file: 5MB per part
	// 100GB file: 10MB per part
	// 150MB uses multipart: true
}

// Uncomment below if needed for custom stream reading:
//
// type mockStreamReader struct {
// 	data   []byte
// 	offset int
// }
//
// func (r *mockStreamReader) Read(p []byte) (n int, err error) {
// 	if r.offset >= len(r.data) {
// 		return 0, io.EOF
// 	}
// 	n = copy(p, r.data[r.offset:])
// 	r.offset += n
// 	return n, nil
// }
//
// func (r *mockStreamReader) Close() error {
// 	return nil
// }
