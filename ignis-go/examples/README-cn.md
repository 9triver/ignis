# Actor-based Object Detection Application

## 依赖

除了在[主项目 README](../README.md) 中要求的依赖，本示例还额外要求安装 [libtensorflow](https://www.tensorflow.org/install/lang_c) 和 [OpenCV](https://opencv.org/)。他们是 [Tensorflow Go](https://pkg.go.dev/github.com/wamuir/graft@v0.8.1/tensorflow) 和 [GoCV](https://pkg.go.dev/gocv.io/x/gocv) 的重要前提条件。

### 安装 Webcam

为了让相机工作，你需要在 `/dev/video0` 下存在支持 webcam 协议的相机。如果路径不同，请在 `main.go` 中修改。请确保 webcam 驱动已经正确安装。

### 安装 libtensorflow

按照[官方安装手册](https://github.com/tensorflow/build/tree/master/golang_install_guide)，你可以运行下列命令来安装 libtensorflow-2.15.0：

```bash
# run with root privileges
curl -L https://storage.googleapis.com/tensorflow/libtensorflow/libtensorflow-cpu-linux-x86_64-2.15.0.tar.gz | tar xz --directory /usr/local
ldconfig
```

官方提供的预编译包默认需要 AVX 指令集。如果你的机器不支持，则需要从源代码重新构建。请参考[Tensorflow 构建指南](https://github.com/tensorflow/tensorflow/blob/master/tensorflow/tools/lib_package/README.md)。

### 安装 OpenCV

根据[官方教程](https://docs.opencv.org/4.x/d7/d9f/tutorial_linux_install.html)的步骤安装 OpenCV。安装后，运行下列命令检查安装的完整性：
```bash
cd $GOPATH/src/gocv.io/x/gocv
go run ./cmd/version/main.go
```

如果安装正确，你将得到类似以下输出：
```
gocv version: 0.36.1
opencv lib version: 4.8.0
```
## 运行

整个系统包含多个worker，其中一个为 head 节点。你应当首先运行 head 节点，再启动其它的 worker 节点。

启动 head：
```bash
go run head/main.go head --addr <head_addr> --port <head_port>
```

启动 worker （可重复多次）：
```bash
go run worker/main.go worker --addr <worker_addr> --remote_addr <head_addr> --remote_port <head_port>
```

为了在集群上部署和运行任务，你可以在安装 `actorc` 后运行 `clients/py_example.py`。
