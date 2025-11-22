import platform_pb2 as _platform_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class MessageType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    UNSPECIFIED: _ClassVar[MessageType]
    ACK: _ClassVar[MessageType]
    READY: _ClassVar[MessageType]
    INVOKE: _ClassVar[MessageType]
    INVOKE_START: _ClassVar[MessageType]
    INVOKE_RESPONSE: _ClassVar[MessageType]
    OBJECT_REQUEST: _ClassVar[MessageType]
    OBJECT_RESPONSE: _ClassVar[MessageType]
    STREAM_CHUNK: _ClassVar[MessageType]
    FUNCTION: _ClassVar[MessageType]
UNSPECIFIED: MessageType
ACK: MessageType
READY: MessageType
INVOKE: MessageType
INVOKE_START: MessageType
INVOKE_RESPONSE: MessageType
OBJECT_REQUEST: MessageType
OBJECT_RESPONSE: MessageType
STREAM_CHUNK: MessageType
FUNCTION: MessageType

class ObjectRequest(_message.Message):
    __slots__ = ("ID", "Target", "ReplyTo")
    ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    REPLYTO_FIELD_NUMBER: _ClassVar[int]
    ID: str
    Target: str
    ReplyTo: str
    def __init__(self, ID: _Optional[str] = ..., Target: _Optional[str] = ..., ReplyTo: _Optional[str] = ...) -> None: ...

class ObjectResponse(_message.Message):
    __slots__ = ("ID", "Target", "Value", "Error")
    ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    ID: str
    Target: str
    Value: _platform_pb2.EncodedObject
    Error: str
    def __init__(self, ID: _Optional[str] = ..., Target: _Optional[str] = ..., Value: _Optional[_Union[_platform_pb2.EncodedObject, _Mapping]] = ..., Error: _Optional[str] = ...) -> None: ...

class Envelope(_message.Message):
    __slots__ = ("Store", "Type", "ObjectRequest", "ObjectResponse", "StreamChunk")
    STORE_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    OBJECTREQUEST_FIELD_NUMBER: _ClassVar[int]
    OBJECTRESPONSE_FIELD_NUMBER: _ClassVar[int]
    STREAMCHUNK_FIELD_NUMBER: _ClassVar[int]
    Store: _platform_pb2.StoreRef
    Type: MessageType
    ObjectRequest: ObjectRequest
    ObjectResponse: ObjectResponse
    StreamChunk: _platform_pb2.StreamChunk
    def __init__(self, Store: _Optional[_Union[_platform_pb2.StoreRef, _Mapping]] = ..., Type: _Optional[_Union[MessageType, str]] = ..., ObjectRequest: _Optional[_Union[ObjectRequest, _Mapping]] = ..., ObjectResponse: _Optional[_Union[ObjectResponse, _Mapping]] = ..., StreamChunk: _Optional[_Union[_platform_pb2.StreamChunk, _Mapping]] = ...) -> None: ...

class Ack(_message.Message):
    __slots__ = ("Error",)
    ERROR_FIELD_NUMBER: _ClassVar[int]
    Error: str
    def __init__(self, Error: _Optional[str] = ...) -> None: ...

class Ready(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class Function(_message.Message):
    __slots__ = ("Name", "Params", "Requirements", "PickledObject", "Language")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PARAMS_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENTS_FIELD_NUMBER: _ClassVar[int]
    PICKLEDOBJECT_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    Name: str
    Params: _containers.RepeatedScalarFieldContainer[str]
    Requirements: _containers.RepeatedScalarFieldContainer[str]
    PickledObject: bytes
    Language: _platform_pb2.Language
    def __init__(self, Name: _Optional[str] = ..., Params: _Optional[_Iterable[str]] = ..., Requirements: _Optional[_Iterable[str]] = ..., PickledObject: _Optional[bytes] = ..., Language: _Optional[_Union[_platform_pb2.Language, str]] = ...) -> None: ...

class Message(_message.Message):
    __slots__ = ("Type", "ConnID", "Ack", "Ready", "Invoke", "InvokeStart", "InvokeResponse", "ObjectRequest", "ObjectResponse", "StreamChunk", "Function")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    CONNID_FIELD_NUMBER: _ClassVar[int]
    ACK_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    INVOKE_FIELD_NUMBER: _ClassVar[int]
    INVOKESTART_FIELD_NUMBER: _ClassVar[int]
    INVOKERESPONSE_FIELD_NUMBER: _ClassVar[int]
    OBJECTREQUEST_FIELD_NUMBER: _ClassVar[int]
    OBJECTRESPONSE_FIELD_NUMBER: _ClassVar[int]
    STREAMCHUNK_FIELD_NUMBER: _ClassVar[int]
    FUNCTION_FIELD_NUMBER: _ClassVar[int]
    Type: MessageType
    ConnID: str
    Ack: Ack
    Ready: Ready
    Invoke: _platform_pb2.Invoke
    InvokeStart: _platform_pb2.InvokeStart
    InvokeResponse: _platform_pb2.InvokeResponse
    ObjectRequest: ObjectRequest
    ObjectResponse: ObjectResponse
    StreamChunk: _platform_pb2.StreamChunk
    Function: Function
    def __init__(self, Type: _Optional[_Union[MessageType, str]] = ..., ConnID: _Optional[str] = ..., Ack: _Optional[_Union[Ack, _Mapping]] = ..., Ready: _Optional[_Union[Ready, _Mapping]] = ..., Invoke: _Optional[_Union[_platform_pb2.Invoke, _Mapping]] = ..., InvokeStart: _Optional[_Union[_platform_pb2.InvokeStart, _Mapping]] = ..., InvokeResponse: _Optional[_Union[_platform_pb2.InvokeResponse, _Mapping]] = ..., ObjectRequest: _Optional[_Union[ObjectRequest, _Mapping]] = ..., ObjectResponse: _Optional[_Union[ObjectResponse, _Mapping]] = ..., StreamChunk: _Optional[_Union[_platform_pb2.StreamChunk, _Mapping]] = ..., Function: _Optional[_Union[Function, _Mapping]] = ...) -> None: ...
