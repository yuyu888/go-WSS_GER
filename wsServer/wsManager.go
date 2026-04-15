package wsServer

import (
	"fmt"
	"wssgo/config"
	"wssgo/libs"
	"wssgo/model"
)

// CrossNodeSendFunc 跨节点推送回调，由 httpServer.Init() 注入，避免循环导入
var CrossNodeSendFunc func(serverAddr, wssid, message string)

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
				continue
			}
			// 保存 wssid → serverAddr 反向映射，供跨节点 broadcast 路由
			if err := usersession.SaveWssid(conn.wsConn.wssid, config.ServiceConf.LocalIp); err != nil {
				libs.Logger.Errorf("SaveWssid failed, wssid=%s err=%v", conn.wsConn.wssid, err)
			}

		case conn := <-WsManager.unregister:
			WsClientPools.remove(conn.id)
			// 清理 wssid 反向映射
			if err := usersession.DelWssid(conn.wsConn.wssid); err != nil {
				libs.Logger.Errorf("DelWssid failed, wssid=%s err=%v", conn.wsConn.wssid, err)
			}
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

// Shutdown 清理所有在线连接在 Redis 中的会话记录，在进程退出前调用。
func (WsManager *ClientManager) Shutdown() {
	usersession := model.NewUserSession()
	WsClientPools.forEach(func(_ string, client *Client) {
		conn := client.wsConn
		uid := conn.loginUid
		if uid == "" {
			uid = conn.deviceId
		}
		if err := usersession.DelInfo(uid, conn.deviceId); err != nil {
			libs.Logger.Errorf("Shutdown DelInfo failed, uid=%s deviceId=%s wssid=%s err=%v",
				uid, conn.deviceId, conn.wssid, err)
		}
		if err := usersession.DelWssid(conn.wssid); err != nil {
			libs.Logger.Errorf("Shutdown DelWssid failed, wssid=%s err=%v", conn.wssid, err)
		}
	})
}
