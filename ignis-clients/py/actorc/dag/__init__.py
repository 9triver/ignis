import functools
from typing import Any, Callable, List, Tuple

from .ir import IR


def instance(name: str | None = None):
    # TODO: add metadata to wrapped instance
    def _wrapper(class_: type):
        return class_

    return _wrapper


def function(name: str | None = None):
    # TODO: add metadata to wrapped function
    def _wrapper(fn: Callable):
        return fn

    return _wrapper


def workflow(bind: List[Tuple[Any, Any]] = []):
    route = {}
    for b in bind:
        key, value = b
        route[key] = value
        route[value] = key

    def _workflow(fn: Callable[[IR], Any]):
        @functools.wraps(fn)
        def handler():
            ir = IR(route)
            r = fn(ir)
            ir.end_with(r)
            return ir

        return handler

    return _workflow


__all__ = ["instance", "function", "workflow"]
