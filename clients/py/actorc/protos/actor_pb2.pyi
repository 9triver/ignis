from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TerminatedReason(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    Stopped: _ClassVar[TerminatedReason]
    AddressTerminated: _ClassVar[TerminatedReason]
    NotFound: _ClassVar[TerminatedReason]
Stopped: TerminatedReason
AddressTerminated: TerminatedReason
NotFound: TerminatedReason

class PID(_message.Message):
    __slots__ = ("Address", "Id", "request_id")
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    Address: str
    Id: str
    request_id: int
    def __init__(self, Address: _Optional[str] = ..., Id: _Optional[str] = ..., request_id: _Optional[int] = ...) -> None: ...

class PoisonPill(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class DeadLetterResponse(_message.Message):
    __slots__ = ("Target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    Target: PID
    def __init__(self, Target: _Optional[_Union[PID, _Mapping]] = ...) -> None: ...

class Watch(_message.Message):
    __slots__ = ("Watcher",)
    WATCHER_FIELD_NUMBER: _ClassVar[int]
    Watcher: PID
    def __init__(self, Watcher: _Optional[_Union[PID, _Mapping]] = ...) -> None: ...

class Unwatch(_message.Message):
    __slots__ = ("Watcher",)
    WATCHER_FIELD_NUMBER: _ClassVar[int]
    Watcher: PID
    def __init__(self, Watcher: _Optional[_Union[PID, _Mapping]] = ...) -> None: ...

class Terminated(_message.Message):
    __slots__ = ("who", "Why")
    WHO_FIELD_NUMBER: _ClassVar[int]
    WHY_FIELD_NUMBER: _ClassVar[int]
    who: PID
    Why: TerminatedReason
    def __init__(self, who: _Optional[_Union[PID, _Mapping]] = ..., Why: _Optional[_Union[TerminatedReason, str]] = ...) -> None: ...

class Stop(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class Touch(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class Touched(_message.Message):
    __slots__ = ("who",)
    WHO_FIELD_NUMBER: _ClassVar[int]
    who: PID
    def __init__(self, who: _Optional[_Union[PID, _Mapping]] = ...) -> None: ...
