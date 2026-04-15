package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"wssgo/config"
	"wssgo/httpServer"
	"wssgo/libs"
	"wssgo/wsServer"
)

var (
	Env = flag.String("env", "prod", "env flag")
)

func main() {
	flag.Parse()
	config.InitServiceConfig(*Env)
	libs.InitLogger("wssgo.log", "wssgo", "info")
	localIp, err := libs.GetLocalIp()
	if err != nil {
		log.Fatal("get local ip error:" + err.Error())
	}
	config.ServiceConf.LocalIp = localIp
	go wsServer.InitRpcServer()
	wsServer.Init()
	srv := httpServer.Init()

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	libs.Logger.Info("shutting down, cleaning up redis sessions...")

	// 1. 清理所有在线连接在 Redis 中的会话记录
	wsServer.WsManager.Shutdown()

	// 2. 优雅关闭 HTTP Server（30 秒超时，等待进行中的请求完成）
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		libs.Logger.Errorf("http server shutdown error: %v", err)
	}

	libs.Logger.Info("shutdown complete")
}
