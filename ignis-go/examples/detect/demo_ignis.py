import asyncio
from functools import partial
import time
from utils_ignis import camera, model, painter, save_result
from actorc.utils import PlatformEndPoint
from actorc.dag import IR, function, workflow
import asyncio

ep = PlatformEndPoint("localhost", 2313)


@function()
def capture(device):
    return camera.capture(device)


@function()
def detect(im):
    return model.inference(im)


@function()
def paint(im, pred):
    return painter.draw_boxes(im, pred)


@workflow()
def driver(ir: IR):
    im = ir.func(capture, {"device": 0})
    pred = ir.func(detect, {"im": im})
    painted = ir.func(paint, {"im": im, "pred": pred})
    return painted


app = driver()
tics = {}


async def start_camera(name, latency):
    print(f"Started camera {name} with interval {latency}ms")
    queue = 0
    tasks = []

    def save(task: str, painted):
        nonlocal queue
        save_result(painted["result"], f"results-ignis/painted_{task}.jpg")
        queue -= 1
        print(f"Done, {task}: {time.time() - tics[task]:.2f}s. {name} pending: {queue}")

    for i in range(10):
        task = f"{name}_{i}"
        print(task)
        tics[task] = time.time()
        painted = asyncio.create_task(app.execute_remote("detect", ep))

        def on_done(f, task):
            save(task, f.result())

        painted.add_done_callback(partial(on_done, task=task))
        queue += 1
        await asyncio.sleep(latency / 1000)
        tasks.append(painted)
    print(f"Thread {name} finised")
    await asyncio.gather(*tasks)


async def main():
    await app.deploy("detect", ep)
    threads = [
        asyncio.create_task(start_camera("cam-1", 300)),
        asyncio.create_task(start_camera("cam-2", 400)),
        asyncio.create_task(start_camera("cam-3", 700)),
    ]
    await asyncio.gather(*threads)


if __name__ == "__main__":
    asyncio.run(main())
