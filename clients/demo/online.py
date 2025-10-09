from lucas import workflow, function, Workflow
from lucas.serverless_function import Metadata

from actorc.controller.context import (
    ActorContext,
    ActorFunction,
    ActorExecutor,
    ActorRuntime,
)
import uuid
import sys

from river import datasets
from river import linear_model

from river import metrics

from river import compose
from river import preprocessing
from river import evaluate
import time

context = ActorContext.createContext("localhost:8082")


# todo: 模型对比


@function(
    wrapper=ActorFunction,
    dependency=[],
    provider="actor",
    name="get_data",
    venv="test2",
)
def get_data(_in):
    dataset = datasets.Phishing()

    def tmp():
        for x, y in dataset:
            yield {
                "x": x,
                "y": y,
            }

    return tmp()


@function(
    wrapper=ActorFunction,
    dependency=[],
    provider="actor",
    name="evaluate_data",
    venv="test2",
)
def evaluate_data(dataset):
    time.sleep(2)
    model = compose.Pipeline(
        preprocessing.StandardScaler(), linear_model.LogisticRegression()
    )
    metric = metrics.ROCAUC()

    for data in dataset:
        x = data["x"]
        y = data["y"]
        print(x, y, file=sys.stderr)
        y_pred = model.predict_proba_one(x)
        model.learn_one(x, y)
        metric.update(y, y_pred)

    # evaluate.progressive_val_score(dataset, model, metric)
    return metric


@function(
    wrapper=ActorFunction,
    dependency=[],
    provider="actor",
    name="store_data",
    venv="test2",
)
def store_data(metrics, storage_path):
    time.sleep(2)
    print(metrics, file=sys.stderr)
    with open(storage_path, "w") as f:
        f.write(str(metrics))

    return metrics


# print(metric)


@workflow(executor=ActorExecutor)
def workflowfunc(wf: Workflow):
    _in = wf.input()
    dataset = wf.call("get_data", {"_in": 0})
    metrics = wf.call("evaluate_data", {"dataset": dataset})
    wf.call("store_data", {"metrics": metrics, "storage_path": _in["storage_path"]})
    return metrics


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
workflow_func(
    {"storage_path": "/home/spark4862/Documents/projects/go/ignis/result.log"}
)
