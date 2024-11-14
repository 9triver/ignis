import asyncio

from actorc.dag import IR, function, workflow
from actorc.utils import PlatformEndPoint

ep = PlatformEndPoint("localhost", 2313)


@function()
def capture(Device): ...


@function()
def inference(Image): ...


@function()
def draw(Image, Result): ...


@workflow()
def service(ir: IR):
    im = ir.func(capture, {"Device": "/dev/video0"})
    res = ir.func(inference, {"Image": im})
    vis = ir.func(draw, {"Image": im, "Result": res})
    return vis


async def main():
    srv = service()
    print(srv)
    await srv.deploy("test", ep)
    futures: list[asyncio.Task[dict]] = []
    for _ in range(10):
        futures.append(asyncio.create_task(srv.execute_remote("test", ep)))
        await asyncio.sleep(1)

    results = await asyncio.gather(*futures)
    print(results)


if __name__ == "__main__":
    asyncio.run(main())
