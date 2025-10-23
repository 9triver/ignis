"""
ApplicationRegistrationService 客户端
用于在 actorc 中调用应用注册服务
"""

import grpc
import os
from typing import Optional

from ..protos import ignis_pb2, ignis_pb2_grpc


class ApplicationRegistrationClient:
    """应用注册服务客户端"""
    
    def __init__(self, server_address: Optional[str] = None):
        """
        初始化客户端
        
        Args:
            server_address: 服务器地址，默认从环境变量 IGNIS_ADDR 获取
        """
        if server_address is None:
            server_address = os.getenv("IGNIS_ADDR", "localhost:50051")
        
        self.server_address = server_address
        self.channel = None
        self.stub = None
    
    def connect(self):
        """建立连接"""
        self.channel = grpc.insecure_channel(self.server_address)
        self.stub = ignis_pb2_grpc.ApplicationRegistrationServiceStub(self.channel)
        print(f"Connected to ApplicationRegistrationService at {self.server_address}")
    
    def disconnect(self):
        """关闭连接"""
        if self.channel:
            self.channel.close()
            self.channel = None
            self.stub = None
            print("Disconnected from ApplicationRegistrationService")
    
    def register_application(self, application_id: str) -> ignis_pb2.ApplicationRegistrationResponse:
        """
        注册应用
        
        Args:
            application_id: 应用ID
            
        Returns:
            ApplicationRegistrationResponse: 注册响应
        """
        if not self.stub:
            raise RuntimeError("Client not connected. Call connect() first.")
        
        request = ignis_pb2.ApplicationRegistrationRequest(
            application_id=application_id
        )
        
        try:
            response = self.stub.RegisterApplication(request)
            print(f"Application registration response: success={response.success}")
            if response.error:
                print(f"Registration error: {response.error}")
            return response
        except grpc.RpcError as e:
            print(f"gRPC error during registration: {e}")
            raise
    
    def __enter__(self):
        """上下文管理器入口"""
        self.connect()
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        """上下文管理器出口"""
        self.disconnect()


def register_application_simple(application_id: str, server_address: Optional[str] = None) -> bool:
    """
    简单的应用注册函数
    
    Args:
        application_id: 应用ID
        server_address: 服务器地址
        
    Returns:
        bool: 注册是否成功
    """
    try:
        with ApplicationRegistrationClient(server_address) as client:
            response = client.register_application(application_id)
            return response.success
    except Exception as e:
        print(f"Failed to register application {application_id}: {e}")
        return False


# 示例用法
if __name__ == "__main__":
    # 方式1: 使用上下文管理器
    with ApplicationRegistrationClient() as client:
        response = client.register_application("my-app-001")
        print(f"Registration successful: {response.success}")
    
    # 方式2: 使用简单函数
    success = register_application_simple("my-app-002")
    print(f"Simple registration successful: {success}")