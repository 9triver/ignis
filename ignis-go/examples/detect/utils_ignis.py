import base64
import glob
import random
import cv2
import numpy as np
import torch
import time
from torchvision.io import read_image
from torchvision.transforms import v2 as T
from torchvision.models.detection import (
    fasterrcnn_resnet50_fpn_v2,
    FasterRCNN_ResNet50_FPN_V2_Weights,
)
from torchvision.utils import draw_bounding_boxes


class Camera:
    def __init__(self, image_dir: str) -> None:
        self.images = glob.glob(f"{image_dir}/*.jpg")

    def capture(self, device):
        return random.choice(self.images)


class Model:
    def __init__(self):
        self.model = fasterrcnn_resnet50_fpn_v2(
            weights=FasterRCNN_ResNet50_FPN_V2_Weights.DEFAULT
        ).cuda()
        self.classes = {
            1: "person",
            2: "bicycle",
            3: "car",
            4: "motorcycle",
            5: "airplane",
            6: "bus",
            7: "train",
            8: "truck",
            9: "boat",
            10: "traffic light",
            11: "fire hydrant",
            13: "stop sign",
            14: "parking meter",
            15: "bench",
            16: "bird",
            17: "cat",
            18: "dog",
            19: "horse",
            20: "sheep",
            21: "cow",
            22: "elephant",
            23: "bear",
            24: "zebra",
            25: "giraffe",
            27: "backpack",
            28: "umbrella",
            31: "handbag",
            32: "tie",
            33: "suitcase",
            34: "frisbee",
            35: "skis",
            36: "snowboard",
            37: "sports ball",
            38: "kite",
            39: "baseball bat",
            40: "baseball glove",
            41: "skateboard",
            42: "surfboard",
            43: "tennis racket",
            44: "bottle",
            46: "wine glass",
            47: "cup",
            48: "fork",
            49: "knife",
            50: "spoon",
            51: "bowl",
            52: "banana",
            53: "apple",
            54: "sandwich",
            55: "orange",
            56: "broccoli",
            57: "carrot",
            58: "hot dog",
            59: "pizza",
            60: "donut",
            61: "cake",
            62: "chair",
            63: "couch",
            64: "potted plant",
            65: "bed",
            67: "dining table",
            70: "toilet",
            72: "tv",
            73: "laptop",
            74: "mouse",
            75: "remote",
            76: "keyboard",
            77: "cell phone",
            78: "microwave",
            79: "oven",
            80: "toaster",
            81: "sink",
            82: "refrigerator",
            84: "book",
            85: "clock",
            86: "vase",
            87: "scissors",
            88: "teddy bear",
            89: "hair drier",
            90: "toothbrush",
        }

    def get_transform(self):
        transforms = []
        transforms.append(T.ToDtype(torch.float, scale=True))
        transforms.append(T.ToPureTensor())
        return T.Compose(transforms)

    def inference(self, image):
        time.sleep(0.7)
        image = read_image(image)
        self.model.eval()
        with torch.no_grad():
            transform = self.get_transform()
            x = transform(image[:3, ...]).cuda()
            pred = self.model([x])[0]
        pred = {k: v.cpu().tolist() for k, v in pred.items()}
        pred["labels"] = [self.classes[label] for label in pred["labels"]]
        return pred


class Painter:
    def draw_boxes(self, image, pred):
        image = read_image(image)
        image = (255.0 * (image - image.min()) / (image.max() - image.min())).to(
            torch.uint8
        )
        image = image[:3, ...]
        pred_labels = [
            f"{label} ({score:.3f})"
            for label, score in zip(pred["labels"], pred["scores"])
            if score >= 0.6
        ]
        pred_boxes = torch.tensor(
            [box for box, score in zip(pred["boxes"], pred["scores"]) if score >= 0.6]
        )
        output_image = draw_bounding_boxes(
            image,
            pred_boxes,
            pred_labels,
            colors="red",
            width=3,
        )
        painted = output_image.permute(1, 2, 0).numpy()
        b64 = cv2.imencode(".jpg", painted)[1].tobytes().decode()
        return b64


camera = Camera("/home/rtq/workspace/demo/coco/val2017")
model = Model()
painter = Painter()


def save_result(data, path):
    for k, v in data["Results"].items():
        # print(v["Data"])
        b = base64.b64decode(v["Data"])
        b = base64.b64decode(b)
        im = cv2.imdecode(np.asarray(bytearray(b), dtype=np.uint8), cv2.IMREAD_COLOR)
        im = cv2.cvtColor(im, cv2.COLOR_BGR2RGB)
        cv2.imwrite(path, im)

