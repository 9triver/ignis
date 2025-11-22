import platform_pb2 as _platform_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CommandType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    UNSPECIFIED: _ClassVar[CommandType]
    R_ADD_HANDLER: _ClassVar[CommandType]
    R_REMOVE_HANDLER: _ClassVar[CommandType]
    R_EXECUTE: _ClassVar[CommandType]
    R_EXIT: _ClassVar[CommandType]
    D_READY: _ClassVar[CommandType]
    D_RETURN: _ClassVar[CommandType]
    STREAM_CHUNK: _ClassVar[CommandType]
UNSPECIFIED: CommandType
R_ADD_HANDLER: CommandType
R_REMOVE_HANDLER: CommandType
R_EXECUTE: CommandType
R_EXIT: CommandType
D_READY: CommandType
D_RETURN: CommandType
STREAM_CHUNK: CommandType

class AddHandler(_message.Message):
    __slots__ = ("Name", "Handler", "Language", "Methods")
    NAME_FIELD_NUMBER: _ClassVar[int]
    HANDLER_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    METHODS_FIELD_NUMBER: _ClassVar[int]
    Name: str
    Handler: bytes
    Language: _platform_pb2.Language
    Methods: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, Name: _Optional[str] = ..., Handler: _Optional[bytes] = ..., Language: _Optional[_Union[_platform_pb2.Language, str]] = ..., Methods: _Optional[_Iterable[str]] = ...) -> None: ...

class RemoveHandler(_message.Message):
    __slots__ = ("Name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    Name: str
    def __init__(self, Name: _Optional[str] = ...) -> None: ...

class Execute(_message.Message):
    __slots__ = ("CorrID", "Name", "Method", "Args")
    class ArgsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _platform_pb2.EncodedObject
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_platform_pb2.EncodedObject, _Mapping]] = ...) -> None: ...
    CORRID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    CorrID: str
    Name: str
    Method: str
    Args: _containers.MessageMap[str, _platform_pb2.EncodedObject]
    def __init__(self, CorrID: _Optional[str] = ..., Name: _Optional[str] = ..., Method: _Optional[str] = ..., Args: _Optional[_Mapping[str, _platform_pb2.EncodedObject]] = ...) -> None: ...

class Exit(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class Ready(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class Return(_message.Message):
    __slots__ = ("CorrID", "Value", "Error")
    CORRID_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    CorrID: str
    Value: _platform_pb2.EncodedObject
    Error: str
    def __init__(self, CorrID: _Optional[str] = ..., Value: _Optional[_Union[_platform_pb2.EncodedObject, _Mapping]] = ..., Error: _Optional[str] = ...) -> None: ...

class Message(_message.Message):
    __slots__ = ("Conn", "Type", "AddHandler", "RemoveHandler", "Execute", "Exit", "Ready", "Return", "StreamChunk")
    CONN_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    ADDHANDLER_FIELD_NUMBER: _ClassVar[int]
    REMOVEHANDLER_FIELD_NUMBER: _ClassVar[int]
    EXECUTE_FIELD_NUMBER: _ClassVar[int]
    EXIT_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    RETURN_FIELD_NUMBER: _ClassVar[int]
    STREAMCHUNK_FIELD_NUMBER: _ClassVar[int]
    Conn: str
    Type: CommandType
    AddHandler: AddHandler
    RemoveHandler: RemoveHandler
    Execute: Execute
    Exit: Exit
    Ready: Ready
    Return: Return
    StreamChunk: _platform_pb2.StreamChunk
    def __init__(self, Conn: _Optional[str] = ..., Type: _Optional[_Union[CommandType, str]] = ..., AddHandler: _Optional[_Union[AddHandler, _Mapping]] = ..., RemoveHandler: _Optional[_Union[RemoveHandler, _Mapping]] = ..., Execute: _Optional[_Union[Execute, _Mapping]] = ..., Exit: _Optional[_Union[Exit, _Mapping]] = ..., Ready: _Optional[_Union[Ready, _Mapping]] = ..., Return: _Optional[_Union[Return, _Mapping]] = ..., StreamChunk: _Optional[_Union[_platform_pb2.StreamChunk, _Mapping]] = ...) -> None: ...
