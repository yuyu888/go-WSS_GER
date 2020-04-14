package httpServer
import (
    "github.com/smallnest/rpcx/client"
    "time"
    "strings"
    "wssgo/libs"
    "wssgo/model"
    "context"
    "wssgo/config"
    // "errors"
    //"fmt"
)
var (
    rpcClientList map[string]client.XClient
)
//初始化rpc服务
func InitRpcClient() {
    d := client.NewEtcdV3Discovery(config.ServiceConf.RpcConf.BasePath, config.ServiceConf.RpcConf.RegisterName, config.ServiceConf.EtcdConf.ServerAddr, nil);
    rpcClientList = make(map[string]client.XClient, len(d.GetServices()));
    option := client.DefaultOption
    option.Retries = 10;
    option.GenBreaker = func() client.Breaker {
        return client.NewConsecCircuitBreaker(5, 30*time.Second);
    }
    for _, rpcConf := range d.GetServices() {
        d := client.NewPeer2PeerDiscovery(rpcConf.Key, "");
        index := strings.Index(rpcConf.Key, "@");

        serverIp := []byte(rpcConf.Key)[index+1:];
        rpcClientList[string(serverIp)] = client.NewXClient(config.ServiceConf.RpcConf.RegisterName, client.Failtry, client.RandomSelect, d, option);
    }
    //js, _ := json.Marshal(rpcClientList);
    //fmt.Println(js)
    return ;
}
//get rpc client
func GetRpcClient(rpcServerAddr string, retry int) (client.XClient, bool) {
    cl := rpcClientList[rpcServerAddr+":"+config.ServiceConf.RpcConf.Port];
    if tmpCl, ok := cl.(client.XClient); !ok {
        if (retry > 0) {
            InitRpcClient();
            return GetRpcClient(rpcServerAddr, retry-1);
        }
        return tmpCl, false;
    }
    return cl, true;
}
//rpc 调用
func RpcCall(rpcClient client.XClient, msg *model.Message, reply *model.Reply){
    reply.Status = 0;
    defer func(){
        if err := recover(); err != nil {
            reply.Status = 1;
            reply.Data = err;
            libs.Logger.Error("rpc client call error:", msg, err);
        }
    }()

    err := rpcClient.Call(context.Background(), "Dispatch",  msg, reply);
    if (err != nil){
        reply.Status = 1;
        reply.Data = err;
    }
}
