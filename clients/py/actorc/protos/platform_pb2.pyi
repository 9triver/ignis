import actor_pb2 as _actor_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Language(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LANG_UNKNOWN: _ClassVar[Language]
    LANG_JSON: _ClassVar[Language]
    LANG_GO: _ClassVar[Language]
    LANG_PYTHON: _ClassVar[Language]
LANG_UNKNOWN: Language
LANG_JSON: Language
LANG_GO: Language
LANG_PYTHON: Language

class StoreRef(_message.Message):
    __slots__ = ("ID", "PID")
    ID_FIELD_NUMBER: _ClassVar[int]
    PID_FIELD_NUMBER: _ClassVar[int]
    ID: str
    PID: _actor_pb2.PID
    def __init__(self, ID: _Optional[str] = ..., PID: _Optional[_Union[_actor_pb2.PID, _Mapping]] = ...) -> None: ...

class ActorRef(_message.Message):
    __slots__ = ("Store", "ID", "PID")
    STORE_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    PID_FIELD_NUMBER: _ClassVar[int]
    Store: StoreRef
    ID: str
    PID: _actor_pb2.PID
    def __init__(self, Store: _Optional[_Union[StoreRef, _Mapping]] = ..., ID: _Optional[str] = ..., PID: _Optional[_Union[_actor_pb2.PID, _Mapping]] = ...) -> None: ...

class ActorInfo(_message.Message):
    __slots__ = ("Ref", "CalcLatency", "LinkLatency")
    REF_FIELD_NUMBER: _ClassVar[int]
    CALCLATENCY_FIELD_NUMBER: _ClassVar[int]
    LINKLATENCY_FIELD_NUMBER: _ClassVar[int]
    Ref: ActorRef
    CalcLatency: int
    LinkLatency: int
    def __init__(self, Ref: _Optional[_Union[ActorRef, _Mapping]] = ..., CalcLatency: _Optional[int] = ..., LinkLatency: _Optional[int] = ...) -> None: ...

class Flow(_message.Message):
    __slots__ = ("ID", "Source")
    ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    ID: str
    Source: StoreRef
    def __init__(self, ID: _Optional[str] = ..., Source: _Optional[_Union[StoreRef, _Mapping]] = ...) -> None: ...

class EncodedObject(_message.Message):
    __slots__ = ("ID", "Data", "Source", "Language", "Stream")
    ID_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    STREAM_FIELD_NUMBER: _ClassVar[int]
    ID: str
    Data: bytes
    Source: StoreRef
    Language: Language
    Stream: bool
    def __init__(self, ID: _Optional[str] = ..., Data: _Optional[bytes] = ..., Source: _Optional[_Union[StoreRef, _Mapping]] = ..., Language: _Optional[_Union[Language, str]] = ..., Stream: bool = ...) -> None: ...

class StreamChunk(_message.Message):
    __slots__ = ("StreamID", "Target", "EoS", "Value", "Error")
    STREAMID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    EOS_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    StreamID: str
    Target: str
    EoS: bool
    Value: EncodedObject
    Error: str
    def __init__(self, StreamID: _Optional[str] = ..., Target: _Optional[str] = ..., EoS: bool = ..., Value: _Optional[_Union[EncodedObject, _Mapping]] = ..., Error: _Optional[str] = ...) -> None: ...

class Invoke(_message.Message):
    __slots__ = ("Target", "SessionID", "Param", "Value")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SESSIONID_FIELD_NUMBER: _ClassVar[int]
    PARAM_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    Target: str
    SessionID: str
    Param: str
    Value: Flow
    def __init__(self, Target: _Optional[str] = ..., SessionID: _Optional[str] = ..., Param: _Optional[str] = ..., Value: _Optional[_Union[Flow, _Mapping]] = ...) -> None: ...

class InvokeStart(_message.Message):
    __slots__ = ("Info", "SessionID", "ReplyTo")
    INFO_FIELD_NUMBER: _ClassVar[int]
    SESSIONID_FIELD_NUMBER: _ClassVar[int]
    REPLYTO_FIELD_NUMBER: _ClassVar[int]
    Info: ActorInfo
    SessionID: str
    ReplyTo: str
    def __init__(self, Info: _Optional[_Union[ActorInfo, _Mapping]] = ..., SessionID: _Optional[str] = ..., ReplyTo: _Optional[str] = ...) -> None: ...

class InvokeResponse(_message.Message):
    __slots__ = ("Target", "SessionID", "Result", "Error", "Info")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SESSIONID_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    INFO_FIELD_NUMBER: _ClassVar[int]
    Target: str
    SessionID: str
    Result: Flow
    Error: str
    Info: ActorInfo
    def __init__(self, Target: _Optional[str] = ..., SessionID: _Optional[str] = ..., Result: _Optional[_Union[Flow, _Mapping]] = ..., Error: _Optional[str] = ..., Info: _Optional[_Union[ActorInfo, _Mapping]] = ...) -> None: ...
