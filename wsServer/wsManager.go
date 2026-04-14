package wsServer

import (
	"fmt"
	"wssgo/config"
	"wssgo/libs"
	"wssgo/model"
)

// 客户端管理
type ClientManager struct {
	// 新创建的长连接 client
	register chan *Client
	// 注销的长连接 client
	unregister chan *Client
}

// 客户端 Client
type Client struct {
	// 连接唯一 id（wssid）
	id     string
	wsConn *wsConnection
}

// 创建客户端管理者
var WsManager = ClientManager{
	register:   make(chan *Client),
	unregister: make(chan *Client),
}

func (WsManager *ClientManager) ProcLoop() {
	usersession := model.NewUserSession()
	for {
		select {
		case conn := <-WsManager.register:
			WsClientPools.save(conn.id, conn)
			uid := conn.wsConn.loginUid
			if uid == "" {
				uid = conn.wsConn.deviceId
			}
			wsInfo := &model.Session{
				WsServerAddr: config.ServiceConf.LocalIp,
				WssId:        conn.wsConn.wssid,
			}
			if err := usersession.SaveInfo(uid, conn.wsConn.deviceId, wsInfo); err != nil {
				// 会话写入失败：记录日志并关闭连接，避免产生无法被推送到的幽灵连接
				libs.Logger.Errorf("SaveInfo failed, uid=%s deviceId=%s wssid=%s err=%v",
					uid, conn.wsConn.deviceId, conn.wsConn.wssid, err)
				WsClientPools.remove(conn.id)
				conn.wsConn.wsClose()
			}

		case conn := <-WsManager.unregister:
			WsClientPools.remove(conn.id)
		}
	}
}

func (WsManager *ClientManager) doRegister(conn *wsConnection) {
	clientId := conn.wssid
	client := &Client{id: clientId, wsConn: conn}
	WsManager.register <- client
}

func (WsManager *ClientManager) doUnRegister(conn *wsConnection) {
	clientId := conn.wssid
	client := &Client{id: clientId, wsConn: conn}
	WsManager.unregister <- client
}

func (WsManager *ClientManager) DoSendMsgToWssid(wssid string, message []byte) {
	cl, ok := WsClientPools.get(wssid)
	if ok {
		cl.wsConn.wsWrite(1, message)
	} else {
		fmt.Println("websocket link is drop")
	}
}
