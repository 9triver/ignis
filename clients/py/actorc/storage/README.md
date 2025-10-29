# Ignis Storage Client

Python 客户端，用于上传和下载文件到 ignis storage 服务。

## 安装

```bash
# 作为 actorc 包的一部分安装
pip install -e /path/to/ignis/clients/py/actorc
```

## 快速开始

### 基本用法

```python
from actorc.storage import StorageClient

# 初始化客户端
client = StorageClient(endpoint="storage.ignis.local:9000")

# 上传文件
client.upload_file(
    bucket="my-bucket",
    key="data/file.txt",
    file_path="/path/to/local/file.txt"
)

# 下载文件
client.download_file(
    bucket="my-bucket",
    key="data/file.txt",
    file_path="/path/to/save/file.txt"
)

# 上传字节数据
client.put_object(
    bucket="my-bucket",
    key="data/content.txt",
    data=b"Hello, Storage!",
    metadata={"author": "ignis"}
)

# 下载为字节
data = client.get_object(bucket="my-bucket", key="data/content.txt")
print(data.decode())

# 列出对象
result = client.list_objects(bucket="my-bucket", prefix="data/")
for obj in result["objects"]:
    print(f"{obj['key']} - {obj['size']} bytes")

# 检查对象是否存在
exists = client.object_exists(bucket="my-bucket", key="data/file.txt")
print(f"Object exists: {exists}")

# 删除对象
client.delete_object(bucket="my-bucket", key="data/file.txt")
```

### 上下文管理器

```python
from actorc.storage import StorageClient

with StorageClient(endpoint="storage.ignis.local:9000") as client:
    client.upload_file(
        bucket="my-bucket",
        key="uploads/data.bin",
        file_path="/path/to/data.bin"
    )
```

### 流式上传大文件

```python
from actorc.storage import StorageClient

client = StorageClient(endpoint="storage.ignis.local:9000")

# 流式上传（自动用于 > 100MB 的文件）
client.upload_file(
    bucket="my-bucket",
    key="large-files/video.mp4",
    file_path="/path/to/large/video.mp4",
    chunk_size=10 * 1024 * 1024  # 10MB chunks
)

# 从流上传
with open("large_data.bin", "rb") as f:
    client.upload_from_stream(
        bucket="my-bucket",
        key="data/large_data.bin",
        stream=f,
        content_type="application/octet-stream"
    )
```

### 在工作流中使用

```python
from lucas import workflow, function, Workflow
from lucas.serverless_function import Metadata
from actorc.controller.context import ActorContext, ActorFunction, ActorExecutor, ActorRuntime
from actorc.storage import StorageClient

context = ActorContext.createContext()

@function(
    wrapper=ActorFunction,
    dependency=["actorc"],
    provider="actor",
    name="process_file",
    venv="myenv",
)
def process_file(params):
    """从 storage 读取文件，处理后上传结果"""
    # 初始化存储客户端
    storage = StorageClient()
    
    # 下载输入文件
    input_data = storage.get_object(
        bucket=params["input_bucket"],
        key=params["input_key"]
    )
    
    # 处理数据
    result = process_data(input_data)
    
    # 上传结果
    storage.put_object(
        bucket=params["output_bucket"],
        key=params["output_key"],
        data=result
    )
    
    return {"status": "success", "output_key": params["output_key"]}

@workflow(executor=ActorExecutor)
def data_pipeline(wf: Workflow):
    _in = wf.input()
    result = wf.call("process_file", {
        "input_bucket": "input-data",
        "input_key": "raw/data.csv",
        "output_bucket": "processed-data",
        "output_key": "processed/data.parquet"
    })
    return result
```

## API 参考

### StorageClient

主要的存储客户端类。

#### 初始化

```python
client = StorageClient(
    endpoint="storage.ignis.local:9000",  # 存储服务端点
    access_key="your-access-key",         # 访问密钥（可选）
    secret_key="your-secret-key",         # 秘密密钥（可选）
    use_ssl=False                          # 是否使用 SSL
)
```

环境变量：
- `STORAGE_ENDPOINT`: 默认端点
- `STORAGE_ACCESS_KEY`: 默认访问密钥
- `STORAGE_SECRET_KEY`: 默认秘密密钥

#### 方法

