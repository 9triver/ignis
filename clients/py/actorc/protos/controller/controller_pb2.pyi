import platform_pb2 as _platform_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CommandType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    UNSPECIFIED: _ClassVar[CommandType]
    ACK: _ClassVar[CommandType]
    FR_READY: _ClassVar[CommandType]
    FR_APPEND_DATA: _ClassVar[CommandType]
    FR_APPEND_ACTOR: _ClassVar[CommandType]
    FR_APPEND_PY_FUNC: _ClassVar[CommandType]
    FR_APPEND_PY_CLASS: _ClassVar[CommandType]
    FR_APPEND_ARG: _ClassVar[CommandType]
    FR_APPEND_CLASS_METHOD_ARG: _ClassVar[CommandType]
    FR_INVOKE: _ClassVar[CommandType]
    BK_RETURN_RESULT: _ClassVar[CommandType]
    FR_REGISTER_REQUEST: _ClassVar[CommandType]
    FR_DAG: _ClassVar[CommandType]
    FR_MARK_DAG_NODE_DONE: _ClassVar[CommandType]
    FR_REQUEST_OBJECT: _ClassVar[CommandType]
    BK_RESPONSE_OBJECT: _ClassVar[CommandType]
UNSPECIFIED: CommandType
ACK: CommandType
FR_READY: CommandType
FR_APPEND_DATA: CommandType
FR_APPEND_ACTOR: CommandType
FR_APPEND_PY_FUNC: CommandType
FR_APPEND_PY_CLASS: CommandType
FR_APPEND_ARG: CommandType
FR_APPEND_CLASS_METHOD_ARG: CommandType
FR_INVOKE: CommandType
BK_RETURN_RESULT: CommandType
FR_REGISTER_REQUEST: CommandType
FR_DAG: CommandType
FR_MARK_DAG_NODE_DONE: CommandType
FR_REQUEST_OBJECT: CommandType
BK_RESPONSE_OBJECT: CommandType

class Ack(_message.Message):
    __slots__ = ("Error",)
    ERROR_FIELD_NUMBER: _ClassVar[int]
    Error: str
    def __init__(self, Error: _Optional[str] = ...) -> None: ...

