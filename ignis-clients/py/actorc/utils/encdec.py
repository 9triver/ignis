import base64
import json
from typing import Any, TypedDict

from ..protos import platform_pb2 as platform
from . import serde

LANG_PYTHON = platform.LANG_PYTHON
LANG_GO = platform.LANG_GO
LANG_JSON = platform.LANG_JSON


class EncodedObject(TypedDict):
    Data: str
    Language: int


class EncDec:
    @staticmethod
    def decode(obj: platform.EncodedObject):
        data = obj.Data
        match obj.Language:
            case platform.LANG_PYTHON:
                return serde.loads(data)
            case platform.LANG_JSON:
                return json.loads(data)
            case _:
                raise ValueError(f"unsupported language {obj.Language}")

    @staticmethod
    def decode_dict(obj: EncodedObject):
        data = base64.decodebytes(obj["Data"].encode())
        match lang := obj["Language"]:
            case platform.LANG_PYTHON:
                return serde.loads(data)
            case platform.LANG_JSON:
                return json.loads(data)
            case _:
                raise ValueError(f"unsupported language {lang}")

    @staticmethod
    def encode(obj: Any, language: platform.Language = LANG_JSON):
        match language:
            case platform.LANG_PYTHON:
                data = serde.dumps(obj)
            case platform.LANG_JSON:
                data = json.dumps(obj).encode()
            case _:
                raise ValueError(f"unsupported language {language}")
        return platform.EncodedObject(Data=data, Language=language)
