# 云边协同应用的 Actor 平台

## 构建

### 依赖

首先安装 `pkg-config`, `libczmq-dev`。在 Ubuntu 上，可以通过以下命令安装:

```bash
sudo apt-get install pkg-config libczmq-dev
```

安装用于生成 ProtoBuf 语言绑定的 [Node.JS](https://nodejs.org/)。推荐使用 [NVM](https://github.com/nvm-sh/nvm/) 作为安装器，防止系统包管理工具提供过是的版本。安装完成后，在 [proto 文件夹下](proto/)运行:

```bash
npm install
bash protobuf-gen.sh
```

这一操作会自动安装依赖并生成 Go、Python 和 TypeScript ProtoBuf 消息绑定。

### ActorC 安装

为了和平台后端进行交互，你需要安装 `actorc` 客户端。当前我们支持 `Python` 版本的包，你可以在 `clients/py` 文件夹下运行：
```bash
pip install -e .
```
来安装它。

### 运行示例程序

在 `examples` 文件夹下存放了一个分布式目标检测示例程序，它通过读取预训练模型并对相机拍到的图片进行目标检测并输出。你可以通过 [README](examples/README-cn.md) 来部署和运行。
