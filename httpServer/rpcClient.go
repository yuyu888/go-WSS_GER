package httpServer

import (
	"context"
	"fmt"
	"github.com/smallnest/rpcx/client"
	"net"
	"sync"
	"time"
	"wssgo/config"
	"wssgo/libs"
	"wssgo/model"
)

// rpcClientCache 按 ip:port 缓存已建立的 RPC 客户端
var rpcClientCache sync.Map

// GetRpcClient 按目标地址获取 RPC 客户端，不存在时自动创建并缓存
func GetRpcClient(rpcServerAddr string) (client.XClient, bool) {
	addr := rpcServerAddr
	if _, _, err := net.SplitHostPort(rpcServerAddr); err != nil {
		// 只有 IP，没有端口，自动拼上配置的 RPC 端口
		addr = net.JoinHostPort(rpcServerAddr, config.ServiceConf.RpcConf.Port)
	}

	// 先查缓存
	if v, ok := rpcClientCache.Load(addr); ok {
		return v.(client.XClient), true
	}

	// 未命中，新建连接
	cl, err := newRpcClient(addr)
	if err != nil {
		fmt.Printf("create rpc client to %s failed: %v\n", addr, err)
		return nil, false
	}

	// 存入缓存，若并发时已有其他 goroutine 存入则使用已有的
	actual, _ := rpcClientCache.LoadOrStore(addr, cl)
	return actual.(client.XClient), true
}

// newRpcClient 创建到指定地址的点对点 RPC 客户端
func newRpcClient(addr string) (client.XClient, error) {
	key := fmt.Sprintf("%s@%s", config.ServiceConf.RpcConf.NetWork, addr)
	d := client.NewPeer2PeerDiscovery(key, "")
	option := client.DefaultOption
	option.Retries = 3
	option.ConnectTimeout = 5 * time.Second
	option.GenBreaker = func() client.Breaker {
		return client.NewConsecCircuitBreaker(5, 30*time.Second)
	}
	cl := client.NewXClient(
		config.ServiceConf.RpcConf.RegisterName,
		client.Failtry,
		client.RandomSelect,
		d,
		option,
	)
	return cl, nil
}

// RpcCall 发起 RPC 调用
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
