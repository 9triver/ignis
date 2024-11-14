# Actor-based Object Detection Application

## Dependencies

Besides the dependencies mentioned in the [main README](../README.md), this example requires [libtensorflow](https://www.tensorflow.org/install/lang_c) and [OpenCV](https://opencv.org/) to be installed. They are vital for installing [Tensorflow Go](https://pkg.go.dev/github.com/wamuir/graft@v0.8.1/tensorflow) and [GoCV](https://pkg.go.dev/gocv.io/x/gocv).

### Installing Webcam

Default camera in this example is `/dev/video0`. If you have a different camera, you can change the camera path in the `main.go` file. Also, you need to install webcam drivers before running the application.

### Installing libtensorflow

According to [official installation instructions](https://github.com/tensorflow/build/tree/master/golang_install_guide), you need to install libtensorflow 2.15.0. The following commands can be used to install it on Ubuntu:

```bash
# run with root privileges
curl -L https://storage.googleapis.com/tensorflow/libtensorflow/libtensorflow-cpu-linux-x86_64-2.15.0.tar.gz | tar xz --directory /usr/local
ldconfig
```

Pre-compiled binaries are built with AVX instructions. If your machine doesn't support AVX instructions, you may rebuild libtensorflow from source. [Tensorflow build guide](https://github.com/tensorflow/tensorflow/blob/master/tensorflow/tools/lib_package/README.md) may help you with that.

### Installing OpenCV

You can install OpenCV with [tutorial](https://docs.opencv.org/4.x/d7/d9f/tutorial_linux_install.html). After installation, you can check whether GoCV can work with OpenCV by running the following command:
```bash
cd $GOPATH/src/gocv.io/x/gocv
go run ./cmd/version/main.go
```

If everything is installed correctly, you should see the OpenCV version like this:
```
gocv version: 0.36.1
opencv lib version: 4.8.0
```
## Running

The project is separated into two workers, which one of them is head worker. You should run the head worker first, then run the other worker. You can run the head worker with the following command:
```bash
go run head/main.go head --addr <head_addr> --port <head_port>
```

Then you can run the other worker with the following command:
```bash
go run worker/main.go worker --addr <worker_addr> --remote_addr <head_addr> --remote_port <head_port>
```

To deploy and execute DAG on the cluster, run `clients/py_example.py` with `actorc` installed.
