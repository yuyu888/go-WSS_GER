package wsServer

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
	"net/http"
	"sync"
	"time"
)

const (
	// 心跳间隔：100 秒无消息则发送心跳
	heartbeatInterval = 100 * time.Second
	// 读超时：超过该时间没有读到任何帧则关闭连接
	// 设为心跳间隔的 1.5 倍，给客户端足够时间响应
	readDeadline = 150 * time.Second
)

// http 升级 websocket 协议的配置
var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return isOriginAllowed(origin)
	},
}

// 客户端读写消息
type wsMessage struct {
	messageType int
	data        []byte
}

// 客户端连接
type wsConnection struct {
	wsSocket  *websocket.Conn // 底层 websocket
	inChan    chan *wsMessage  // 读队列
	outChan   chan *wsMessage  // 写队列
	mutex     sync.Mutex      // 避免重复关闭管道
	isClosed  bool
	closeChan chan byte // 关闭通知
	wssid     string   // ws 链接 id，连接建立时产生
	loginUid  string   // 登录用户 uid
	deviceId  string   // 设备号
	isDynamic bool     // 是否活跃
}

func (wsConn *wsConnection) wsReadLoop() {
	defer close(wsConn.inChan)
	for {
		// 每次读之前刷新读超时，保证长空闲连接能被检测到
		_ = wsConn.wsSocket.SetReadDeadline(time.Now().Add(readDeadline))

		msgType, data, err := wsConn.wsSocket.ReadMessage()
		if err != nil {
			goto error
		}
		wsConn.isDynamic = true
		req := &wsMessage{msgType, data}

		select {
		case wsConn.inChan <- req:
		case <-wsConn.closeChan:
			goto closed
		}
	}
error:
	wsConn.wsClose()
closed:
}

func (wsConn *wsConnection) wsWriteLoop() {
	defer close(wsConn.outChan)
	for {
		select {
		case msg, ok := <-wsConn.outChan:
			if !ok || msg == nil {
				goto closed
			}
			wsConn.isDynamic = true
			if err := wsConn.wsSocket.WriteMessage(msg.messageType, msg.data); err != nil {
				goto error
			}
		case <-wsConn.closeChan:
			goto closed
		}
	}
error:
	wsConn.wsClose()
closed:
}

func (wsConn *wsConnection) procLoop() {
	// 启动心跳 goroutine
	go func() {
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if wsConn.isDynamic {
					wsConn.isDynamic = false
					continue
				}
				heartBeat := `{"errcode":200, "wssid":"` + wsConn.wssid + `","request_id":"","response_data":"heartbeat from server","action":"wsHeartBeat"}`
				if err := wsConn.wsWrite(websocket.TextMessage, []byte(heartBeat)); err != nil {
					fmt.Println("heartbeat fail")
					wsConn.wsClose()
					return
				}
			case <-wsConn.closeChan:
				return
			}
		}
	}()

	// 限制单连接最大并发处理数为 2
	sem := make(chan struct{}, 2)
	var wg sync.WaitGroup

	for {
		msg, err := wsConn.wsRead()
		if err != nil {
			fmt.Println("read fail")
			break
		}
		// 在派发 goroutine 之前先 Add，避免 break 时 wg.Wait 看到未完成的计数
		wg.Add(1)
		sem <- struct{}{}
		go func(data []byte) {
			defer wg.Done()
			defer func() { <-sem }()
			doProcess(data, wsConn)
		}(msg.data)
	}

	wg.Wait()
}

func doProcess(msg []byte, wsConn *wsConnection) {
	process(msg, wsConn)
}

func wsHandler(resp http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	deviceId := query.Get("device_id")

	wsSocket, err := wsUpgrader.Upgrade(resp, req, nil)
	if err != nil {
		return
	}
	wsConn := &wsConnection{
		wsSocket:  wsSocket,
		inChan:    make(chan *wsMessage, 1000),
		outChan:   make(chan *wsMessage, 1000),
		closeChan: make(chan byte),
		isClosed:  false,
	}
	if deviceId == "" {
		errResp := `{"errcode":4001, "wssid":"` + wsConn.wssid + `","request_id":"","response_data":"Lack of device_id","action":"error"}`
		wsConn.wsSocket.WriteMessage(1, []byte(errResp))
		wsConn.wsSocket.Close()
		return
	}
	wsConn.deviceId = deviceId

	loginUid, err := req.Cookie("uid")
	if err == nil {
		wsConn.loginUid = loginUid.Value
		fmt.Println(loginUid.Value)
	} else {
		wsConn.loginUid = ""
	}
	wsConn.wsInit()
	go wsConn.procLoop()
	go wsConn.wsReadLoop()
	go wsConn.wsWriteLoop()
}

func (wsConn *wsConnection) wsWrite(messageType int, data []byte) error {
	select {
	case wsConn.outChan <- &wsMessage{messageType, data}:
	case <-wsConn.closeChan:
		return errors.New("websocket closed")
	}
	return nil
}

func (wsConn *wsConnection) wsRead() (*wsMessage, error) {
	select {
	case msg, ok := <-wsConn.inChan:
		if !ok || msg == nil {
			return nil, errors.New("websocket closed")
		}
		return msg, nil
	case <-wsConn.closeChan:
	}
	return nil, errors.New("websocket closed")
}

func (wsConn *wsConnection) wsClose() {
	wsConn.mutex.Lock()
	if wsConn.isClosed {
		wsConn.mutex.Unlock()
		return
	}
	wsConn.isClosed = true
	close(wsConn.closeChan)
	wsConn.mutex.Unlock()
	WsManager.doUnRegister(wsConn)
	wsConn.wsSocket.Close()
}

func (wsConn *wsConnection) wsInit() {
	wsConn.wssid = uuid.NewV4().String()
	resp := `{"errcode":200, "wssid":"` + wsConn.wssid + `","request_id":"","response_data":"websocket create success","action":"wsInit"}`
	wsConn.wsWrite(websocket.TextMessage, []byte(resp))
	WsManager.doRegister(wsConn)
}

func Init() {
	fmt.Println("wsServer is run")
	InitUrlList()
	go WsManager.ProcLoop()
	http.HandleFunc("/ws", wsHandler)
}
