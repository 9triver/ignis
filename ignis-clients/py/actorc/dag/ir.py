from typing import Any

from ..protos.dag import node_pb2 as pb
from ..utils import DEFAULT_EP, EncDec, PlatformEndPoint, send_dag_cmd
from .dag import DAG, ControlNode, DataNode
from .ld import Lambda


class IR:
    def __init__(self, route=None) -> None:
        self.route = route
        self.dag = DAG()

    def func(self, fn, fnParams: dict[str, Any]) -> Lambda:
        fn_ctl_node = ControlNode(fn=fn)
        self.dag.add_node(fn_ctl_node)
        for key, ld in fnParams.items():
            if not isinstance(ld, Lambda):
                ld = Lambda(ld)
                data_node = DataNode(ld)
            else:
                data_node = ld.getDataNode()
            data_node.add_succ_control_node(fn_ctl_node)
            fn_ctl_node.add_pre_data_node(ld._dataNode)
            self.dag.add_node(data_node)

            fn_ctl_node.defParams(ld, key)

        r = Lambda()
        data_node = DataNode(r)
        self.dag.add_node(data_node)
        fn_ctl_node.set_data_node(data_node)
        return r

    def execute(self):
        return self.dag.run()

    def end_with(self, ld: Lambda):
        data_node: DataNode = ld.getDataNode()
        data_node.is_end_node = True

    async def execute_remote(self, name: str, ep: PlatformEndPoint = DEFAULT_EP):
        cmd = pb.Command(Type=pb.COMMAND_Execute, DAG=name, Execute=pb.Execute())
        return await send_dag_cmd(cmd, ep)

    async def deploy(self, name: str, ep: PlatformEndPoint = DEFAULT_EP):
        dag = self.dag
        await send_dag_cmd(
            pb.Command(Type=pb.COMMAND_Create, DAG=name, Create=pb.Create()),
            ep,
        )

        for node in dag.nodes:
            if isinstance(node, ControlNode):
                await send_dag_cmd(
                    pb.Command(
                        Type=pb.COMMAND_AppendNode,
                        DAG=name,
                        AppendNode=pb.AppendNode(
                            Name=str(node),
                            Params=node.ld_to_key.values(),
                            Group=str(node.fn.__qualname__),
                        ),
                    ),
                    ep,
                )
            if isinstance(node, DataNode):
                if node.ld.value == None:
                    continue
                await send_dag_cmd(
                    pb.Command(
                        Type=pb.COMMAND_AppendNode,
                        DAG=name,
                        AppendNode=pb.AppendNode(
                            Name=str(node), Value=EncDec.encode(node.ld.value)
                        ),
                    ),
                    ep,
                )
                for control_node in node.get_succ_control_nodes():
                    control_node: ControlNode
                    await send_dag_cmd(
                        pb.Command(
                            Type=pb.COMMAND_AppendEdge,
                            DAG=name,
                            AppendEdge=pb.AppendEdge(
                                From=str(node),
                                To=str(control_node),
                                Param=control_node.ld_to_key[node.ld],
                            ),
                        ),
                        ep,
                    )

        for node in dag.nodes:
            if isinstance(node, ControlNode):
                data_node: DataNode = node.get_data_node()
                for dst in data_node.get_succ_control_nodes():
                    dst: ControlNode
                    await send_dag_cmd(
                        pb.Command(
                            Type=pb.COMMAND_AppendEdge,
                            DAG=name,
                            AppendEdge=pb.AppendEdge(
                                From=str(node),
                                To=str(dst),
                                Param=dst.ld_to_key[data_node.ld],
                            ),
                        ),
                        ep,
                    )
                if data_node.is_end_node:
                    await send_dag_cmd(
                        pb.Command(
                            Type=pb.COMMAND_AppendOutput,
                            DAG=name,
                            AppendOutput=pb.AppendOutput(Node=str(node)),
                        ),
                        ep,
                    )
        await send_dag_cmd(
            pb.Command(Type=pb.COMMAND_Serve, DAG=name, Serve=pb.Serve()),
            ep,
        )

    def __str__(self) -> str:
        return str(self.dag)
