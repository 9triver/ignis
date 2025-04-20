from typing import NamedTuple

import httpx

from ..protos.dag import node_pb2 as dag


class PlatformEndPoint(NamedTuple):
    addr: str = "localhost"
    port: int = 2313

    @property
    def _base_addr(self):
        return f"http://{self.addr}:{self.port}"

    @property
    def dag(self):
        return self._base_addr + "/dag"


DEFAULT_EP = PlatformEndPoint()
DEFAULT_ASYNC_CLIENT = httpx.AsyncClient(timeout=httpx.Timeout(None))


async def send_dag_cmd(cmd: dag.Command, ep: PlatformEndPoint = DEFAULT_EP) -> dict:
    data = cmd.SerializeToString()

    resp = await DEFAULT_ASYNC_CLIENT.post(ep.dag, content=data)
    resp.raise_for_status()

    return resp.json()
