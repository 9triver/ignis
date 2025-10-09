import os
import sys
import time
from typing import Iterable

import cv2
import numpy as np
import torch
from lucas import workflow, function, Workflow
from lucas.serverless_function import Metadata
from actorc.controller.context import (
    ActorContext,
    ActorFunction,
    ActorExecutor,
    ActorRuntime,
)
import uuid

from torchvision.models.detection import (
    fasterrcnn_resnet50_fpn,
    FasterRCNN_ResNet50_FPN_Weights,
)
from torchvision.transforms import functional as F

COCO_INSTANCE_CATEGORY_NAMES = [
    "__background__",
    "person",
    "bicycle",
    "car",
    "motorcycle",
    "airplane",
    "bus",
    "train",
    "truck",
    "boat",
    "traffic light",
    "fire hydrant",
    "N/A",
    "stop sign",
    "parking meter",
    "bench",
    "bird",
    "cat",
    "dog",
    "horse",
    "sheep",
    "cow",
    "elephant",
    "bear",
    "zebra",
    "giraffe",
    "N/A",
    "backpack",
    "umbrella",
    "N/A",
    "N/A",
    "handbag",
    "tie",
    "suitcase",
    "frisbee",
    "skis",
    "snowboard",
    "sports ball",
    "kite",
    "baseball bat",
    "baseball glove",
    "skateboard",
    "surfboard",
    "tennis racket",
    "bottle",
    "N/A",
    "wine glass",
    "cup",
    "fork",
    "knife",
    "spoon",
    "bowl",
    "banana",
    "apple",
    "sandwich",
    "orange",
    "broccoli",
    "carrot",
    "hot dog",
    "pizza",
    "donut",
    "cake",
    "chair",
    "couch",
    "potted plant",
    "bed",
    "N/A",
    "dining table",
    "N/A",
    "N/A",
    "toilet",
    "N/A",
    "tv",
    "laptop",
    "mouse",
    "remote",
    "keyboard",
    "cell phone",
    "microwave",
    "oven",
    "toaster",
    "sink",
    "refrigerator",
    "N/A",
    "book",
    "clock",
    "vase",
    "scissors",
    "teddy bear",
    "hair drier",
    "toothbrush",
]

model = fasterrcnn_resnet50_fpn(weights=FasterRCNN_ResNet50_FPN_Weights.DEFAULT)
model.eval()

context = ActorContext.createContext("localhost:8082")


@function(
    wrapper=ActorFunction,
    dependency=["opencv-python-headless"],
    provider="actor",
    name="read_image",
    venv="test2",
    replicas=4,
)
def read_image(path: str):
    im = cv2.imread(path)
    return im


@function(
    wrapper=ActorFunction,
    dependency=[],
    provider="actor",
    name="inference",
    venv="test2",
    replicas=3,
)
def inference(im: np.ndarray):
    image_tensor = F.to_tensor(im).unsqueeze(0)  # 转换为张量并添加批次维度

    with torch.no_grad():
        predictions = model(image_tensor)

    return predictions[0]


@function(
    wrapper=ActorFunction,
    dependency=[],
    provider="actor",
    name="paint",
    venv="test2",
)
def paint(image: np.ndarray, result: dict):
    for box, label, score in zip(result["boxes"], result["labels"], result["scores"]):
        if score < 0.8:
            continue

        x1, y1, x2, y2 = box
        cv2.rectangle(image, (int(x1), int(y1)), (int(x2), int(y2)), (0, 255, 0), 2)
        cv2.putText(
            image,
            f"{COCO_INSTANCE_CATEGORY_NAMES[label]}: {score:.2f}",
            (int(x1), int(y1) - 10),
            cv2.FONT_HERSHEY_SIMPLEX,
            0.5,
            (0, 255, 0),
            2,
        )
    cv2.imwrite(f"/home/spark4862/Documents/projects/go/ignis/result.jpg", image)
    return image


@workflow(executor=ActorExecutor)
def workflowfunc(wf: Workflow):
    _in = wf.input()

    im = wf.call("read_image", {"path": _in["path"]})
    pred = wf.call("inference", {"im": im})
    vis = wf.call("paint", {"image": im, "result": pred})
    return vis


workflow_i = workflowfunc.generate()
# dag = workflow_i.valicate()
# import json

# print(json.dumps(dag.metadata(fn_export=True), indent=2))


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


detect = workflowfunc.export(actorWorkflowExportFunc)
# print("----first execute----")
img_dir = "/home/spark4862/Documents/projects/go/ignis/clients/demo/images"
for file in os.listdir(img_dir):
    if file.endswith(".jpg"):
        detect({"path": os.path.join(img_dir, file)})
