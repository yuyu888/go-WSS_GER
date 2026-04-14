package wsServer

import (
	"context"
	"fmt"
	"github.com/smallnest/rpcx/server"
	"net"
	"wssgo/config"
	"wssgo/model"
)

type TransitData struct{}

func (t *TransitData) Dispatch(ctx context.Context, args *model.Message, reply *model.Reply) error {
	WsManager.DoSendMsgToWssid(args.Wssid, []byte(args.Content))
	return nil
}

func InitRpcServer() {
	addr := net.JoinHostPort(config.ServiceConf.LocalIp, config.ServiceConf.RpcConf.Port)
	network := config.ServiceConf.RpcConf.NetWork
	s := server.NewServer()
	s.RegisterName(config.ServiceConf.RpcConf.RegisterName, new(TransitData), "")
	fmt.Printf("rpcServer is run on %s\n", addr)
	if err := s.Serve(network, addr); err != nil {
		fmt.Printf("rpcServer stopped: %v\n", err)
	}
}
