from lucas import Runtime, Function
from lucas.serverless_function import Metadata
from lucas.workflow.executor import Executor
from lucas.workflow.dag import DAGNode, DataNode, ControlNode
from lucas.utils.logging import log

from ..protos.controller import controller_pb2, controller_pb2_grpc
from ..protos import platform_pb2
from ..utils import EncDec

import cloudpickle
import grpc
import uuid
import queue
import time
import inspect
import threading
import os

actorContext: "ActorContext | None" = None


class ActorContext:
    @staticmethod
    def createContext(master_address: str = None):
        global actorContext
        if actorContext is None:
            if master_address is None:
                master_address = os.getenv("IGNIS_ADDR", "localhost:50051")
            actorContext = ActorContext(master_address)
        return actorContext

    def __init__(self, master_address: str = None):
        if master_address is None:
            master_address = os.getenv("IGNIS_ADDR", "localhost:50051")
        self._master_address = master_address
        self._channel = grpc.insecure_channel(
            master_address,
            options=[("grpc.max_receive_message_length", 512 * 1024 * 1024)],
        )
        self._stub = controller_pb2_grpc.ServiceStub(self._channel)
        self._q = queue.Queue()
        self._response_stream = self._stub.Session(self._generate())
        self._result_map: dict[str, platform_pb2.Flow] = {}
        self._thread = threading.Thread(target=self._run, daemon=True)
        self._thread.start()

    def _generate(self):
        while True:
            msg = self._q.get()
            yield msg

    def _run(self):
        while True:
            for response in self._response_stream:
                response: controller_pb2.Message
                if response.Type == controller_pb2.CommandType.BK_RETURN_RESULT:
                    result: controller_pb2.ReturnResult = response.ReturnResult
                    sessionID = result.SessionID
                    instanceID = result.InstanceID
                    name = result.Name
                    value = result.Value
                    key = f"{sessionID}-{instanceID}-{name}"
                    self._result_map[key] = value
            time.sleep(1)

    def get_result(self, key: str) -> platform_pb2.Flow:
        return self._result_map.get(key)

    def send(self, message: controller_pb2.Message):
        self._q.put(message)


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

    def call(self, fnName: str, fnParams: dict) -> platform_pb2.Flow:
        print(f"call {fnName}")
        sessionID = fnParams["sessionID"]
        instanceID = fnParams["instanceID"]
        name = fnParams["name"]
        key = f"{sessionID}-{instanceID}-{name}"
        result = None
        while result is None:
            result = actorContext.get_result(key)
            if result is None:
                time.sleep(1)
        return result

    def tell(self, fnName: str, fnParams: dict):
        print("tell function here")
        fn = self._router.get(fnName)
        if fn is None:
            raise ValueError(f"Function {fnName} not found in router")

        return {"function": fnName, "params": fnParams, "data": fn(fnParams)}


class ActorFunction(Function):
    def onFunctionInit(self, fn):
        dependcy = self._config.dependency
        fn_name = self._config.name
        venv = self._config.venv
        try:
            replicas = self._config.replicas
        except AttributeError:
            replicas = 1
        sig = inspect.signature(fn)
        params = []
        for name, param in sig.parameters.items():
            params.append(name)
        print("pickle function here")
        message = controller_pb2.Message(
            Type=controller_pb2.CommandType.FR_APPEND_PY_FUNC,
            AppendPyFunc=controller_pb2.AppendPyFunc(
                Name=fn_name,
                Params=params,
                Venv=venv,
                Requirements=dependcy,
                PickledObject=cloudpickle.dumps(self._fn),
                Language=platform_pb2.LANG_PYTHON,
                Replicas=replicas,
            ),
        )
        actorContext.send(message)

    def _transformfunction(self, fn):
        return fn


class ActorExecutor(Executor):
    def __init__(self, dag):
        super().__init__(dag)

    def execute(self):
        session_id = str(uuid.uuid4())
        while not self.dag.hasDone():
            task: list[DAGNode] = []
            for node in self.dag.get_nodes():
                if node._done:
                    continue
                if isinstance(node, DataNode):
                    if node._ready:
                        task.append(node)
                if isinstance(node, ControlNode):
                    if node.get_pre_data_nodes() == []:
                        task.append(node)

            _end = False
            while len(task) != 0:
                node = task.pop(0)
                node._done = True
                if isinstance(node, DataNode):
                    for control_node in node.get_succ_control_nodes():
                        control_node: ControlNode
                        control_node_metadata = control_node.metadata()
                        params = control_node_metadata["params"]
                        fn_type = control_node_metadata["functiontype"]
                        data = node._ld.value

                        if fn_type == "remote":  # 要调用的函数是远程函数时才需要
                            if isinstance(data, controller_pb2.Data):
                                rpc_data = data
                            else:
                                rpc_data = controller_pb2.Data(
                                    Type=controller_pb2.Data.ObjectType.OBJ_ENCODED,
                                    Encoded=EncDec.encode(
                                        data, language=platform_pb2.LANG_PYTHON
                                    ),
                                )
                            # if (
                            #     data_type == controller_pb2.Data.ObjectType.OBJ_ENCODED
                            # ):  # 如果是实际值，就要序列化
                            #     data = cloudpickle.dumps(data)
                            #     rpc_data = controller_pb2.Data(
                            #         Type=data_type,
                            #         Encoded=platform_pb2.EncodedObject(
                            #             ID="",
                            #             Data=data,
                            #             Language=platform_pb2.Language.LANG_PYTHON,
                            #         ),
                            #     )
                            # else:
                            #     rpc_data = controller_pb2.Data(Type=data_type, Ref=data)

                            appendArg = controller_pb2.AppendArg(
                                SessionID=session_id,
                                InstanceID=control_node_metadata["id"],
                                Name=control_node_metadata["functionname"],
                                Param=params[node._ld.getid()],
                                Value=rpc_data,
                            )
                            message = controller_pb2.Message(
                                Type=controller_pb2.CommandType.FR_APPEND_ARG,
                                AppendArg=appendArg,
                            )
                            actorContext.send(message)

                        log.info(f"{control_node.describe()} appargs {node._ld.value}")
                        if control_node.appargs(node._ld):
                            if control_node._fn_type == "remote":
                                control_node._datas["sessionID"] = session_id
                                control_node._datas["instanceID"] = (
                                    control_node_metadata["id"]
                                )
                                control_node._datas["name"] = control_node_metadata[
                                    "functionname"
                                ]
                            task.append(control_node)
                elif isinstance(node, ControlNode):
                    fn = node._fn
                    params = node._datas
                    r_node: DataNode = node.get_data_node()
                    result = fn(params)
                    if node._fn_type == "local":
                        r_node.set_value(result)
                    elif node._fn_type == "remote":
                        r_node.set_value(result)
                    r_node.set_ready()
                    log.info(f"{node.describe()} calculate {r_node.describe()}")
                    if r_node.is_ready():
                        task.append(r_node)
            if _end:
                break
        result = None
        for node in self.dag.get_nodes():
            if isinstance(node, DataNode) and node._is_end_node:
                result = node._ld.value
                break
        self.dag.reset()
        return result
