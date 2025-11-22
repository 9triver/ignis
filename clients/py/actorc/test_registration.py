#!/usr/bin/env python3
"""
简单的应用注册测试脚本
"""

import os
import sys

# 确保可以导入 actorc 模块
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

def test_imports():
    """测试导入是否正常"""
    print("测试模块导入...")
    
    try:
        from protos import ignis_pb2, ignis_pb2_grpc
        print("✓ ignis protobuf 模块导入成功")
    except ImportError as e:
        print(f"✗ ignis protobuf 模块导入失败: {e}")
        return False
    
    try:
        from client.registration_client import ApplicationRegistrationClient
        print("✓ 注册客户端模块导入成功")
    except ImportError as e:
        print(f"✗ 注册客户端模块导入失败: {e}")
        return False
    
    try:
        from controller.context import ActorContext
        print("✓ ActorContext 模块导入成功")
    except ImportError as e:
        print(f"✗ ActorContext 模块导入失败: {e}")
        return False
    
    return True


def test_message_creation():
    """测试消息创建"""
    print("\n测试消息创建...")
    
    try:
        from protos import ignis_pb2
        
        # 创建请求消息
        request = ignis_pb2.ApplicationRegistrationRequest(
            application_id="test-app-001"
        )
        print(f"✓ 创建请求消息成功: application_id={request.application_id}")
        
        # 创建响应消息
        response = ignis_pb2.ApplicationRegistrationResponse(
            success=True,
            error=""
        )
        print(f"✓ 创建响应消息成功: success={response.success}")
        
        return True
        
    except Exception as e:
        print(f"✗ 消息创建失败: {e}")
        return False


def test_client_creation():
    """测试客户端创建"""
    print("\n测试客户端创建...")
    
    try:
        from client.registration_client import ApplicationRegistrationClient
        
        # 创建客户端（不连接）
        client = ApplicationRegistrationClient("localhost:50051")
        print("✓ 客户端创建成功")
        
        return True
        
    except Exception as e:
        print(f"✗ 客户端创建失败: {e}")
        return False


def main():
    """主测试函数"""
    print("ApplicationRegistrationService 测试")
    print("=" * 40)
    
    tests = [
        ("模块导入测试", test_imports),
        ("消息创建测试", test_message_creation),
        ("客户端创建测试", test_client_creation)
    ]
    
    results = []
    for name, test_func in tests:
        try:
            result = test_func()
            results.append((name, result))
        except Exception as e:
            print(f"{name} 执行异常: {e}")
            results.append((name, False))
    
    # 总结
    print("\n" + "=" * 40)
    print("测试总结:")
    success_count = 0
    for name, success in results:
        status = "✓ 通过" if success else "✗ 失败"
        print(f"  {name}: {status}")
        if success:
            success_count += 1
    
    print(f"\n总体结果: {success_count}/{len(results)} 测试通过")
    
    if success_count == len(results):
        print("\n🎉 所有测试通过！可以尝试连接到实际的 Ignis 服务进行应用注册。")
        print("\n使用方法:")
        print("1. 确保 Ignis 服务正在运行")
        print("2. 设置环境变量: export IGNIS_ADDR=your_server:port")
        print("3. 运行示例: python examples/registration_example.py")
    else:
        print("\n❌ 部分测试失败，请检查代码和依赖。")


if __name__ == "__main__":
    main()