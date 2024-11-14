# Actor Platform for Cloud-Edge Applications

A remote computing platform based on [Proto Actor](https://proto.actor/) for cloud-edge applications.

## Building

### Dependencies

Install `pkg-config`, `libczmq-dev`, this can be done on Ubuntu with:

```bash
sudo apt-get install pkg-config libczmq-dev
```

Then, install [Node.JS](https://nodejs.org/), which is needed for generating TypeScript bindings. We recommend using [NVM](https://github.com/nvm-sh/nvm/), since system package managers often have outdated versions. After installing NVM, run under [proto folder](proto/):

```bash
npm install
bash protobuf-gen.sh
```

This will generate Go, Python and TypeScript bindings for the protobuf messages used in the project.

### ActorC installation

In order to communicate with actor platform, you need to install `actorc` client package. Currently, we support `Python` package. To install it, go to `clients/py` and run:
```bash
pip install -e .
```

### Run an example application

An example application is provided in the `examples` folder. It is a simple object detection application that uses a pre-trained model to detect objects in images. You may refer to the [README](examples/README.md) in the example folder for more information.
