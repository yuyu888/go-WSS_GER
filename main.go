package main

import (
	"flag"
	"log"
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
	httpServer.Init()
}