##### put_object(bucket, key, data, content_type=None, metadata=None)

上传对象到存储。

**参数：**
- `bucket` (str): 存储桶名称
- `key` (str): 对象键（路径）
- `data` (bytes): 对象数据
- `content_type` (str, 可选): MIME 类型
- `metadata` (dict, 可选): 自定义元数据

**返回：** dict - 上传详情

##### get_object(bucket, key)

从存储下载对象。

**参数：**
- `bucket` (str): 存储桶名称
- `key` (str): 对象键

**返回：** bytes - 对象数据

**异常：**
- `NotFoundError`: 对象不存在
- `StorageError`: 下载失败

##### upload_file(bucket, key, file_path, content_type=None, metadata=None, chunk_size=None)

从本地文件系统上传文件。

**参数：**
- `bucket` (str): 存储桶名称
- `key` (str): 对象键（目标路径）
- `file_path` (str): 本地文件路径
- `content_type` (str, 可选): MIME 类型（自动检测）
- `metadata` (dict, 可选): 自定义元数据
- `chunk_size` (int, 可选): 分块大小（默认 5MB）

**返回：** dict - 上传详情

##### download_file(bucket, key, file_path, overwrite=False)

下载对象到本地文件系统。

**参数：**
- `bucket` (str): 存储桶名称
- `key` (str): 对象键
- `file_path` (str): 本地保存路径
- `overwrite` (bool): 是否覆盖已存在的文件

**返回：** dict - 下载详情

##### list_objects(bucket, prefix="", max_keys=1000)

列出存储桶中的对象。

**参数：**
- `bucket` (str): 存储桶名称
- `prefix` (str): 前缀过滤
- `max_keys` (int): 最大返回数量

**返回：** dict - 对象列表

##### delete_object(bucket, key)

删除对象。

**参数：**
- `bucket` (str): 存储桶名称
- `key` (str): 对象键

##### object_exists(bucket, key)

检查对象是否存在。

**参数：**
- `bucket` (str): 存储桶名称
- `key` (str): 对象键

**返回：** bool - 是否存在

##### head_object(bucket, key)

获取对象元数据（不下载内容）。

**参数：**
- `bucket` (str): 存储桶名称
- `key` (str): 对象键

**返回：** dict - 对象元数据

##### upload_from_stream(bucket, key, stream, content_type=None, metadata=None, chunk_size=None)

从流上传数据。

**参数：**
- `bucket` (str): 存储桶名称
- `key` (str): 对象键
- `stream` (BinaryIO): 文件类对象
- `content_type` (str, 可选): MIME 类型
- `metadata` (dict, 可选): 自定义元数据
- `chunk_size` (int, 可选): 分块大小

**返回：** dict - 上传详情

## 错误处理

```python
from actorc.storage import StorageClient, StorageError, NotFoundError

client = StorageClient()

try:
    data = client.get_object(bucket="my-bucket", key="missing.txt")
except NotFoundError as e:
    print(f"对象不存在: {e.bucket}/{e.key}")
except StorageError as e:
    print(f"存储错误 [{e.code}]: {e.message}")
```

### 错误类型

- `StorageError`: 所有存储错误的基类
- `NotFoundError`: 对象或存储桶不存在
- `PermissionError`: 访问被拒绝
- `NetworkError`: 网络相关错误
- `InvalidParameterError`: 无效参数

## 注意事项

1. **存储桶命名规则：**
   - 3-63 个字符
   - 只能包含小写字母、数字和连字符
   - 必须以字母或数字开头和结尾

2. **对象键规则：**
   - 不能为空
   - 最大长度 1024 字符
   - 不应以 `/` 开头

3. **大文件处理：**
   - 大于 100MB 的文件自动使用流式上传
   - 可以通过 `chunk_size` 参数调整分块大小

4. **内容类型：**
   - 如果未指定，会根据文件扩展名自动检测
   - 支持常见的 MIME 类型

## 待实现功能

当前实现是一个框架，以下功能待 gRPC proto 定义后实现：

- [ ] 实际的 gRPC 通信
- [ ] 真实的流式上传/下载
- [ ] 分块上传（multipart upload）
- [ ] 预签名 URL 生成
- [ ] 存储桶管理操作

## 许可证

ignis 框架的一部分。

