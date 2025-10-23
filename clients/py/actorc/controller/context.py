from lucas import Function, Runtime
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
import re

actorContext: "ActorContext | None" = None


def parse_memory_string(memory_str):
    """
    将Kubernetes风格的内存字符串转换为字节数
    支持的单位: K, M, G, T, P, E (1000进制) 和 Ki, Mi, Gi, Ti, Pi, Ei (1024进制)

    Args:
        memory_str: 内存字符串，如 "1024Mi", "2Gi", "512M" 等

    Returns:
        int: 以字节为单位的内存大小
    """
    if isinstance(memory_str, (int, float)):
        return int(memory_str)

    if not isinstance(memory_str, str):
        raise ValueError(f"Invalid memory format: {memory_str}")

    # 移除空格并转换为大写
    memory_str = memory_str.strip()

    # 正则表达式匹配数字和单位
    match = re.match(r"^(\d+(?:\.\d+)?)\s*([KMGTPE]i?)?$", memory_str, re.IGNORECASE)
    if not match:
        raise ValueError(f"Invalid memory format: {memory_str}")

    value = float(match.group(1))
    unit = match.group(2)

    if not unit:
        # 没有单位，默认为字节
        return int(value)

    unit = unit.upper()

    # 1024进制单位 (Ki, Mi, Gi, Ti, Pi, Ei)
    if unit.endswith("I"):
        multipliers = {
            "KI": 1024,
            "MI": 1024**2,
            "GI": 1024**3,
            "TI": 1024**4,
            "PI": 1024**5,
            "EI": 1024**6,
        }
    else:
        # 1000进制单位 (K, M, G, T, P, E)
        multipliers = {
            "K": 1000,
            "M": 1000**2,
            "G": 1000**3,
            "T": 1000**4,
            "P": 1000**5,
            "E": 1000**6,
        }

    if unit not in multipliers:
        raise ValueError(f"Unknown memory unit: {unit}")

    return int(value * multipliers[unit])


def convert_dag_to_proto(dag):
    """Convert lucas DAG object to proto DAG message"""
    dag_metadata = dag.metadata()
    proto_nodes = []

    for node_data in dag_metadata:
        if node_data["type"] == "ControlNode":
            # Create ControlNode proto message
            control_node = controller_pb2.ControlNode()
            control_node.Id = node_data["id"]
            control_node.Done = node_data["done"]
            control_node.FunctionName = node_data["functionname"]
            # Handle params map
            for key, value in node_data["params"].items():
                control_node.Params[key] = str(value)
            control_node.Current = node_data["current"]
            control_node.DataNode = node_data["data_node"]
            control_node.PreDataNodes.extend(node_data["pre_data_nodes"])
            control_node.FunctionType = node_data["functiontype"]

            # Create DAGNode with ControlNode
            dag_node = controller_pb2.DAGNode()
            dag_node.Type = "ControlNode"
            dag_node.ControlNode.CopyFrom(control_node)
            proto_nodes.append(dag_node)

        elif node_data["type"] == "DataNode":
            # Create DataNode proto message
            data_node = controller_pb2.DataNode()
            data_node.Id = node_data["id"]
            data_node.Done = node_data["done"]
            data_node.Lambda = node_data["lambda"]
            data_node.Ready = node_data["ready"]
            data_node.SufControlNodes.extend(node_data["suf_control_nodes"])
            data_node.ChildNode.extend(node_data["child_node"])

            # Handle optional fields
            if node_data["pre_control_node"] is not None:
                data_node.PreControlNode = node_data["pre_control_node"]
            if node_data["parent_node"] is not None:
                data_node.ParentNode = node_data["parent_node"]

            # Create DAGNode with DataNode
            dag_node = controller_pb2.DAGNode()
            dag_node.Type = "DataNode"
            dag_node.DataNode.CopyFrom(data_node)
            proto_nodes.append(dag_node)

    # Create and return proto DAG
    proto_dag = controller_pb2.DAG()
    proto_dag.Nodes.extend(proto_nodes)
    return proto_dag


class ActorContext:
    @staticmethod
    def createContext(ignis_address: str = None, app_id: str = None):
        global actorContext
        if actorContext is None:
            if ignis_address is None:
                ignis_address = os.getenv("IGNIS_ADDR", "localhost:50051")
            if app_id is None:
                app_id = os.getenv("APP_ID", None)
            actorContext = ActorContext(ignis_address, app_id)
        return actorContext

    def __init__(self, ignis_address: str = None, app_id: str = None):
        if ignis_address is None:
            log.error("IGNIS_ADDR is not set")
            raise ValueError("IGNIS_ADDR is not set")
        if app_id is None:
            log.error("APP_ID is not set")
            raise ValueError("APP_ID is not set")
        self._ignis_address = ignis_address
        self._app_id = app_id
        self._channel = grpc.insecure_channel(
            ignis_address,
            options=[("grpc.max_receive_message_length", 512 * 1024 * 1024)],
        )
        self._stub = controller_pb2_grpc.ServiceStub(self._channel)
        self._q = queue.Queue()
        self._response_stream = self._stub.Session(self._generate())
        self._result_map: dict[str, platform_pb2.Flow] = {}
        self._thread = threading.Thread(target=self._run, daemon=True)

        # register application
        self.send(
            controller_pb2.Message(
                Type=controller_pb2.CommandType.FR_REGISTER_REQUEST,
                RegisterRequest=controller_pb2.RegisterRequest(
                    ApplicationID=app_id,
                ),
            )
        )

        # wait for ready
        for response in self._response_stream:
            response: controller_pb2.Message
            if response.Type == controller_pb2.CommandType.ACK:
                ack: controller_pb2.Ack = response.Ack
                if ack.Error != "":
                    log.error(f"Register application failed: {ack.Error}")
                    raise ValueError(f"Register application failed: {ack.Error}")
                break

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
        try:
            resources = dict(self._config.resources)
            resources["cpu"] = resources.get("cpu", 500)
            resources["memory"] = resources.get("memory", "128Mi")
            resources["gpu"] = resources.get("gpu", 0)
        except AttributeError:
            resources = {
                "cpu": 500,
                "memory": "128Mi",
                "gpu": 0,
            }
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
                Resources=controller_pb2.Resources(
                    CPU=resources["cpu"],
                    Memory=parse_memory_string(resources["memory"]),
                    GPU=resources["gpu"],
                ),
            ),
        )
        actorContext.send(message)

    def _transformfunction(self, fn):
        return fn


class ActorExecutor(Executor):
    def __init__(self, dag):
        super().__init__(dag)
        # send DAG to controller
        proto_dag = convert_dag_to_proto(dag)
        message = controller_pb2.Message(
            Type=controller_pb2.CommandType.FR_DAG,
            DAG=proto_dag,
        )
        actorContext.send(message)

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
                # TODO
                message = controller_pb2.Message(
                    Type=controller_pb2.CommandType.FR_MARK_DAG_NODE_DONE,
                    MarkDAGNodeDone=controller_pb2.MarkDAGNodeDone(
                        SessionId=session_id,
                        NodeId=node.metadata()["id"],
                    ),
                )
                print(1)
                actorContext.send(message)
            if _end:
                break
        result = None
        for node in self.dag.get_nodes():

            if isinstance(node, DataNode) and node._is_end_node:
                result = node._ld.value
                break

        self.dag.reset()
        return result
