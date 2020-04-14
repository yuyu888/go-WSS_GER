package httpServer


import (
    "fmt"
    "net/http"
    "time"
    "wssgo/wsServer"
    "wssgo/model"
    // "log"
    //"sync"
    //"sync/atomic"
)

func httpHandlerIndex(w http.ResponseWriter, r *http.Request) {


    query := r.URL.Query()
    //msg := query.Get("msg")
    uid := query.Get("uid")
    if uid!="" {
        c1 := http.Cookie{
            Name:     "uid",
            Value:    uid,
            HttpOnly: true,
        }
        // 把cookie写入客户端
        http.SetCookie(w, &c1)
    }

    //message := r.PostFormValue("params")
    //fmt.Fprintln(w, message)
    //
    //result  := msg + message + " is task return"
    //fmt.Fprintln(w, result)

    result  := "hello word"
    fmt.Fprintln(w, result)
}

func httpHandlerTest(w http.ResponseWriter, r *http.Request) {
    time.Sleep(2 * time.Second)
    query := r.URL.Query()
    id := query.Get("id")
    message := r.PostFormValue("msg")
    result  := id + message + " is task return"
    fmt.Fprintln(w, result)
}

func httpHandlerSendMsgToWssid(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query()
    wssid := query.Get("wssid")
    message := query.Get("msg")
    wsServer.WsManager.DoSendMsgToWssid(wssid, []byte(message))
    fmt.Fprintln(w, "信息：" + message+" 发送给 " + wssid)
}

func httpHandlerSendMsg(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query()
    uid := query.Get("uid")
    deviceId := query.Get("deviceid")
    msg := r.PostFormValue("msg")

    if len(uid) == 0 && len(deviceId) == 0 {
        resp := `{"errcode":4001, "response_data":"uid, deviceId is empty"}`
        fmt.Fprintln(w, resp)
        return ;
    }
    if len(msg) == 0 {
        resp := `{"errcode":4002, "response_data":"msg is empty"}`
        fmt.Fprintln(w, resp)
        return ;
    }

    usersession := model.NewUserSession()

    userinfo, err := usersession.GetInfo(uid, deviceId)
    if err != nil{
        resp := `{"errcode":5003, "something is wrong for getuserinfo "}`
        fmt.Fprintln(w, resp)
        return ;
    }else{
        fmt.Fprintln(w, userinfo)
    }
    //var sendCount int32 = 0
    for device_id, sessionInfo := range userinfo {
        serverAddr, wssid, err := usersession.GetWsServer(sessionInfo);
        if err != nil || len(serverAddr) == 0 {
            continue;
        }
        fmt.Fprintln(w,fmt.Sprintf("%s", serverAddr))
        fmt.Fprintln(w, device_id)
        fmt.Fprintln(w, sessionInfo)
        message := &model.Message{Content:msg, Wssid:fmt.Sprintf("%s", wssid)}

        push(message, fmt.Sprintf("%s", serverAddr));
    }

    //data := map[string]int32{"send_count": sendCount};
}

func push( msg *model.Message, serverAddr string) {
    cl, ok := GetRpcClient(serverAddr, 2);
    if !ok {
        fmt.Println("rpcclient is wrong")
        return ;
    }
    reply := new(model.Reply);
    RpcCall(cl, msg, reply);
    if reply.Status > 0 {
        return ;
    }
    return;
}

//func httpHandlerSetRedis(w http.ResponseWriter, r *http.Request) {
//    query := r.URL.Query()
//    uid := query.Get("uid")
//    deviceId := query.Get("deviceid")
//    fmt.Fprintln(w, "uid：" + uid+" deviceid: " + deviceId)
//
//    usersession := model.NewUserSession()
//    wsInfo := &model.Session{WsServerAddr:"127.0.0.1", WssId:"123567"}
//    err := usersession.SaveInfo(uid, deviceId, wsInfo)
//    fmt.Println(err)
//}



func Init() {
    fmt.Println("httpServer is run")
    http.HandleFunc("/", httpHandlerIndex)
    http.HandleFunc("/test", httpHandlerTest)
    //http.HandleFunc("/setredis", httpHandlerSetRedis)
    http.HandleFunc("/sendmsgtowssid", httpHandlerSendMsgToWssid)
    http.HandleFunc("/sendmsg", httpHandlerSendMsg)
    http.ListenAndServe("0.0.0.0:80", nil)
}
