import sys
import time
from typing import Iterable
from lucas import workflow, function, Workflow
from lucas.serverless_function import Metadata
from actorc.controller.context import (
    ActorContext,
    ActorFunction,
    ActorExecutor,
    ActorRuntime,
)
import uuid

context = ActorContext.createContext("localhost:8082")


@function(
    wrapper=ActorFunction,
    dependency=[],
    provider="actor",
    name="read_data",
    venv="test2",
)
def read_data(dataset: str, name: str):
    import os

    os.environ["HF_ENDPOINT"] = "https://hf-mirror.com"
    from datasets import load_dataset

    dataset = load_dataset(dataset, name)["train"]

    def generator():
        for i, x in enumerate(dataset):
            print(x)
            time.sleep(0.5)
            yield x
            if i > 100:
                return

    return generator()


@function(
    wrapper=ActorFunction,
    dependency=[],
    provider="actor",
    name="process_data",
    venv="test2",
)
def process_data(loader: Iterable[dict[str, str]]):
    def generator():
        for x in loader:
            if x["text"] == "":
                continue
            x["text"] = x["text"].strip()
            print("processed:", x["text"], file=sys.stderr)
            yield x

    return generator()


@function(
    wrapper=ActorFunction,
    dependency=[],
    provider="actor",
    name="train",
    venv="test2",
)
def train(loader: Iterable[dict[str, str]]):
    import sys

    param = 0
    for x in loader:
        print(x, file=sys.stderr)
        param += 1
    print(param, file=sys.stderr)

    return param


@workflow(executor=ActorExecutor)
def workflowfunc(wf: Workflow):
    _in = wf.input()

    loader = wf.call("read_data", {"dataset": _in["dataset"], "name": _in["name"]})
    processed = wf.call("process_data", {"loader": loader})
    trained = wf.call("train", {"loader": processed})
    return trained


workflow_i = workflowfunc.generate()
dag = workflow_i.valicate()
import json

print(json.dumps(dag.metadata(fn_export=True), indent=2))


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
print("----first execute----")
workflow_func({"dataset": "wikitext", "name": "wikitext-2-raw-v1"})
