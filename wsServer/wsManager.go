package wsServer

import (
    "fmt"
    //"encoding/json"
)

//客户端管理
type ClientManager struct {
    //客户端 map 储存并管理所有的长连接client，在线的为true，不在的为false
    clients map[string]*Client
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
    clients:    make(map[string]*Client),
}

func (WsManager *ClientManager) ProcLoop() {
    for {
        select {
            //如果有新的连接接入,就通过channel把连接传递给conn
            case conn := <-WsManager.register:
            //把客户端的连接设置为true
                WsClientPools.save(conn.id, conn)
            //如果连接断开了
            case conn := <-WsManager.unregister:
                WsClientPools.remove(conn.id)
        }
    }
}

func (WsManager *ClientManager) doRegister(conn *wsConnection) {
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

func (WsManager *ClientManager) doBroadcast(deviceId string, message []byte, uid string) {
    if uid==""{
        uid = deviceId
    }

    //user := model.UserSession{Uid:uid, DeviceId:deviceId}
    //userInfo, err := user.GetInfo()
    //if(err==nil){
    //    userSession := model.Session{};
    //    uerr := json.Unmarshal([]byte(userInfo[deviceId]), &userSession);
    //    if uerr == nil {
    //        cl, ok:= WsClientPools.get(userSession.WssId)
    //        if ok {
    //            cl.wsConn.wsWrite(1, message)
    //        }else{
    //            err := user.DelInfo()
    //            fmt.Println(err)
    //        }
    //    }else{
    //        err := user.DelInfo()
    //        fmt.Println(err)
    //    }
    //}

    //if cl, ok := WsManager.clients[wssid]; ok {
    //    fmt.Println("广播x"+wssid)
    //    cl.wsConn.wsWrite(1, message)
    //}else{
    //    fmt.Println("websocket link is drop")
    //}
}

func (WsManager *ClientManager) doSendMsgToWssid(wssid string, message []byte) {
    if cl, ok := WsManager.clients[wssid]; ok {
        cl.wsConn.wsWrite(1, message)
    }else{
        fmt.Println("websocket link is drop")
    }
}