package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/component-base/cli"

	"github.com/HappyLadySauce/TraveLight/cmd/app"
)

const (
	basename = "TraveLight"
)

// @title TraveLight API
// @version 1.0
// @description TraveLight 后端 API 文档。
// @BasePath /
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main() {
	// 创建可取消的根 context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 设置信号处理，实现优雅关闭
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		// 可以在这里添加日志记录接收到的信号
		_ = sig
		cancel()
	}()

	cmd := app.NewAPICommand(ctx, basename)
	code := cli.Run(cmd)
	os.Exit(code)
}
