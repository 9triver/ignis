import os
import sys

# trick: add the protos folder to the python package search path
# because generated protobuf files use relative imports to import other generated files
PACKAGE_PATH = os.path.dirname(__file__)
PROTO_PATH = os.path.join(PACKAGE_PATH, "protos")

sys.path.append(PROTO_PATH)


__all__ = ["PROTO_PATH"]
