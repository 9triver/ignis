"""
Examples of using the Storage Client.

这些示例展示了如何在实际场景中使用 StorageClient。
"""

from actorc.storage import StorageClient, StorageError, NotFoundError


def example_basic_upload_download():
    """基本的上传和下载示例"""
    print("=== 基本上传和下载 ===")
    
    client = StorageClient(endpoint="localhost:9000")
    
    # 上传文件
    print("上传文件...")
    client.upload_file(
        bucket="my-bucket",
        key="documents/report.pdf",
        file_path="/path/to/report.pdf",
        metadata={"author": "John Doe", "version": "1.0"}
    )
    print("✓ 文件上传成功")
    
    # 下载文件
    print("下载文件...")
    client.download_file(
        bucket="my-bucket",
        key="documents/report.pdf",
        file_path="/path/to/downloaded_report.pdf",
        overwrite=True
    )
    print("✓ 文件下载成功")


def example_upload_bytes():
    """上传字节数据示例"""
    print("\n=== 上传字节数据 ===")
    
    client = StorageClient()
    
    # 创建一些数据
    data = b"Hello, ignis storage!\n" * 100
    
    # 上传
    print(f"上传 {len(data)} 字节...")
    result = client.put_object(
        bucket="my-bucket",
        key="data/hello.txt",
        data=data,
        content_type="text/plain"
    )
    print(f"✓ 上传成功: {result}")
    
    # 下载
    print("下载并验证...")
    downloaded = client.get_object(bucket="my-bucket", key="data/hello.txt")
    assert downloaded == data
    print("✓ 数据验证成功")


def example_list_objects():
    """列出对象示例"""
    print("\n=== 列出对象 ===")
    
    client = StorageClient()
    
    # 上传一些测试文件
    for i in range(5):
        client.put_object(
            bucket="my-bucket",
            key=f"test/file{i}.txt",
            data=f"Content {i}".encode()
        )
    
    # 列出所有对象
    result = client.list_objects(bucket="my-bucket", prefix="test/")
    print(f"找到 {len(result['objects'])} 个对象:")
    for obj in result["objects"]:
        print(f"  - {obj['key']} ({obj['size']} bytes)")


def example_error_handling():
    """错误处理示例"""
    print("\n=== 错误处理 ===")
    
    client = StorageClient()
    
    # 尝试下载不存在的对象
    try:
        client.get_object(bucket="my-bucket", key="nonexistent.txt")
    except NotFoundError as e:
        print(f"✓ 捕获到预期错误: {e}")
    
    # 尝试使用无效的存储桶名称
    try:
        client.upload_file(
            bucket="Invalid-Bucket-Name",  # 大写字母无效
            key="test.txt",
            file_path="/tmp/test.txt"
        )
    except StorageError as e:
        print(f"✓ 捕获到参数错误: {e}")


def example_streaming_upload():
    """流式上传大文件示例"""
    print("\n=== 流式上传 ===")
    
    client = StorageClient()
    
    # 模拟大文件上传
    import io
    
    # 创建 10MB 的数据流
    large_data = io.BytesIO(b"x" * (10 * 1024 * 1024))
    
    print("上传 10MB 数据流...")
    result = client.upload_from_stream(
        bucket="my-bucket",
        key="large/data.bin",
        stream=large_data,
        content_type="application/octet-stream",
        chunk_size=1024 * 1024  # 1MB chunks
    )
    print(f"✓ 流式上传成功: {result}")


def example_check_exists():
    """检查对象是否存在示例"""
    print("\n=== 检查对象存在 ===")
    
    client = StorageClient()
    
    # 上传对象
    client.put_object(
        bucket="my-bucket",
        key="test/exists.txt",
        data=b"I exist!"
    )
    
    # 检查存在
    exists = client.object_exists(bucket="my-bucket", key="test/exists.txt")
    print(f"✓ test/exists.txt 存在: {exists}")
    
    # 检查不存在
    exists = client.object_exists(bucket="my-bucket", key="test/notexists.txt")
    print(f"✓ test/notexists.txt 存在: {exists}")


def example_workflow_integration():
    """在 workflow 中使用的示例"""
    print("\n=== Workflow 集成示例 ===")
    
    from lucas import workflow, function, Workflow
    from actorc.controller.context import ActorFunction, ActorExecutor
    
    @function(
        wrapper=ActorFunction,
        dependency=["actorc"],
        provider="actor",
        name="process_data_from_storage",
        venv="myenv",
    )
    def process_data_from_storage(params):
        """从 storage 读取数据，处理后保存回去"""
        client = StorageClient()
        
        # 读取输入
        print(f"读取 {params['input_key']}...")
        input_data = client.get_object(
            bucket=params["bucket"],
            key=params["input_key"]
        )
        
        # 处理数据（这里只是简单转换为大写）
        processed = input_data.decode().upper().encode()
        
        # 保存结果
        print(f"保存到 {params['output_key']}...")
        client.put_object(
            bucket=params["bucket"],
            key=params["output_key"],
            data=processed,
            metadata={"processed": "true"}
        )
        
        return {
            "status": "success",
            "input_size": len(input_data),
            "output_size": len(processed)
        }
    
    @workflow(executor=ActorExecutor)
    def data_processing_workflow(wf: Workflow):
        _in = wf.input()
        
        result = wf.call("process_data_from_storage", {
            "bucket": _in["bucket"],
            "input_key": _in["input_key"],
            "output_key": _in["output_key"]
        })
        
        return result
    
    print("✓ Workflow 函数已定义")
    print("调用方式:")
    print("  workflow_func({'bucket': 'my-bucket', 'input_key': 'in.txt', 'output_key': 'out.txt'})")


def example_context_manager():
    """使用上下文管理器示例"""
    print("\n=== 上下文管理器 ===")
    
    with StorageClient(endpoint="localhost:9000") as client:
        # 在 with 块中使用客户端
        client.put_object(
            bucket="my-bucket",
            key="temp/data.txt",
            data=b"Temporary data"
        )
        print("✓ 使用上下文管理器上传成功")
    
    # 客户端自动关闭
    print("✓ 客户端已自动关闭")


def example_batch_operations():
    """批量操作示例"""
    print("\n=== 批量操作 ===")
    
    client = StorageClient()
    
    # 批量上传
    files_to_upload = [
        ("file1.txt", b"Content 1"),
        ("file2.txt", b"Content 2"),
        ("file3.txt", b"Content 3"),
    ]
    
    print("批量上传文件...")
    for key, data in files_to_upload:
        client.put_object(
            bucket="my-bucket",
            key=f"batch/{key}",
            data=data
        )
        print(f"  ✓ {key} 已上传")
    
    # 批量列出
    result = client.list_objects(bucket="my-bucket", prefix="batch/")
    print(f"\n找到 {len(result['objects'])} 个文件")
    
    # 批量删除
    print("\n批量删除文件...")
    for obj in result["objects"]:
        client.delete_object(bucket="my-bucket", key=obj["key"])
        print(f"  ✓ {obj['key']} 已删除")


if __name__ == "__main__":
    print("Storage Client 使用示例")
    print("=" * 50)
    
    try:
        # 运行所有示例
        example_basic_upload_download()
        example_upload_bytes()
        example_list_objects()
        example_error_handling()
        example_streaming_upload()
        example_check_exists()
        example_context_manager()
        example_batch_operations()
        example_workflow_integration()
        
        print("\n" + "=" * 50)
        print("✓ 所有示例完成")
        
    except Exception as e:
        print(f"\n❌ 错误: {e}")
        import traceback
        traceback.print_exc()

