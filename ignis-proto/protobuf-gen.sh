#!/bin/bash

PROTOC="python -m grpc_tools.protoc"
export PATH="$PATH:$HOME/go/bin"

ACTOR_SRC=$(go list -f {{.Dir}} github.com/asynkron/protoactor-go/actor)
ACTOR_PROTO=$ACTOR_SRC/actor.proto

PROTOC="$PROTOC -I $ACTOR_SRC -I ./messages"
PROTO_SRC="./messages/*.proto ./messages/executor/*.proto ./messages/controller/*.proto"

if [ ! -d ./python ]; then
  echo "Creating output directory for Python: ./python"
  mkdir -p ./python
fi

echo "Generating protobuf files for Go"
$PROTOC --go_out=./go --go_opt=paths=source_relative --go-grpc_out=./go --go-grpc_opt=paths=source_relative $PROTO_SRC

PY_OUTPUTS="./python ../clients/py/actorc/protos"

echo "Generating protobuf files for Python"
for PY_OUTPUT in $PY_OUTPUTS; do
  if [ ! -d $PY_OUTPUT ]; then
    echo "Creating output directory for Python: $PY_OUTPUT"
    mkdir -p $PY_OUTPUT
  fi
  $PROTOC --python_out=$PY_OUTPUT --pyi_out=$PY_OUTPUT --grpc_python_out=$PY_OUTPUT $PROTO_SRC $ACTOR_PROTO
done
