package wsServer

import (
    "net/http"
    "github.com/gorilla/websocket"
    "errors"
    "fmt"
    "sync"
    "time"
    "github.com/satori/go.uuid"
)

// http升级websocket协议的配置
var wsUpgrader = websocket.Upgrader{
    // 允许所有CORS跨域请求
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

// 客户端读写消息
type wsMessage struct {
    messageType int
    data []byte
}

// 客户端连接
type wsConnection struct {
    wsSocket *websocket.Conn // 底层websocket
    inChan chan *wsMessage	// 读队列
    outChan chan *wsMessage // 写队列
    mutex sync.Mutex	// 避免重复关闭管道
    isClosed bool
    closeChan chan byte  // 关闭通知
    wssid string //ws链接id 连接建立时产生
    mfwUid string // 登录用户uid
    deviceId string //设备号
    isDynamic bool //是否活跃
}

func (wsConn *wsConnection)wsReadLoop() {
    defer close(wsConn.inChan)
    for {
        // 读一个message
        msgType, data, err := wsConn.wsSocket.ReadMessage()
        if err != nil {
            goto error
        }
        wsConn.isDynamic = true
        req := &wsMessage{
            msgType,
            data,
        }

        // 放入请求队列
        select {
        case wsConn.inChan <- req:
        case <- wsConn.closeChan:
            goto closed
        }
    }
    error:
    wsConn.wsClose()
    closed:
}

func (wsConn *wsConnection)wsWriteLoop() {
    defer close(wsConn.outChan)
    for {
        select {
        // 取一个应答
        case msg := <- wsConn.outChan:
            wsConn.isDynamic = true
        // 写给websocket
            if err := wsConn.wsSocket.WriteMessage(msg.messageType, msg.data); err != nil {
                goto error
            }
        case <- wsConn.closeChan:
            goto closed
        }
    }
    error:
    wsConn.wsClose()
    closed:
}

func (wsConn *wsConnection)procLoop() {
    // 启动一个gouroutine发送心跳
    go func() {
        uid := wsConn.mfwUid
        if uid==""{
            uid=wsConn.deviceId
        }
        for {
            time.Sleep(10 * time.Second)
            if wsConn.isDynamic==true{
                wsConn.isDynamic = false
                continue
            }
            heartBeat := `{"wssid":"`+wsConn.wssid+`","request_id":"","response_data":"heartbeat from server","action":"wsHeartBeat"}`
            if err := wsConn.wsSocket.WriteMessage(websocket.TextMessage, []byte(heartBeat)); err != nil {
                fmt.Println("heartbeat fail")
                wsConn.wsClose()
                break
            }
        }
    }()

    // 这是一个同步处理模型（只是一个例子），如果希望并行处理可以每个请求一个gorutine，注意控制并发goroutine的数量!!!
    //for {
    //	msg, err := wsConn.wsRead()
    //	if err != nil {
    //		fmt.Println("read fail")
    //		break
    //	}
    //	reqerr := process(msg.data, wsConn)
    //	if(reqerr!=nil){
    //		err = wsConn.wsWrite(msg.messageType, []byte(fmt.Sprintf("%s", reqerr)))
    //		if err != nil {
    //			fmt.Println("write fail")
    //			break
    //		}
    //	}
    //}

    gch := make(chan bool, 2)
    var wg = sync.WaitGroup{}
    for{
        wg.Add(1)
        gch <- true
        msg, err := wsConn.wsRead()
        if err != nil {
            fmt.Println("read fail")
            break
        }
        go doProcess(msg.data, wsConn, gch, &wg)
    }
    close(gch)
    wg.Wait()
}

func doProcess(msg []byte, wsConn *wsConnection, ch chan bool, wg *sync.WaitGroup){
    defer wg.Done()
    reqerr := process(msg, wsConn)
    if(reqerr!=nil){
        err := wsConn.wsWrite(1, []byte(fmt.Sprintf("%s", reqerr)))
        if err != nil {
            fmt.Println("write fail")
        }
    }
    <- ch
}

func wsHandler(resp http.ResponseWriter, req *http.Request) {
    query := req.URL.Query()
    deviceId := query.Get("device_id")

    // 应答客户端告知升级连接为websocket
    wsSocket, err := wsUpgrader.Upgrade(resp, req, nil)
    if err != nil {
        return
    }
    wsConn := &wsConnection{
        wsSocket: wsSocket,
        inChan: make(chan *wsMessage, 1000),
        outChan: make(chan *wsMessage, 1000),
        closeChan: make(chan byte),
        isClosed: false,
    }
    if deviceId==""{
        resp := `{"wssid":"`+wsConn.wssid+`","request_id":"","response_data":"Lack of device_id","action":"error"}`
        wsConn.wsSocket.WriteMessage(1, []byte(resp))
        wsConn.wsSocket.Close()
        return
    }else{
        wsConn.deviceId = deviceId
    }
    mfwUid, err := req.Cookie("uid")
    if err ==nil{
        wsConn.mfwUid = mfwUid.Value
        fmt.Println(mfwUid.Value)

    }else{
        wsConn.mfwUid =""
    }
    wsConn.wsInit()
    // 处理器
    go wsConn.procLoop()
    // 读协程
    go wsConn.wsReadLoop()
    // 写协程
    go wsConn.wsWriteLoop()
}

func (wsConn *wsConnection)wsWrite(messageType int, data []byte) error {
    select {
    case wsConn.outChan <- &wsMessage{messageType, data}:
    case <- wsConn.closeChan:
        return errors.New("websocket closed")
    }
    return nil
}

func (wsConn *wsConnection)wsRead() (*wsMessage, error) {
    select {
    case msg := <- wsConn.inChan:
        return msg, nil
    case <- wsConn.closeChan:
    }
    return nil, errors.New("websocket closed")
}

func (wsConn *wsConnection)wsClose() {
    wsConn.wsSocket.Close()
    WsManager.doUnRegister(wsConn)
    wsConn.mutex.Lock()
    defer wsConn.mutex.Unlock()
    if !wsConn.isClosed {
        wsConn.isClosed = true
        close(wsConn.closeChan)
    }
}

func (wsConn *wsConnection)wsInit() {
    wsConn.wssid = uuid.NewV4().String()
    resp := `{"wssid":"`+wsConn.wssid+`","request_id":"","response_data":"websocket create success","action":"wsInit"}`
    wsConn.wsWrite(websocket.TextMessage, []byte(resp))
    WsManager.doRegister(wsConn)
}

func Init() {
    fmt.Println("wsServer is run")
    go WsManager.ProcLoop()
    http.HandleFunc("/ws", wsHandler)
    http.ListenAndServe("0.0.0.0:80", nil)
}
