package httpServer

import (
	"context"
	"github.com/smallnest/rpcx/client"
	"strings"
	"time"
	"wssgo/config"
	"wssgo/libs"
	"wssgo/model"
	// "errors"
	//"fmt"
)

var (
	rpcClientList map[string]client.XClient
)

// 初始化rpc服务
func InitRpcClient() {
	d := client.NewEtcdV3Discovery(config.ServiceConf.RpcConf.BasePath, config.ServiceConf.RpcConf.RegisterName, config.ServiceConf.EtcdConf.ServerAddr, nil)
	rpcClientList = make(map[string]client.XClient, len(d.GetServices()))
	option := client.DefaultOption
	option.Retries = 10
	option.GenBreaker = func() client.Breaker {
		return client.NewConsecCircuitBreaker(5, 30*time.Second)
	}
	for _, rpcConf := range d.GetServices() {
		d := client.NewPeer2PeerDiscovery(rpcConf.Key, "")
		index := strings.Index(rpcConf.Key, "@")

		serverIp := []byte(rpcConf.Key)[index+1:]
		rpcClientList[string(serverIp)] = client.NewXClient(config.ServiceConf.RpcConf.RegisterName, client.Failtry, client.RandomSelect, d, option)
	}
	//js, _ := json.Marshal(rpcClientList);
	//fmt.Println(js)
	return
}

// get rpc client
func GetRpcClient(rpcServerAddr string, retry int) (client.XClient, bool) {
	if rpcClientList == nil {
		InitRpcClient()
	}
	addr := rpcServerAddr + ":" + config.ServiceConf.RpcConf.Port
	for i := 0; i <= retry; i++ {
		if cl, ok := rpcClientList[addr]; ok && cl != nil {
			return cl, true
		}
		if i < retry {
			InitRpcClient()
		}
	}
	return nil, false
}

// rpc 调用
func RpcCall(rpcClient client.XClient, msg *model.Message, reply *model.Reply) {
	reply.Status = 0
	defer func() {
		if err := recover(); err != nil {
			reply.Status = 1
			reply.Data = err
			libs.Logger.Error("rpc client call error:", msg, err)
		}
	}()

	err := rpcClient.Call(context.Background(), "Dispatch", msg, reply)
	if err != nil {
		reply.Status = 1
		reply.Data = err
	}
}
