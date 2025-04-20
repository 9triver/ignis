from typing import Any, NamedTuple
import sys

import zmq

from ..protos import platform_pb2 as platform
from ..protos.executor import executor_pb2 as executor
from ..utils import EncDec, serde


class RemoteFunction(NamedTuple):
    language: platform.Language
    methods: list[str]
    obj: Any

    def call(self, method: str, **args):
        if method == "":
            return self.obj(**args)

        if method not in self.methods:
            raise KeyError(f"No such method exposed {method}")

        func = getattr(self.obj, method)
        return func(**args)


class Executor:
    def __init__(self, name: str = "__system") -> None:
        self.name = name
        self.registries: dict[str, RemoteFunction] = {}

    def add_handler(self, cmd: executor.AddHandler) -> Any:
        obj = serde.loads(cmd.Handler)
        self.registries[cmd.Name] = RemoteFunction(cmd.Language, list(cmd.Methods), obj)
        return obj

    def remove_handler(self, cmd: executor.RemoveHandler):
        if cmd.Name in self.registries:
            del self.registries[cmd.Name]

    def execute(self, cmd: executor.Execute) -> executor.Return:
        try:
            args = {k: EncDec.decode(v) for k, v in cmd.Args.items()}
            func = self.registries[cmd.Name]
            value = func.call(cmd.Method, **args)
            return executor.Return(
                CorrID=cmd.CorrID,
                Value=EncDec.encode(value, func.language),
            )
        except Exception as e:
            return executor.Return(
                CorrID=cmd.CorrID,
                Error=f"{e.__class__.__name__}: {e}",
            )

    def loop(self, socket: zmq.SyncSocket):
        while True:
            msg = socket.recv()
            cmd = executor.Message.FromString(msg)
            match cmd.Type:
                case executor.R_ADD_HANDLER:
                    print("receive add handler", file=sys.stderr)
                    self.add_handler(cmd.AddHandler)
                case executor.R_REMOVE_HANDLER:
                    print("receive remove handler", file=sys.stderr)
                    self.remove_handler(cmd.RemoveHandler)
                case executor.R_EXECUTE:
                    print("receive execute", file=sys.stderr)
                    ret = self.execute(cmd.Execute)
                    msg = executor.Message(
                        Conn=self.name, Type=executor.D_RETURN, Return=ret
                    )
                    print(f"sending back {msg}", file=sys.stderr)
                    socket.send(msg.SerializeToString())
                case executor.R_EXIT:
                    return
                case _:
                    print("unknown command type, ignoring", file=sys.stderr)

    def serve(self, ipc_addr: str):
        ctx = zmq.Context()
        socket = ctx.socket(zmq.DEALER)
        socket.connect(ipc_addr)
        data = executor.Ready()
        msg = executor.Message(Conn=self.name, Type=executor.D_READY, Ready=data)
        print(f"connected to {ipc_addr}", file=sys.stderr)
        socket.send(msg.SerializeToString())

        try:
            self.loop(socket)
        except Exception as e:
            print("executor stopped", e)
        finally:
            socket.close()
            ctx.term()
