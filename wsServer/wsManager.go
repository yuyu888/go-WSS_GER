package wsServer

import (
    "fmt"
    //"encoding/json"
    "wssgo/model"
    "wssgo/config"
)

//客户端管理
type ClientManager struct {
    //web端发送来的的message我们用broadcast来接收，并最后分发给所有的client
    //新创建的长连接client
    register chan *Client
    //新注销的长连接client
    unregister chan *Client
}


//客户端 Client
type Client struct {
    //用户id
    id string
    //连接的socket
    wsConn *wsConnection

}

//创建客户端管理者
var WsManager = ClientManager{
    register:   make(chan *Client),
    unregister: make(chan *Client),
}

func (WsManager *ClientManager) ProcLoop() {
    usersession := model.NewUserSession()
    for {
        select {
            //如果有新的连接接入,就通过channel把连接传递给conn
            case conn := <-WsManager.register:
            //把客户端的连接设置为true
                WsClientPools.save(conn.id, conn)
                //WsManager.clients[conn.id] = conn
                uid := conn.wsConn.loginUid
                if uid=="" {
                    uid = conn.wsConn.deviceId
                }
                wsInfo := &model.Session{WsServerAddr:config.ServiceConf.LocalIp, WssId:conn.wsConn.wssid}
                err := usersession.SaveInfo(uid, conn.wsConn.deviceId, wsInfo)
                fmt.Println("================")

                fmt.Println(uid)
                fmt.Println(conn.wsConn.deviceId)
                fmt.Println(wsInfo)
                fmt.Println(err)
                fmt.Println("================")

        //如果连接断开了
            case conn := <-WsManager.unregister:
                WsClientPools.remove(conn.id)
        }
    }
}

func (WsManager *ClientManager) doRegister(conn *wsConnection) {
    fmt.Println("111111")
    //每一次连接都会新开一个client，client.id通过uuid生成保证每次都是不同的
    clientId := conn.wssid
    client := &Client{id: clientId, wsConn: conn}
    //注册一个新的链接
    WsManager.register <- client
}

func (WsManager *ClientManager) doUnRegister(conn *wsConnection) {
    clientId := conn.wssid
    client := &Client{id: clientId, wsConn: conn}
    //注册一个新的链接
    WsManager.unregister <- client

}

func (WsManager *ClientManager) DoBroadcast(deviceId string, message []byte, uid string) {
    if uid==""{
        uid = deviceId
    }
}

func (WsManager *ClientManager) DoSendMsgToWssid(wssid string, message []byte) {
        cl, ok:= WsClientPools.get(wssid)
        if ok {
            cl.wsConn.wsWrite(1, message)
        }else{
            fmt.Println("websocket link is drop")
        }
}