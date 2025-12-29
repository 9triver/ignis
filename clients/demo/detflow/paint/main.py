import numpy as np
import cv2


idx = 0


def paint(image: np.ndarray, result: dict):
    global idx

    confidence_threshold = 0.7  # 置信度阈值

    # 现在模型只输出2个类别：0=背景，1=头盔
    helmet_detections = []
    for box, label, score in zip(result["boxes"], result["labels"], result["scores"]):
        if score < confidence_threshold:
            continue

        # 只保留label=1的检测结果（头盔类别）
        if label == 1:  # 类别1是头盔
            helmet_detections.append((box, label, score))

    # 绘制头盔检测结果
    for box, label, score in helmet_detections:
        x1, y1, x2, y2 = box

        # 使用红色框绘制头盔检测结果
        cv2.rectangle(image, (int(x1), int(y1)), (int(x2), int(y2)), (0, 0, 255), 2)

        # 使用hard_hat作为类别名称
        class_name = "hard_hat"

        # 绘制标签和置信度
        label_text = f"{class_name}: {score:.2f}"
        cv2.putText(
            image,
            label_text,
            (int(x1), int(y1) - 10),
            cv2.FONT_HERSHEY_SIMPLEX,
            0.6,
            (0, 0, 255),
            2,
        )

    print(f"检测到头盔数量: {len(helmet_detections)}")
    cv2.imwrite(f"/app/helmet_detection_result-{idx}.jpg", image)
    idx += 1
