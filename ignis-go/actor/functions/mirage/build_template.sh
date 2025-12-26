#!/bin/bash

TARGET=${1:-unix}

# clear cache
make clean

mirage configure -t $TARGET

# clone websocket repo
git clone https://github.com/vbmithr/ocaml-websocket.git websocket

# workaround: incorrect dep constraints
sed -i -E 's/(&.*)$/}/' ./mirage/ws-handler-$TARGET.opam

make depends

# workaround: export private package
echo "module Input_channel = Input_channel" >> ./duniverse/ocaml-cohttp/cohttp-mirage/src/cohttp_mirage.ml

# workaround: use local websocket package
sed -i 's/yojson/yojson websocket/g' dune.build

# build target
make build
