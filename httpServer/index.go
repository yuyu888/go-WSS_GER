package httpServer


import (
    "fmt"
    "net/http"
    "time"
    "wssgo/wsServer"
    // "log"
)

func httpHandlerIndex(w http.ResponseWriter, r *http.Request) {
    result  := "hello word"
    fmt.Fprintln(w, result)

    //query := r.URL.Query()
    //msg := query.Get("msg")
    //uid := query.Get("uid")
    //if uid!="" {
    //    c1 := http.Cookie{
    //        Name:     "mfw_uid",
    //        Value:    uid,
    //        HttpOnly: true,
    //    }
    //    // 把cookie写入客户端
    //    http.SetCookie(w, &c1)
    //}

    //message := r.PostFormValue("params")
    //fmt.Fprintln(w, message)
    //
    //result  := msg + message + " is task return"
    //fmt.Fprintln(w, result)
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


func Init() {
    fmt.Println("httpServer is run")
    http.HandleFunc("/", httpHandlerIndex)
    http.HandleFunc("/test", httpHandlerTest)
    http.HandleFunc("/sendmsgtowssid", httpHandlerSendMsgToWssid)
    http.ListenAndServe("0.0.0.0:80", nil)
}
