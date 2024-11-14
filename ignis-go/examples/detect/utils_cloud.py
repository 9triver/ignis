import asyncio
from concurrent import futures
import glob
import pickle
import queue
import random
import threading
import time
import torch
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
        im = read_image(random.choice(self.images))
        return pickle.dumps(im)


class Model:
    def __init__(self):
        self.q = queue.Queue()
        self.device = "cuda" if torch.cuda.is_available() else "cpu"
        self.model = fasterrcnn_resnet50_fpn_v2(
            weights=FasterRCNN_ResNet50_FPN_V2_Weights.DEFAULT
        ).to(self.device)
        self._runner = threading.Thread(target=self.runner)
        self._runner.start()
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

    def runner(self):
        while True:
            image, fut = self.q.get()
            # print("Pull, pending:", self.q.qsize())
            pred = self._inference(image)
            fut.set_result(pred)

    def get_transform(self):
        transforms = []
        transforms.append(T.ToDtype(torch.float, scale=True))
        transforms.append(T.ToPureTensor())
        return T.Compose(transforms)

    def inference(self, image):
        # print("Push, pending:", self.q.qsize())
        fut = futures.Future()
        self.q.put((image, fut))
        return fut

    def _inference(self, image):
        time.sleep(0.7)
        image = pickle.loads(image)
        self.model.eval()
        with torch.no_grad():
            transform = self.get_transform()
            x = transform(image[:3, ...]).to(self.device)
            pred = self.model([x])[0]
        pred = {k: v.cpu().tolist() for k, v in pred.items()}
        pred["labels"] = [self.classes[label] for label in pred["labels"]]
        return pred


class Painter:
    def _draw_boxes(self, image, pred):
        image = pickle.loads(image)
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
            # font_size=24,
            # font="DejaVuSans",
        )
        painted = output_image.permute(1, 2, 0).numpy()
        return painted

    def draw_boxes(self, image, pred: futures.Future):
        fut = futures.Future()

        def callback(f: futures.Future):
            painted = self._draw_boxes(image, f.result())
            fut.set_result(painted)

        pred.add_done_callback(callback)
        return fut


camera = Camera("coco/val2017")
model = Model()
painter = Painter()


__all__ = ["camera", "model", "painter"]
