# Ignis Storage 存储包

[English](README.md) | **中文**

ignis 框架中的轻量级、可扩展的对象存储操作接口。

## 概述

`storage` 包为对象存储提供了一个简洁的抽象层，使其易于与 S3 兼容服务、云存储提供商或自定义存储后端集成。它旨在与 ignis 的对象系统和流式处理能力无缝协作。

## 特性

- 🎯 **简单直观的 API** - 为常见存储操作提供简洁的接口
- 🔄 **流式处理支持** - 与 ignis Stream 集成，高效处理大型对象
- 🔌 **可扩展** - 易于为不同的存储后端实现
- 📦 **分块上传** - 支持上传超大对象
- 🔗 **预签名 URL** - 生成临时 URL 供客户端直接访问
- ⚡ **上下文感知** - 支持取消和超时
- 🛡️ **类型安全的错误** - 具有特定错误码的全面错误处理

## 接口

### ObjectStorage（核心）

基本对象存储操作：

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

### StreamStorage（流式存储）

扩展的流式处理能力：

```go
type StreamStorage interface {
    ObjectStorage
    
    // 流式操作
    PutStream(ctx context.Context, req *PutStreamRequest) (*PutObjectResponse, error)
    GetStream(ctx context.Context, req *GetObjectRequest) (io.ReadCloser, *ObjectMetadata, error)
    
    // Ignis Stream 集成
    PutIgnisStream(ctx context.Context, req *PutIgnisStreamRequest) (*PutObjectResponse, error)
    GetIgnisStream(ctx context.Context, req *GetObjectRequest) (objects.Interface, error)
}
```

### BucketManager（存储桶管理）

存储桶管理操作：

```go
type BucketManager interface {
    CreateBucket(ctx context.Context, bucket string, opts *BucketOptions) error
    DeleteBucket(ctx context.Context, bucket string) error
    ListBuckets(ctx context.Context) ([]BucketInfo, error)
    BucketExists(ctx context.Context, bucket string) (bool, error)
    GetBucketInfo(ctx context.Context, bucket string) (*BucketInfo, error)
}
```

### MultipartUpload（分块上传）

大文件分块上传：

```go
type MultipartUpload interface {
    InitiateMultipartUpload(ctx context.Context, req *InitiateMultipartRequest) (*MultipartUploadInfo, error)
    UploadPart(ctx context.Context, req *UploadPartRequest) (*UploadPartResponse, error)
    CompleteMultipartUpload(ctx context.Context, req *CompleteMultipartRequest) (*PutObjectResponse, error)
    AbortMultipartUpload(ctx context.Context, req *AbortMultipartRequest) error
    ListParts(ctx context.Context, req *ListPartsRequest) (*ListPartsResponse, error)
}
```

## 使用示例

### 基本操作

```go
import (
    "context"
    "github.com/9triver/ignis/storage"
    "github.com/your-org/iarnet/internal/storage/s3"
)

// 初始化存储（具体实现相关）
storage, err := s3.NewS3Storage(&s3.S3Config{
    Endpoint:  "s3.amazonaws.com",
    Region:    "us-east-1",
    AccessKey: "YOUR_ACCESS_KEY",
    SecretKey: "YOUR_SECRET_KEY",
})

// 上传对象
resp, err := storage.PutObject(context.Background(), &storage.PutObjectRequest{
    Bucket:      "my-bucket",
    Key:         "data/file.txt",
    Data:        []byte("Hello, World!"),
    ContentType: "text/plain",
    Metadata: map[string]string{
        "author": "ignis",
    },
})

// 下载对象
obj, err := storage.GetObject(context.Background(), &storage.GetObjectRequest{
    Bucket: "my-bucket",
    Key:    "data/file.txt",
})
fmt.Println(string(obj.Data))

// 列出对象
list, err := storage.ListObjects(context.Background(), &storage.ListObjectsRequest{
    Bucket: "my-bucket",
    Prefix: "data/",
})
for _, obj := range list.Objects {
    fmt.Printf("%s (%d bytes)\n", obj.Key, obj.Size)
}

// 删除对象
err = storage.DeleteObject(context.Background(), &storage.DeleteObjectRequest{
    Bucket: "my-bucket",
    Key:    "data/file.txt",
})
```

### 流式操作

```go
// 从 io.Reader 上传
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

// 以流的方式下载
reader, metadata, err := streamStorage.GetStream(context.Background(), &storage.GetObjectRequest{
    Bucket: "my-bucket",
    Key:    "uploads/large-file.bin",
})
defer reader.Close()

// 处理流...
io.Copy(outputFile, reader)
```

### Ignis Stream 集成

