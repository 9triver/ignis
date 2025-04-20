import grpc
import cloudpickle
from lucas import function, Function, Runtime
from lucas.serverless_function import Metadata
import uuid
from ..protos.controller import controller_pb2 as pb, controller_pb2_grpc as prpc


class ActorRuntime(Runtime):
    def __init__(self, metadata: Metadata):
        super().__init__()
        self._input = metadata._params
        self._namespace = metadata._namespace
        self._router = metadata._router

    def input(self):
        return self._input

    def output(self, _out):
        return _out

    def get_result(self):
        return pb.ReturnResult()

    def call(self, fnName: str, fnParams: dict):
        print(f"call {fnName}")
        fn = self._router.get(fnName)
        if fn is None:
            raise ValueError(f"Function {fnName} not found in router")

        return {"function": fnName, "params": fnParams, "data": fn(fnParams)}

    def tell(self, fnName: str, fnParams: dict):
        print("tell function here")
        fn = self._router.get(fnName)
        if fn is None:
            raise ValueError(f"Function {fnName} not found in router")

        return {"function": fnName, "params": fnParams, "data": fn(fnParams)}


def generate(msg):
    yield msg


class ActorFunction(Function):
    def onFunctionInit(self, fn):
        dependcy = self._config.dependency
        fn_name = self._config.name
        params = self._config.params
        venv = self._config.venv
        obj = cloudpickle.dumps(self)
        with grpc.insecure_channel("localhost:50051") as channel:
            print("send")
            stub = prpc.ServiceStub(channel)
            message = pb.Message(
                Type=pb.CommandType.FR_APPEND_PY_FUNC,
                AppendPyFunc=pb.AppendPyFunc(
                    Name=fn_name,
                    Params=params,
                    Venv=venv,
                    Requirements=dependcy,
                    PickledObject=obj,
                ),
            )
            response_stream = stub.Session(generate(message))
            for response in response_stream:
                print(response)

    def _transformfunction(self, fn):
        return fn

    def __call__(self, data: dict):
        metadata = Metadata(
            id=str(uuid.uuid4()),
            params=data,
            namespace=None,
            router={},
            request_type="invoke",
            redis_db=None,
            producer=None,
        )
        rt = ActorRuntime(metadata)
        return self._fn(rt)


@function(
    wrapper=ActorFunction,
    dependency=["torch", "numpy"],
    provider="actor",
    name="funca",
    params=["a"],
    venv="conda",
)
def funca(rt: Runtime):
    return rt.output(rt.input())
