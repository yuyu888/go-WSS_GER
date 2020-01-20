package main

import (
    //"wssgo/wsServer"
    //"wssgo/httpServer"
    "wssgo/config"
    "fmt"
    "flag"
    "wssgo/libs"
    "log"
)

var (
    Env = flag.String("env", "prod", "env flag")
)

func main() {
    flag.Parse()
    fmt.Println(*Env)
    config.InitServiceConfig(*Env)
    localIp, err := libs.GetLocalIp()
    if err != nil {
        log.Fatal("get local ip error:" + err.Error())
    }
    config.ServiceConf.LocalIp = localIp

    //redisConf := libs.GetDefaultRedisConf()
    //redisCli := libs.NewRedis(redisConf)
    //redisCli.Connect()
    ////redisCli.Set("test", "hello", 1)
    //rVal, err:= redisCli.Get("test")
    //fmt.Println(rVal)

    //go httpServer.Init();
    //wsServer.Init();
}