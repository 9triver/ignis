package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/9triver/ignis/platform"
	"github.com/sirupsen/logrus"
)

func main() {
	// 解析命令行参数
	rpcAddr := flag.String("addr", "localhost:50052", "RPC server address (format: host:port)")
	flag.Parse()

	// 初始化日志
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logrus.SetLevel(logrus.InfoLevel)

	// 创建根上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建平台实例
	logrus.Infof("Starting Ignis platform on %s", *rpcAddr)
	p := platform.NewPlatform(ctx, *rpcAddr, nil)

	// 启动平台（在 goroutine 中运行）
	errCh := make(chan error, 1)
	go func() {
		if err := p.Run(); err != nil && err != context.Canceled {
			errCh <- err
		}
	}()

	// 等待中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		logrus.Infof("Received signal: %v, shutting down...", sig)
		cancel()
	case err := <-errCh:
		logrus.Errorf("Platform error: %v", err)
		cancel()
	}

	// 等待平台关闭
	<-ctx.Done()
	logrus.Info("Ignis platform shutdown complete")
}
