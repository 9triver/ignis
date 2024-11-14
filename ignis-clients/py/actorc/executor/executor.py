from typing import Any, NamedTuple

import zmq

from ..protos import platform_pb2 as platform
from ..protos.ipc import dealer_pb2 as dealer
from ..protos.ipc import router_pb2 as router
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

    def add_handler(self, cmd: router.AddHandler) -> Any:
        obj = serde.loads(cmd.Handler)
        self.registries[cmd.Name] = RemoteFunction(cmd.Language, list(cmd.Methods), obj)
        return obj

    def remove_handler(self, cmd: router.RemoveHandler):
        if cmd.Name in self.registries:
            del self.registries[cmd.Name]

    def execute(self, cmd: router.Execute) -> dealer.Return:
        try:
            args = {k: EncDec.decode(v) for k, v in cmd.Args.items()}
            func = self.registries[cmd.Name]
            value = func.call(cmd.Method, **args)
            return dealer.Return(
                ID=cmd.ID,
                Value=EncDec.encode(value, func.language),
            )
        except Exception as e:
            return dealer.Return(
                ID=cmd.ID,
                Error=f"{e.__class__.__name__}: {e}",
            )

    def serve(self, ipc_addr: str):
        ctx = zmq.Context()
        socket = ctx.socket(zmq.DEALER)
        socket.connect(ipc_addr)
        data = dealer.Ready(Name=self.name)
        msg = dealer.DealerMessage(Command=dealer.DEALER_READY, Ready=data)
        print(f"connected to {ipc_addr}")
        socket.send(msg.SerializeToString())

        while True:
            msg = socket.recv()
            cmd = router.RouterMessage.FromString(msg)
            match cmd.Command:
                case router.ROUTER_ADD_HANDLER:
                    self.add_handler(cmd.AddHandler)
                case router.ROUTER_REMOVE_HANDLER:
                    self.remove_handler(cmd.RemoveHandler)
                case router.ROUTER_EXECUTE:
                    ret = self.execute(cmd.Execute)
                    msg = dealer.DealerMessage(Command=dealer.DEALER_RETURN, Return=ret)
                    socket.send(msg.SerializeToString())
                case router.ROUTER_EXIT:
                    break
                case _:
                    print("unknown command type, ignoring")

        socket.close()
        ctx.term()
