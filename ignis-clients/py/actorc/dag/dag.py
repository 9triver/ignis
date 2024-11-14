from typing import Any, List

from .ld import Lambda


class DAGNode:
    def __init__(self) -> None:
        pass


class ControlNode(DAGNode):
    def __init__(self, fn) -> None:
        self.fn = fn
        self.pre_data_nodes = []
        self.ld_to_key: dict[Lambda, str] = {}
        self.datas = {}

    def add_pre_data_node(self, data_node: DAGNode):
        self.pre_data_nodes.append(data_node)

    def set_data_node(self, data_node):
        self.data_node = data_node

    def get_pre_data_nodes(self):
        return self.pre_data_nodes

    def get_data_node(self):
        return self.data_node

    def defParams(self, ld: Lambda, key: str):
        self.ld_to_key[ld] = key

    def appargs(self, ld: Lambda) -> bool:
        key = self.ld_to_key[ld]
        self.datas[key] = ld.value
        if len(self.datas) == len(self.ld_to_key):
            return True
        else:
            return False

    def calculate(self):
        res = self.fn(self.datas)
        self.data_node.set_value(res)
        return self.get_data_node()

    def __str__(self) -> str:
        res = f"(ControlNode {super().__str__()}) {self.fn.__qualname__}"
        return res


class DataNode(DAGNode):
    def __init__(self, ld: Lambda) -> None:
        self.ld = ld
        self.ready = ld.value != None
        self.succ_control_nodes = []
        self.is_end_node = False
        ld.dataNode(self)

    def add_succ_control_node(self, control_node: ControlNode):
        self.succ_control_nodes.append(control_node)

    def get_succ_control_nodes(self):
        return self.succ_control_nodes

    def set_value(self, value: Any):
        self.ld.value = value
        self.ready = True

    def __str__(self) -> str:
        res = f"[DataNode {super()}] {self.ld}"
        return res


class DAG:
    def __init__(self) -> None:
        self.nodes: List[DAGNode] = []

    def add_node(self, node: DAGNode):
        self.nodes.append(node)

    def __str__(self):
        res = ""
        for node in self.nodes:
            if isinstance(node, DataNode):
                res += str(node)
                for control_node in node.get_succ_control_nodes():
                    control_node: ControlNode
                    res += f"  -> {str(control_node)}\n"
            if isinstance(node, ControlNode):
                res += str(node)
                data_node: DataNode = node.get_data_node()
                res += f"  -> {str(data_node)}\n"
        return res

    def run(self):
        task = []
        for node in self.nodes:
            if isinstance(node, DataNode):
                if node.ready:
                    task.append(node)
            if isinstance(node, ControlNode):
                if node.get_pre_data_nodes() == []:
                    task.append(node)

        while len(task) != 0:
            node = task.pop(0)
            if isinstance(node, DataNode):
                for control_node in node.get_succ_control_nodes():
                    control_node: ControlNode
                    if control_node.appargs(node.ld):
                        task.append(control_node)
            if isinstance(node, ControlNode):
                r_node: DataNode = node.calculate()
                task.append(r_node)
