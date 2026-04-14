package httpServer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"wssgo/config"
	"wssgo/model"
	"wssgo/wsServer"
)

type httpResp struct {
	ErrCode      int    `json:"errcode"`
	ResponseData string `json:"response_data"`
}

func jsonResp(w http.ResponseWriter, errCode int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	body, _ := json.Marshal(&httpResp{ErrCode: errCode, ResponseData: msg})
	w.Write(body)
}

func httpHandlerIndex(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	uid := query.Get("uid")
	if uid != "" {
		c1 := http.Cookie{
			Name:     "uid",
			Value:    uid,
			HttpOnly: true,
		}
		http.SetCookie(w, &c1)
	}
	fmt.Fprintln(w, "hello word")
}

func httpHandlerTest(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	id := query.Get("id")
	message := r.PostFormValue("msg")
	result := id + message + " is task return"
	fmt.Fprintln(w, result)
}

func httpHandlerSendMsgToWssid(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	wssid := query.Get("wssid")
	message := query.Get("msg")
	wsServer.WsManager.DoSendMsgToWssid(wssid, []byte(message))
	fmt.Fprintln(w, "信息："+message+" 发送给 "+wssid)
}

func httpHandlerSendMsg(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	uid := query.Get("uid")
	deviceId := query.Get("deviceid")
	msg := r.PostFormValue("msg")

	if len(uid) == 0 && len(deviceId) == 0 {
		jsonResp(w, 4001, "uid, deviceId is empty")
		return
	}
	if len(msg) == 0 {
		jsonResp(w, 4002, "msg is empty")
		return
	}

	usersession := model.NewUserSession()
	userinfo, err := usersession.GetInfo(uid, deviceId)
	if err != nil {
		jsonResp(w, 5003, "get user info failed: "+err.Error())
		return
	}

	var sendCount int
	for device_id, sessionInfo := range userinfo {
		serverAddr, wssid, err := usersession.GetWsServer(sessionInfo)
		if err != nil || len(serverAddr) == 0 {
			continue
		}
		_ = device_id
		message := &model.Message{Content: msg, Wssid: fmt.Sprintf("%s", wssid)}
		push(message, fmt.Sprintf("%s", serverAddr))
		sendCount++
	}

	jsonResp(w, 200, fmt.Sprintf("sent to %d device(s)", sendCount))
}

func push(msg *model.Message, serverAddr string) {
	cl, ok := GetRpcClient(serverAddr)
	if !ok {
		fmt.Println("rpcclient is wrong")
		return
	}
	reply := new(model.Reply)
	RpcCall(cl, msg, reply)
}

func Init() {
	fmt.Println("httpServer is run")
	mux := http.NewServeMux()
	mux.HandleFunc("/", httpHandlerIndex)
	mux.HandleFunc("/test", httpHandlerTest)
	mux.HandleFunc("/sendmsgtowssid", httpHandlerSendMsgToWssid)
	mux.HandleFunc("/sendmsg", httpHandlerSendMsg)

	handler := http.Handler(mux)
	handler = wrapWithBodyLimit(handler, config.ServiceConf.HttpConf.MaxBodyBytes)
	handler = wrapWithRateLimit(handler)

	srv := &http.Server{
		Addr:         config.ServiceConf.HttpConf.Addr,
		Handler:      handler,
		ReadTimeout:  timeoutDuration(config.ServiceConf.HttpConf.ReadTimeoutSec),
		WriteTimeout: timeoutDuration(config.ServiceConf.HttpConf.WriteTimeoutSec),
		IdleTimeout:  timeoutDuration(config.ServiceConf.HttpConf.IdleTimeoutSec),
	}
	if err := srv.ListenAndServe(); err != nil {
		fmt.Printf("http server stopped: %v\n", err)
	}
}
