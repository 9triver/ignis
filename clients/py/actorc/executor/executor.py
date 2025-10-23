import inspect
import queue
import sys
import threading
from collections.abc import Callable
from typing import Any, NamedTuple, Iterable

import zmq

from ..protos import platform_pb2 as platform
from ..protos.executor import executor_pb2 as executor
from ..utils import EncDec, serde
from ..utils.encdec import Streams
import json


class RemoteFunction(NamedTuple):
    language: platform.Language
    fn: Callable[..., Any]

    def call(self, *args, **kwargs):
        return self.fn(**kwargs)


class _DebugFuncs:
    @staticmethod
    def add(a: int, b: int) -> int:
        return a + b

    @staticmethod
    def sum(ints: Iterable[int]) -> int:
        print(ints, file=sys.stderr)
        s = 0
        for i in ints:
            s += i
        return s

    @staticmethod
    def gen(n: int) -> Iterable[int]:
        for i in range(n):
            yield i

    @staticmethod
    def map(ints: Iterable[int]) -> Iterable[int]:
        for i in ints:
            yield i * 2


class Executor:
    def __init__(self, name: str = "__system") -> None:
        self.name = name
        self.registries: dict[str, RemoteFunction] = {
            # for debugging purpose
            "__add": RemoteFunction(platform.LANG_JSON, _DebugFuncs.add),
            "__sum": RemoteFunction(platform.LANG_JSON, _DebugFuncs.sum),
            "__gen": RemoteFunction(platform.LANG_JSON, _DebugFuncs.gen),
            "__map": RemoteFunction(platform.LANG_JSON, _DebugFuncs.map),
        }
        self.send_q = queue.Queue[executor.Message | None]()

    def on_add_handler(self, cmd: executor.AddHandler):
        obj = serde.loads(cmd.Handler)
        if not callable(obj):
            print(f"function {cmd.Name} is not callable", file=sys.stderr)
            return
        self.registries[cmd.Name] = RemoteFunction(cmd.Language, obj)
        # 输出反序列后的函数签名
        try:
            sig = inspect.signature(obj)
            print(f"已注册反序列后的函数签名: {cmd.Name}{sig}", file=sys.stderr)
        except Exception as e:
            print(f"无法获取函数 {cmd.Name} 的签名: {e}", file=sys.stderr)
            return
        self.registries[cmd.Name] = RemoteFunction(cmd.Language, obj)

    def on_remove_handler(self, cmd: executor.RemoveHandler):
        if cmd.Name in self.registries:
            del self.registries[cmd.Name]

    def on_execute(self, cmd: executor.Execute):
        def send_chunk(obj: Any):
            try:
                enc = EncDec.encode(obj, func.language)
                chunk = platform.StreamChunk(StreamID=cmd.CorrID, Value=enc)
            except Exception as ex:
                chunk = platform.StreamChunk(
                    StreamID=cmd.CorrID, Error=f"{ex.__class__.__name__}: {ex}"
                )

            msg = executor.Message(
                Conn=self.name, Type=executor.STREAM_CHUNK, StreamChunk=chunk
            )
            self.send_q.put(msg)

        try:
            # print(cmd.Args, file=sys.stderr)
            print({k: v.ID for k, v in cmd.Args.items()}, file=sys.stderr)
            args = {k: EncDec.decode(v) for k, v in cmd.Args.items()}
            
            # Safe JSON serialization with error handling
            try:
                args_json = json.dumps(args, indent=2, ensure_ascii=False)
                print(f"function {cmd.Name} args {args_json}", file=sys.stderr)
            except (TypeError, ValueError) as json_err:
                print(f"function {cmd.Name} args (JSON serialization failed: {json_err}): {repr(args)}", file=sys.stderr)
            
            func = self.registries[cmd.Name]
            print(f"About to call function {cmd.Name}", file=sys.stderr)
            
            # Safe function call with additional error handling
            try:
                value = func.call(**args)
                print(f"function {cmd.Name} completed successfully", file=sys.stderr)
            except Exception as call_err:
                print(f"function {cmd.Name} call failed: {call_err}", file=sys.stderr)
                raise
                
            print(f"function {cmd.Name} returns {value}", file=sys.stderr)
            encoded = EncDec.encode(value, func.language)
            ret = executor.Return(CorrID=cmd.CorrID, Value=encoded)
            msg = executor.Message(Conn=self.name, Type=executor.D_RETURN, Return=ret)
            self.send_q.put(msg)

            if inspect.isgenerator(value):
                for obj in value:
                    send_chunk(obj)
                eos = platform.StreamChunk(StreamID=cmd.CorrID, EoS=True)
                msg = executor.Message(
                    Conn=self.name, Type=executor.STREAM_CHUNK, StreamChunk=eos
                )
                self.send_q.put(msg)
        except Exception as e:
            err = f"{e.__class__.__name__}: {e}"
            ret = executor.Return(CorrID=cmd.CorrID, Error=err)
            msg = executor.Message(Conn=self.name, Type=executor.D_RETURN, Return=ret)
            self.send_q.put(msg)

    @staticmethod
    def on_stream_chunk(cmd: platform.StreamChunk):
        if cmd.EoS:
            Streams.close(cmd.StreamID)
            return

        obj = EncDec.decode(cmd.Value)
        Streams.put(cmd.StreamID, obj)

    def loop(self, socket: zmq.SyncSocket):
        while True:
            msg = socket.recv()
            cmd = executor.Message.FromString(msg)
            match cmd.Type:
                case executor.R_ADD_HANDLER:
                    print("receive add handler", file=sys.stderr)
                    self.on_add_handler(cmd.AddHandler)
                case executor.R_REMOVE_HANDLER:
                    print("receive remove handler", file=sys.stderr)
                    self.on_remove_handler(cmd.RemoveHandler)
                case executor.R_EXECUTE:
                    print("receive execute", file=sys.stderr)
                    threading.Thread(
                        target=self.on_execute, args=(cmd.Execute,)
                    ).start()
                    # self.on_execute(cmd.Execute)
                case executor.R_EXIT:
                    self.send_q.put(None)
                    return
                case executor.STREAM_CHUNK:
                    self.on_stream_chunk(cmd.StreamChunk)
                case _:
                    print("unknown command type, ignoring", file=sys.stderr)

    def _start_send(self, socket: zmq.SyncSocket):
        while True:
            msg = self.send_q.get()
            self.send_q.task_done()
            if msg is None:
                break
            socket.send(msg.SerializeToString())

    def serve(self, ipc_addr: str):
        ctx = zmq.Context()
        socket = ctx.socket(zmq.DEALER)
        socket.connect(ipc_addr)
        data = executor.Ready()
        msg = executor.Message(Conn=self.name, Type=executor.D_READY, Ready=data)
        print(f"connected to {ipc_addr}", file=sys.stderr)
        socket.send(msg.SerializeToString())

        try:
            t = threading.Thread(target=self._start_send, args=(socket,))
            t.start()
            self.loop(socket)
        except Exception as e:
            print("executor stopped", e)
        finally:
            socket.close()
            ctx.term()
