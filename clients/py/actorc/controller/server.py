import grpc
import cloudpickle
from concurrent import futures
from ..protos.controller import controller_pb2, controller_pb2_grpc
from ..protos import platform_pb2


class FunctionToExecute:
    def __init__(self, fn, params):
        self._fn = fn
        self._args = {}
        self._params = params

    def set_args(self, data: dict):
        for key, val in data.items():
            self._args[key] = val

    def can_run(self) -> bool:
        for params in self._params:
            if params not in self._args:
                return False
        return True

    def run(self):
        result = self._fn(self._args)
        return result


class GRPCServer(controller_pb2_grpc.ServiceServicer):
    def __init__(self):
        self._funcs = {}
        self._data_obj = {}

    def Session(self, request_iterator, context):
        for request in request_iterator:
            request: controller_pb2.Message
            if request.Type == controller_pb2.CommandType.FR_APPEND_PY_FUNC:
                pyFunc = request.AppendPyFunc
                name = pyFunc.Name
                params = pyFunc.Params
                fn = pyFunc.PickledObject
                fn = cloudpickle.loads(fn)

                func = FunctionToExecute(fn, params)
                self._funcs[name] = func
                print(f"Function {name} registered")

                response = controller_pb2.Message(
                    Type=controller_pb2.CommandType.ACK,
                    Ack=controller_pb2.Ack(Error=""),
                )
                yield response
            elif request.Type == controller_pb2.CommandType.FR_APPEND_ARG:
                appendArg = request.AppendArg
                sessionID = appendArg.SessionID
                instanceID = appendArg.InstanceID
                functionname = appendArg.Name
                args_name = appendArg.Param
                value: controller_pb2.Data = appendArg.Value
                if value.Type == controller_pb2.Data.ObjectType.OBJ_REF:
                    ref = value.Ref
                    data = self._data_obj[ref.ObjectID]
                elif value.Type == controller_pb2.Data.ObjectType.OBJ_ENCODED:
                    data = value.Encoded
                    data = data.Data
                    data = cloudpickle.loads(data)
                func: FunctionToExecute = self._funcs[functionname]
                func.set_args({args_name: data})
                if func.can_run():
                    result = func.run()
                    print(result)
                    key = f"{sessionID}-{instanceID}-{functionname}"
                    self._data_obj[key] = result
                    response = controller_pb2.Message(
                        Type=controller_pb2.CommandType.BK_RETURN_RESULT,
                        ReturnResult=controller_pb2.ReturnResult(
                            SessionID=sessionID,
                            InstanceID=instanceID,
                            Name=functionname,
                            Value=platform_pb2.Flow(ObjectID=key, Source={}),
                        ),
                    )
                    yield response
                else:
                    response = controller_pb2.Message(
                        Type=controller_pb2.CommandType.ACK,
                        Ack=controller_pb2.Ack(Error=""),
                    )
                    yield response


server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
controller_pb2_grpc.add_ServiceServicer_to_server(GRPCServer(), server)
server.add_insecure_port("[::]:50051")
server.start()
print("start")
try:
    import time

    time.sleep(3600)
except KeyboardInterrupt:
    server.stop(0)
