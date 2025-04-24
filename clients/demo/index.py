import sys

from lucas import workflow, function, Runtime, Workflow, Function
from lucas.serverless_function import Metadata
from lucas.workflow.executor import Executor
from lucas.workflow.dag import DAGNode, DataNode, ControlNode
from lucas.utils.logging import log

from actorc.protos.controller import controller_pb2, controller_pb2_grpc
from actorc.protos import platform_pb2

import cloudpickle
import grpc
import uuid
import queue
import time

channel = grpc.insecure_channel("localhost:8082")
stub = controller_pb2_grpc.ServiceStub(channel)
q = queue.Queue()


def generate():
    while True:
        msg = q.get()
        if msg is None:
            break
        print(msg)
        yield msg


response_stream = stub.Session(generate())

import threading

result_map: dict[str, platform_pb2.Flow] = {}


def run():
    while True:
        for response in response_stream:
            print("resp", response)
            response: controller_pb2.Message
            if response.Type == controller_pb2.CommandType.BK_RETURN_RESULT:
                result: controller_pb2.ReturnResult = response.ReturnResult
                sessionID = result.SessionID
                instanceID = result.InstanceID
                name = result.Name
                value = result.Value
                key = f"{sessionID}-{instanceID}-{name}"
                result_map[key] = value.Ref
        time.sleep(1)


threading.Thread(target=run, daemon=True).start()


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
            result = result_map.get(key)
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
        params = self._config.params
        venv = self._config.venv
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
            ),
        )
        q.put(message)

    def _transformfunction(self, fn):
        def actor_function(data: dict):
            # just use for local invoke
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
            result = fn(rt)
            return result

        return actor_function


class ActorExecutor(Executor):
    def __init__(self, dag):
        super().__init__(dag)
        self._sesstionID = str(uuid.uuid4())

    def execute(self):
        while not self.dag.hasDone():
            task: list[DAGNode] = []
            for node in self.dag.get_nodes():
                if node._done:
                    continue
                if isinstance(node, DataNode):
                    if node.is_ready():
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
                            if isinstance(data, platform_pb2.Flow):
                                rpc_data = controller_pb2.Data(
                                    Type=controller_pb2.Data.ObjectType.OBJ_REF,
                                    Ref=data,
                                )
                            else:
                                rpc_data = controller_pb2.Data(
                                    Type=controller_pb2.Data.ObjectType.OBJ_ENCODED,
                                    Encoded=platform_pb2.EncodedObject(
                                        ID="obj." + str(uuid.uuid4()),
                                        Data=cloudpickle.dumps(data),
                                        Language=platform_pb2.Language.LANG_PYTHON,
                                    ),
                                )

                            appendArg = controller_pb2.AppendArg(
                                SessionID=self._sesstionID,
                                InstanceID=control_node_metadata["id"],
                                Name=control_node_metadata["functionname"],
                                Param=params[node._ld.getid()],
                                Value=rpc_data,
                            )
                            message = controller_pb2.Message(
                                Type=controller_pb2.CommandType.FR_APPEND_ARG,
                                AppendArg=appendArg,
                            )
                            q.put(message)

                        log.info(f"{control_node.describe()} appargs {node._ld.value}")
                        if control_node.appargs(node._ld):
                            if control_node._fn_type == "remote":
                                control_node._datas["sessionID"] = self._sesstionID
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


@function(
    wrapper=ActorFunction,
    dependency=[],
    provider="actor",
    name="read_data",
    params=["dataset", "name"],
    venv="test2",
)
def read_data(rt: Runtime):
    inputs = rt.input()
    import os

    os.environ["HF_ENDPOINT"] = "https://hf-mirror.com"
    from datasets import load_dataset

    dataset = load_dataset(inputs["dataset"], inputs["name"])["train"]

    def generator():
        for i, x in enumerate(dataset):
            print(x)
            yield x
            if i > 100:
                return

    return rt.output(generator())


@function(
    wrapper=ActorFunction,
    dependency=[],
    provider="actor",
    name="process_data",
    params=["loader"],
    venv="test2",
)
def process_data(rt: Runtime):
    inputs = rt.input()

    def generator():
        for x in inputs["loader"]:
            if x["text"] == "":
                continue
            x["text"] = x["text"].strip()
            yield x

    return rt.output(generator())


@function(
    wrapper=ActorFunction,
    dependency=[],
    provider="actor",
    name="train",
    params=["loader"],
    venv="test2",
)
def process_data(rt: Runtime):
    inputs = rt.input()
    param = 0
    for x in inputs["loader"]:
        print(x, file=sys.stderr)
        param += 1
    print(param, file=sys.stderr)
    return rt.output(param)


@workflow(executor=ActorExecutor)
def workflowfunc(wf: Workflow):
    _in = wf.input()
    loader = wf.call("read_data", {"dataset": _in["dataset"], "name": _in["name"]})
    processed = wf.call("process_data", {"loader": loader})
    trained = wf.call("train", {"loader": processed})
    return trained


def actorWorkflowExportFunc(dict: dict):
    # just use for local invoke
    from lucas import routeBuilder

    route = routeBuilder.build()
    route_dict = {}
    for function in route.functions:
        route_dict[function.name] = function.handler
    for workflow in route.workflows:
        route_dict[workflow.name] = function.handler
    metadata = Metadata(
        id=str(uuid.uuid4()),
        params=dict,
        namespace=None,
        router=route_dict,
        request_type="invoke",
        redis_db=None,
        producer=None,
    )
    rt = ActorRuntime(metadata)
    workflowfunc.set_runtime(rt)
    workflow = workflowfunc.generate()
    return workflow.execute()


workflow_func = workflowfunc.export(actorWorkflowExportFunc)
workflow_func({"dataset": "wikitext", "name": "wikitext-2-raw-v1"})

q.put(None)