class Ready(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class Data(_message.Message):
    __slots__ = ("Type", "Ref", "Encoded")
    class ObjectType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        OBJ_UNSPECIFIED: _ClassVar[Data.ObjectType]
        OBJ_REF: _ClassVar[Data.ObjectType]
        OBJ_ENCODED: _ClassVar[Data.ObjectType]
        OBJ_STREAM: _ClassVar[Data.ObjectType]
    OBJ_UNSPECIFIED: Data.ObjectType
    OBJ_REF: Data.ObjectType
    OBJ_ENCODED: Data.ObjectType
    OBJ_STREAM: Data.ObjectType
    TYPE_FIELD_NUMBER: _ClassVar[int]
    REF_FIELD_NUMBER: _ClassVar[int]
    ENCODED_FIELD_NUMBER: _ClassVar[int]
    Type: Data.ObjectType
    Ref: _platform_pb2.Flow
    Encoded: _platform_pb2.EncodedObject
    def __init__(self, Type: _Optional[_Union[Data.ObjectType, str]] = ..., Ref: _Optional[_Union[_platform_pb2.Flow, _Mapping]] = ..., Encoded: _Optional[_Union[_platform_pb2.EncodedObject, _Mapping]] = ...) -> None: ...

class AppendData(_message.Message):
    __slots__ = ("SessionID", "Object")
    SESSIONID_FIELD_NUMBER: _ClassVar[int]
    OBJECT_FIELD_NUMBER: _ClassVar[int]
    SessionID: str
    Object: _platform_pb2.EncodedObject
    def __init__(self, SessionID: _Optional[str] = ..., Object: _Optional[_Union[_platform_pb2.EncodedObject, _Mapping]] = ...) -> None: ...

class AppendActor(_message.Message):
    __slots__ = ("Name", "Params", "Ref")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PARAMS_FIELD_NUMBER: _ClassVar[int]
    REF_FIELD_NUMBER: _ClassVar[int]
    Name: str
    Params: _containers.RepeatedScalarFieldContainer[str]
    Ref: _platform_pb2.ActorRef
    def __init__(self, Name: _Optional[str] = ..., Params: _Optional[_Iterable[str]] = ..., Ref: _Optional[_Union[_platform_pb2.ActorRef, _Mapping]] = ...) -> None: ...

class Resources(_message.Message):
    __slots__ = ("CPU", "Memory", "GPU")
    CPU_FIELD_NUMBER: _ClassVar[int]
    MEMORY_FIELD_NUMBER: _ClassVar[int]
    GPU_FIELD_NUMBER: _ClassVar[int]
    CPU: int
    Memory: int
    GPU: int
    def __init__(self, CPU: _Optional[int] = ..., Memory: _Optional[int] = ..., GPU: _Optional[int] = ...) -> None: ...

class AppendPyFunc(_message.Message):
    __slots__ = ("Name", "Params", "Venv", "Requirements", "PickledObject", "Language", "Resources", "Replicas")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PARAMS_FIELD_NUMBER: _ClassVar[int]
    VENV_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENTS_FIELD_NUMBER: _ClassVar[int]
    PICKLEDOBJECT_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    REPLICAS_FIELD_NUMBER: _ClassVar[int]
    Name: str
    Params: _containers.RepeatedScalarFieldContainer[str]
    Venv: str
    Requirements: _containers.RepeatedScalarFieldContainer[str]
    PickledObject: bytes
    Language: _platform_pb2.Language
    Resources: Resources
    Replicas: int
    def __init__(self, Name: _Optional[str] = ..., Params: _Optional[_Iterable[str]] = ..., Venv: _Optional[str] = ..., Requirements: _Optional[_Iterable[str]] = ..., PickledObject: _Optional[bytes] = ..., Language: _Optional[_Union[_platform_pb2.Language, str]] = ..., Resources: _Optional[_Union[Resources, _Mapping]] = ..., Replicas: _Optional[int] = ...) -> None: ...

class AppendPyClass(_message.Message):
    __slots__ = ("Name", "Methods", "Venv", "Requirements", "PickledObject", "Language", "Resources", "Replicas")
    class ClassMethod(_message.Message):
        __slots__ = ("Name", "Params")
        NAME_FIELD_NUMBER: _ClassVar[int]
        PARAMS_FIELD_NUMBER: _ClassVar[int]
        Name: str
        Params: _containers.RepeatedScalarFieldContainer[str]
        def __init__(self, Name: _Optional[str] = ..., Params: _Optional[_Iterable[str]] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    METHODS_FIELD_NUMBER: _ClassVar[int]
    VENV_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENTS_FIELD_NUMBER: _ClassVar[int]
    PICKLEDOBJECT_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    REPLICAS_FIELD_NUMBER: _ClassVar[int]
    Name: str
    Methods: _containers.RepeatedCompositeFieldContainer[AppendPyClass.ClassMethod]
    Venv: str
    Requirements: _containers.RepeatedScalarFieldContainer[str]
    PickledObject: bytes
    Language: _platform_pb2.Language
    Resources: Resources
    Replicas: int
    def __init__(self, Name: _Optional[str] = ..., Methods: _Optional[_Iterable[_Union[AppendPyClass.ClassMethod, _Mapping]]] = ..., Venv: _Optional[str] = ..., Requirements: _Optional[_Iterable[str]] = ..., PickledObject: _Optional[bytes] = ..., Language: _Optional[_Union[_platform_pb2.Language, str]] = ..., Resources: _Optional[_Union[Resources, _Mapping]] = ..., Replicas: _Optional[int] = ...) -> None: ...

class AppendArg(_message.Message):
    __slots__ = ("SessionID", "InstanceID", "Name", "Param", "Value")
    SESSIONID_FIELD_NUMBER: _ClassVar[int]
    INSTANCEID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    PARAM_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    SessionID: str
    InstanceID: str
    Name: str
    Param: str
    Value: Data
    def __init__(self, SessionID: _Optional[str] = ..., InstanceID: _Optional[str] = ..., Name: _Optional[str] = ..., Param: _Optional[str] = ..., Value: _Optional[_Union[Data, _Mapping]] = ...) -> None: ...

class AppendClassMethodArg(_message.Message):
    __slots__ = ("SessionID", "InstanceID", "MethodName", "Param", "Value")
    SESSIONID_FIELD_NUMBER: _ClassVar[int]
    INSTANCEID_FIELD_NUMBER: _ClassVar[int]
    METHODNAME_FIELD_NUMBER: _ClassVar[int]
    PARAM_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    SessionID: str
    InstanceID: str
    MethodName: str
    Param: str
    Value: Data
    def __init__(self, SessionID: _Optional[str] = ..., InstanceID: _Optional[str] = ..., MethodName: _Optional[str] = ..., Param: _Optional[str] = ..., Value: _Optional[_Union[Data, _Mapping]] = ...) -> None: ...

class Invoke(_message.Message):
    __slots__ = ("SessionID", "InstanceID", "Name")
    SESSIONID_FIELD_NUMBER: _ClassVar[int]
    INSTANCEID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SessionID: str
    InstanceID: str
    Name: str
    def __init__(self, SessionID: _Optional[str] = ..., InstanceID: _Optional[str] = ..., Name: _Optional[str] = ...) -> None: ...

class ReturnResult(_message.Message):
    __slots__ = ("SessionID", "InstanceID", "Name", "Value", "Error")
    SESSIONID_FIELD_NUMBER: _ClassVar[int]
    INSTANCEID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    SessionID: str
    InstanceID: str
    Name: str
    Value: Data
    Error: str
    def __init__(self, SessionID: _Optional[str] = ..., InstanceID: _Optional[str] = ..., Name: _Optional[str] = ..., Value: _Optional[_Union[Data, _Mapping]] = ..., Error: _Optional[str] = ...) -> None: ...

class RegisterRequest(_message.Message):
    __slots__ = ("ApplicationID",)
    APPLICATIONID_FIELD_NUMBER: _ClassVar[int]
    ApplicationID: str
    def __init__(self, ApplicationID: _Optional[str] = ...) -> None: ...

class ControlNode(_message.Message):
    __slots__ = ("Id", "Done", "FunctionName", "Params", "Current", "DataNode", "PreDataNodes", "FunctionType")
    class ParamsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    DONE_FIELD_NUMBER: _ClassVar[int]
    FUNCTIONNAME_FIELD_NUMBER: _ClassVar[int]
    PARAMS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_FIELD_NUMBER: _ClassVar[int]
    DATANODE_FIELD_NUMBER: _ClassVar[int]
    PREDATANODES_FIELD_NUMBER: _ClassVar[int]
    FUNCTIONTYPE_FIELD_NUMBER: _ClassVar[int]
    Id: str
    Done: bool
    FunctionName: str
    Params: _containers.ScalarMap[str, str]
    Current: int
    DataNode: str
    PreDataNodes: _containers.RepeatedScalarFieldContainer[str]
    FunctionType: str
    def __init__(self, Id: _Optional[str] = ..., Done: bool = ..., FunctionName: _Optional[str] = ..., Params: _Optional[_Mapping[str, str]] = ..., Current: _Optional[int] = ..., DataNode: _Optional[str] = ..., PreDataNodes: _Optional[_Iterable[str]] = ..., FunctionType: _Optional[str] = ...) -> None: ...

class DataNode(_message.Message):
    __slots__ = ("Id", "Done", "Lambda", "Ready", "SufControlNodes", "PreControlNode", "ParentNode", "ChildNode")
    ID_FIELD_NUMBER: _ClassVar[int]
    DONE_FIELD_NUMBER: _ClassVar[int]
    LAMBDA_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    SUFCONTROLNODES_FIELD_NUMBER: _ClassVar[int]
    PRECONTROLNODE_FIELD_NUMBER: _ClassVar[int]
    PARENTNODE_FIELD_NUMBER: _ClassVar[int]
    CHILDNODE_FIELD_NUMBER: _ClassVar[int]
    Id: str
    Done: bool
    Lambda: str
    Ready: bool
    SufControlNodes: _containers.RepeatedScalarFieldContainer[str]
    PreControlNode: str
    ParentNode: str
    ChildNode: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, Id: _Optional[str] = ..., Done: bool = ..., Lambda: _Optional[str] = ..., Ready: bool = ..., SufControlNodes: _Optional[_Iterable[str]] = ..., PreControlNode: _Optional[str] = ..., ParentNode: _Optional[str] = ..., ChildNode: _Optional[_Iterable[str]] = ...) -> None: ...

class DAGNode(_message.Message):
    __slots__ = ("Type", "ControlNode", "DataNode")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    CONTROLNODE_FIELD_NUMBER: _ClassVar[int]
    DATANODE_FIELD_NUMBER: _ClassVar[int]
    Type: str
    ControlNode: ControlNode
    DataNode: DataNode
    def __init__(self, Type: _Optional[str] = ..., ControlNode: _Optional[_Union[ControlNode, _Mapping]] = ..., DataNode: _Optional[_Union[DataNode, _Mapping]] = ...) -> None: ...

class DAG(_message.Message):
    __slots__ = ("Nodes",)
    NODES_FIELD_NUMBER: _ClassVar[int]
    Nodes: _containers.RepeatedCompositeFieldContainer[DAGNode]
    def __init__(self, Nodes: _Optional[_Iterable[_Union[DAGNode, _Mapping]]] = ...) -> None: ...

class MarkDAGNodeDone(_message.Message):
    __slots__ = ("NodeId", "SessionId")
    NODEID_FIELD_NUMBER: _ClassVar[int]
    SESSIONID_FIELD_NUMBER: _ClassVar[int]
    NodeId: str
    SessionId: str
    def __init__(self, NodeId: _Optional[str] = ..., SessionId: _Optional[str] = ...) -> None: ...

class RequestObject(_message.Message):
    __slots__ = ("ID", "Target")
    ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ID: str
    Target: str
    def __init__(self, ID: _Optional[str] = ..., Target: _Optional[str] = ...) -> None: ...

class ResponseObject(_message.Message):
    __slots__ = ("ID", "Value", "Error")
    ID_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    ID: str
    Value: _platform_pb2.EncodedObject
    Error: str
    def __init__(self, ID: _Optional[str] = ..., Value: _Optional[_Union[_platform_pb2.EncodedObject, _Mapping]] = ..., Error: _Optional[str] = ...) -> None: ...

class Message(_message.Message):
    __slots__ = ("Type", "Ack", "Ready", "AppendData", "AppendActor", "AppendPyFunc", "AppendPyClass", "AppendArg", "AppendClassMethodArg", "Invoke", "ReturnResult", "RegisterRequest", "DAG", "MarkDAGNodeDone", "RequestObject", "ResponseObject")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    ACK_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    APPENDDATA_FIELD_NUMBER: _ClassVar[int]
    APPENDACTOR_FIELD_NUMBER: _ClassVar[int]
    APPENDPYFUNC_FIELD_NUMBER: _ClassVar[int]
    APPENDPYCLASS_FIELD_NUMBER: _ClassVar[int]
    APPENDARG_FIELD_NUMBER: _ClassVar[int]
    APPENDCLASSMETHODARG_FIELD_NUMBER: _ClassVar[int]
    INVOKE_FIELD_NUMBER: _ClassVar[int]
    RETURNRESULT_FIELD_NUMBER: _ClassVar[int]
    REGISTERREQUEST_FIELD_NUMBER: _ClassVar[int]
    DAG_FIELD_NUMBER: _ClassVar[int]
    MARKDAGNODEDONE_FIELD_NUMBER: _ClassVar[int]
    REQUESTOBJECT_FIELD_NUMBER: _ClassVar[int]
    RESPONSEOBJECT_FIELD_NUMBER: _ClassVar[int]
    Type: CommandType
    Ack: Ack
    Ready: Ready
    AppendData: AppendData
    AppendActor: AppendActor
    AppendPyFunc: AppendPyFunc
    AppendPyClass: AppendPyClass
    AppendArg: AppendArg
    AppendClassMethodArg: AppendClassMethodArg
    Invoke: Invoke
    ReturnResult: ReturnResult
    RegisterRequest: RegisterRequest
    DAG: DAG
    MarkDAGNodeDone: MarkDAGNodeDone
    RequestObject: RequestObject
    ResponseObject: ResponseObject
    def __init__(self, Type: _Optional[_Union[CommandType, str]] = ..., Ack: _Optional[_Union[Ack, _Mapping]] = ..., Ready: _Optional[_Union[Ready, _Mapping]] = ..., AppendData: _Optional[_Union[AppendData, _Mapping]] = ..., AppendActor: _Optional[_Union[AppendActor, _Mapping]] = ..., AppendPyFunc: _Optional[_Union[AppendPyFunc, _Mapping]] = ..., AppendPyClass: _Optional[_Union[AppendPyClass, _Mapping]] = ..., AppendArg: _Optional[_Union[AppendArg, _Mapping]] = ..., AppendClassMethodArg: _Optional[_Union[AppendClassMethodArg, _Mapping]] = ..., Invoke: _Optional[_Union[Invoke, _Mapping]] = ..., ReturnResult: _Optional[_Union[ReturnResult, _Mapping]] = ..., RegisterRequest: _Optional[_Union[RegisterRequest, _Mapping]] = ..., DAG: _Optional[_Union[DAG, _Mapping]] = ..., MarkDAGNodeDone: _Optional[_Union[MarkDAGNodeDone, _Mapping]] = ..., RequestObject: _Optional[_Union[RequestObject, _Mapping]] = ..., ResponseObject: _Optional[_Union[ResponseObject, _Mapping]] = ...) -> None: ...
