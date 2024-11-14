from functools import partial
import threading
import time
from utils_cloud import camera, model, painter
import cv2


def capture(device):
    return camera.capture(device)


def detect(im):
    return model.inference(im)


def paint(im, result):
    return painter.draw_boxes(im, result)


def driver():
    im = capture(0)
    pred = detect(im)
    painted = paint(im, pred)
    return painted


tics = {}


def start_camera(name, latency):
    print(f"Started camera {name} with interval {latency}ms")
    queue = 0

    def save(task_name: str, painted):
        nonlocal queue
        queue -= 1
        print(
            f"Done, {task_name}: {time.time() - tics[task]:.2f}s. {name} pending: {queue}"
        )
        painted = cv2.cvtColor(painted, cv2.COLOR_BGR2RGB)
        cv2.imwrite(f"results-cloud/painted_{task_name}.png", painted)

    for i in range(10):
        task = f"{name}_{i}"
        tics[task] = time.time()
        painted = driver()

        def on_done(f, task):
            save(task, f.result())

        painted.add_done_callback(partial(on_done, task=task))
        queue += 1
        time.sleep(latency / 1000)
    print(f"Thread {name} finised")


if __name__ == "__main__":
    t1 = threading.Thread(target=start_camera, args=("cam-1", 300))
    t2 = threading.Thread(target=start_camera, args=("cam-2", 400))
    t3 = threading.Thread(target=start_camera, args=("cam-3", 700))
    threads = [t1, t2, t3]
    for t in threads:
        t.start()
    for t in threads:
        t.join()
