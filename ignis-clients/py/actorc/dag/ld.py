from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from .dag import DataNode


class Lambda:
    def __init__(self, value: Any | None = None) -> None:
        self.value = value

    def getDataNode(self):
        return self._dataNode

    def dataNode(self, node: "DataNode"):
        self._dataNode = node
        return

    def __getattr__(self, name: str):
        attr = getattr(self.value, name)

        if callable(attr):

            def wrapper(*args, **kwargs):
                return attr(*args, **kwargs)

            return wrapper
        else:
            return attr

    def __add__(self, value):
        if not isinstance(value, Lambda):
            value = Lambda(value)

        return lambda x: lambda y: x + y

        # ctl_node = ControlNode(lambda x: lambda y: x + y)
        # ctl_node.add_pre_data_node(self._dataNode)
        # ctl_node.add_pre_data_node(value_node)
        # self._dataNode.add_succ_control_node(ctl_node)
        # value_node.add_succ_control_node(ctl_node)
        # return Lambda(self.value + value)

    def __sub__(self, value):
        return self.value - value

    def __mul__(self, value):
        return self.value * value

    def __truediv__(self, value):
        return self.value / value

    def __iadd__(self, value):
        self.value += value
        return self

    def __isub__(self, value):
        self.value -= value
        return self

    def __imul__(self, value):
        self.value *= value
        return self

    def __itruediv__(self, value):
        self.value /= value
        return self

    def __str__(self) -> str:
        return f"{super().__str__()}:{self.value}"