```go
import "github.com/9triver/ignis/objects"

// 从数据源创建 ignis Stream
dataChannel := make(chan objects.Interface, 10)
go func() {
    defer close(dataChannel)
    for i := 0; i < 100; i++ {
        data := fmt.Sprintf("chunk-%d", i)
        dataChannel <- objects.NewLocal([]byte(data), objects.LangGo)
    }
}()

stream := objects.NewStream(dataChannel, objects.LangGo)

// 上传 ignis Stream
streamStorage := storage.(storage.StreamStorage)
_, err := streamStorage.PutIgnisStream(context.Background(), &storage.PutIgnisStreamRequest{
    Bucket: "my-bucket",
    Key:    "streams/data.bin",
    Stream: stream,
})

// 作为 ignis Stream 下载
stream, err := streamStorage.GetIgnisStream(context.Background(), &storage.GetObjectRequest{
    Bucket: "my-bucket",
    Key:    "streams/data.bin",
})

// 在 actor 函数中使用
result := myActorFunction(stream)
```

### 错误处理

```go
obj, err := storage.GetObject(ctx, req)
if err != nil {
    if storage.IsNotFoundError(err) {
        log.Println("对象不存在")
    } else if storage.IsAccessError(err) {
        log.Println("权限被拒绝")
    } else if storage.IsNetworkError(err) {
        log.Println("网络错误，重试中...")
        // 重试逻辑
    } else if storage.IsRetryable(err) {
        log.Println("临时错误，可以重试")
    } else {
        log.Printf("致命错误: %v", err)
    }
}

// 访问错误详情
if se, ok := err.(*storage.StorageError); ok {
    log.Printf("错误码: %s, 存储桶: %s, 键: %s", se.Code, se.Bucket, se.Key)
}
```

### 工具函数

```go
import "github.com/9triver/ignis/storage"

// 验证存储桶名称
err := storage.ValidateBucketName("my-bucket")

// 验证对象键
err = storage.ValidateObjectKey("path/to/file.txt")

// 规范化键
key := storage.NormalizeKey("//path///to//file.txt") // "path/to/file.txt"

// 连接路径段
key = storage.JoinKey("folder", "subfolder", "file.txt") // "folder/subfolder/file.txt"

// 推测内容类型
contentType := storage.GuessContentType("image.jpg") // "image/jpeg"

// 计算分块上传的最佳分块大小
partSize := storage.CalculatePartSize(5 * 1024 * 1024 * 1024) // 5GB 文件

// 检查是否应该使用分块上传
useMultipart := storage.ShouldUseMultipart(150 * 1024 * 1024) // 150MB 返回 true
```

## 实现指南

为特定后端实现存储接口时：

1. **从 ObjectStorage 开始** - 首先实现核心接口
2. **添加 StreamStorage** - 以获得大对象的更好性能
3. **使用预定义错误** - 返回适当的 StorageError 类型
4. **验证输入** - 使用像 `ValidateBucketName` 这样的工具函数
5. **支持上下文** - 正确处理取消操作
6. **添加日志** - 使用 ignis 日志系统
7. **线程安全** - 确保并发访问是安全的

### 实现结构示例

```go
package s3

import "github.com/9triver/ignis/storage"

type S3Storage struct {
    client *s3.Client
    // ... 配置字段
}

func NewS3Storage(config *S3Config) (*S3Storage, error) {
    // 初始化 S3 客户端
}

// 实现 storage.ObjectStorage
func (s *S3Storage) PutObject(ctx context.Context, req *storage.PutObjectRequest) (*storage.PutObjectResponse, error) {
    // 验证输入
    if err := storage.ValidateBucketName(req.Bucket); err != nil {
        return nil, err
    }
    if err := storage.ValidateObjectKey(req.Key); err != nil {
        return nil, err
    }
    
    // 调用 S3 API
    // 使用 storage.WrapStorageError 处理错误
    // 返回响应
}

// 实现其他接口...
```

## 错误码参考

| 错误码 | 描述 | 可重试 |
|------|------|--------|
| `ObjectNotFound` | 对象不存在 | 否 |
| `ObjectAlreadyExists` | 对象已存在 | 否 |
| `BucketNotFound` | 存储桶不存在 | 否 |
| `PermissionDenied` | 访问被拒绝 | 否 |
| `NetworkError` | 网络故障 | 是 |
| `Timeout` | 操作超时 | 是 |
| `InternalError` | 内部错误 | 是 |

查看 `errors.go` 获取完整的错误码列表。

## 最佳实践

1. **始终使用 context** - 传递适当的上下文以支持超时/取消
2. **尽早验证输入** - 在进行 API 调用之前使用工具函数
3. **正确处理错误** - 检查错误类型并提供有意义的消息
4. **对大文件使用流式处理** - 不要将大文件完全加载到内存中
5. **设置适当的内容类型** - 使用 `GuessContentType` 或显式设置
6. **添加元数据** - 包含有用的元数据以便调试和跟踪
7. **清理资源** - 使用完毕后关闭读取器/流

## 线程安全

所有接口都设计为线程安全的。实现应该支持并发操作。

## 性能考虑

- 对于 > 100MB 的对象使用 `StreamStorage`
- 对于 > 100MB 的对象使用分块上传（大多数实现中自动进行）
- 根据网络和对象大小设置适当的分块大小
- 考虑使用预签名 URL 供客户端直接上传/下载
- 在实现中使用连接池

## 许可证

ignis 框架的一部分。

