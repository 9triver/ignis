# Actor Platform for Cloud-Edge Applications

A remote computing platform based on [Proto Actor](https://proto.actor/) for cloud-edge applications.

## Building

### Dependencies

Install `pkg-config`, `libczmq-dev`, this can be done on Ubuntu with:

```bash
sudo apt-get install pkg-config libczmq-dev
```

Then, create a venv and install [gRPC](https://grpc.org.cn/docs/languages/python/quickstart/), which is needed for generating protocol buffer code for Python and Go:
```bash
python -m venv venv
source venv/bin/activate

pip install grpcio
pip install grpcio-tools

go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest 
```

After installing gRPC, run under [proto folder](ignis-proto/):

```bash
bash protobuf-gen.sh
```

This will generate Go and Python bindings for the protobuf messages and gRPC services used in the project.

### Actor client installation

In order to communicate with actor platform, you need to install `actorc` client package. To install it, go to `clients/py` and run:
```bash
pip install -e .
```

The client also depends on [Lucas](https://github.com/Xdydy/Lucas/), which is not included by pypi. Therefore, you need to install it manually:
```bash
git clone https://github.com/Xdydy/Lucas.git
cd Lucas && pip install -e .
```

## Running

An example is under development...
