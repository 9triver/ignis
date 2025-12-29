import torch
from ultralytics import YOLO
import numpy as np
import cv2

device = torch.device("cuda" if torch.cuda.is_available() else "cpu")


# 加载训练好的头盔检测模型（YOLOv8n）
def load_helmet_model():
    model_path = "./eps_helmet/best.pt"

    print("正在加载 YOLO 模型...")
    model = YOLO(model_path)

    if torch.cuda.is_available():
        model.to(device)

    print("✓ 已加载训练好的头盔检测模型")
    print(f"  - 模型路径: {model_path}")
    print(f"  - 设备: {device}")

    return model


model = load_helmet_model()


def inference(im_data: bytes):
    nparr = np.frombuffer(im_data, np.uint8)
    im = cv2.imdecode(nparr, cv2.IMREAD_COLOR)

    result = model(im, conf=0.7, imgsz=1920)[0]
    boxes = []
    labels = []
    scores = []

    if result.boxes is not None:
        boxes_np = result.boxes.xyxy.cpu().numpy()
        scores_np = result.boxes.conf.cpu().numpy()
        labels_np = result.boxes.cls.cpu().numpy().astype(int)

        boxes = boxes_np.tolist()
        scores = scores_np.tolist()
        labels = labels_np.tolist()

    result_dict = {
        "boxes": boxes,
        "labels": labels,
        "scores": scores,
    }

    return result_dict
