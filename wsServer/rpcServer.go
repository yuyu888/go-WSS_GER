package wsServer
import (
    //"context"
    "github.com/smallnest/rpcx/server"
    "github.com/smallnest/rpcx/serverplugin"
    "github.com/rcrowley/go-metrics"
    "log"
    "time"
    "wssgo/config"
    //"encoding/json"
)

type TransitData struct{

}

func (t *TransitData) Dispatch () error {
    return nil;
}

func InitRpcServer(){
    addr := config.ServiceConf.RpcConf.Addr
    network := config.ServiceConf.RpcConf.NetWork
    s := server.NewServer();
    addRegistryPlugin(s, network, addr);
    s.RegisterName(config.ServiceConf.RpcConf.RegisterName, new(TransitData), "");
    s.Serve(network, addr);
}

func addRegistryPlugin(s *server.Server, network, addr string){
    //libs.Logger.Info("rpc server" + network + "add:" + addr + "etcd:" + string(config.ServiceConf.EtcdConf.ServerAddr[0]))
    r := &serverplugin.EtcdV3RegisterPlugin{
        ServiceAddress : network + "@" + addr,
        EtcdServers : config.ServiceConf.EtcdConf.ServerAddr,
        BasePath :	config.ServiceConf.RpcConf.BasePath,
        Metrics	:	metrics.NewRegistry(),
        UpdateInterval : time.Minute,
    }
    err := r.Start();
    if (err != nil){
        log.Fatal(err);
    }
    s.Plugins.Add(r);
}