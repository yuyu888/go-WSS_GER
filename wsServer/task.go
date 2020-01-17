package wsServer


import (
	"fmt"
	"wssgo/libs/curl"
    "encoding/json"
    "errors"
    "strconv"
    "net/url"
    "wssgo/libs"
)

func request(reqParams *RequestData)(string) {
    var requestUrl string

    u, err := url.Parse(reqParams.RequestUrl)
    if err == nil {
        requestUrl = u.Scheme+"://"+u.Host+u.Path
    }
    if _, ok := UrlList[requestUrl]; !ok {
        fmt.Println(requestUrl)
        return "Illegal request, url is not allowed"
    }
    req := curl.NewRequest()
    resp, err := req.SetUrl(reqParams.RequestUrl).
        SetMethod(reqParams.HttpMethod).
        SetPostData(reqParams.PostData).
        SetHeaders(reqParams.Headers).
        Send()
    if err != nil {
        return  fmt.Sprintf("%s", err)
    } else {
        if resp.IsOk() {
            fmt.Println(2222)

            return  resp.Body
        } else {
            return  "httpStatus:"+ strconv.Itoa(resp.Raw.StatusCode)

        }
    }

}

func process(message []byte, wsConn *wsConnection)(error) {
    var wsReqData WsRequestData
    var wsReqParams map[string]interface{}
    var wsRespData ResponseData

    err := json.Unmarshal(message, &wsReqParams)
    if err != nil {
        //wsConn.wsWrite(1, []byte("传入参数json解析失败"))
        return errors.New("传入参数json解析失败")
    }

    cwerr := checkWsReqData(wsReqParams, &wsReqData, &wsRespData)
    if cwerr!=nil  {
        return cwerr
    }

    switch wsReqData.RequestType {
        case "req&resp":
            doRequestBusiness(wsReqData.RequestData, &wsRespData, wsConn)
        case "broadcast":
            doBroadcast(wsReqData.RequestData, &wsRespData, wsConn)
        default:
            wsConn.wsWrite(1, []byte("request_type: "+wsReqData.RequestType+" 非法"))
    }
    return nil
}

func checkWsReqData(wsReqParams map[string]interface{},  wsReqData *WsRequestData, wsRespData *ResponseData)(error){
    RequestType, ok := wsReqParams["request_type"].(string)
    if(ok){
        wsReqData.RequestType = RequestType
    }else{
        return errors.New("Lack of request_type")
    }

    RequestId, ok := wsReqParams["request_id"].(string)
    if(ok){
        wsReqData.RequestId  = RequestId
        wsRespData.RequestId = RequestId
    }else{
        return errors.New("Lack of request_id")
    }
    Wssid, ok := wsReqParams["wssid"].(string)
    if(ok){
        wsReqData.Wssid = Wssid
        wsRespData.Wssid = Wssid
    }else{
        return errors.New("Lack of wssid")
    }
    RequestData, ok := wsReqParams["request_data"].(map[string]interface{})
    if(ok){
        wsReqData.RequestData = RequestData
    }else{
        return errors.New("Lack of request_data")
    }
    Action, ok := wsReqParams["action"].(string)
    if(ok){
        wsReqData.Action = Action
        wsRespData.Action = Action
    }else{
        return errors.New("Lack of action")
    }
    return nil
}
func doRequestBusiness(reqParams map[string]interface{}, wsRespData *ResponseData, wsConn *wsConnection)(error) {
    //fmt.Println(reqParams)
    var reqData RequestData
    crerr := checkReqParams(reqParams, &reqData)
    if crerr!=nil  {
        wsRespData.ResponseData = fmt.Sprintf("%s", crerr)
    }else{
        respRs :=request(&reqData)
        wsRespData.ResponseData = respRs
    }
    display(wsRespData, wsConn)
    return  nil
}

func checkReqParams(reqParams map[string]interface{},  reqData *RequestData)(error){
    fmt.Println(reqParams)

    HttpMethod, ok := reqParams["http_method"].(string)
    if(ok){
        reqData.HttpMethod = HttpMethod
    }else{
        return errors.New("Lack of http_method")
    }

    RequestUrl, ok := reqParams["request_url"].(string)
    if(ok){
        reqData.RequestUrl = RequestUrl
    }else{
        return errors.New("Lack of request_url")
    }
    PostData, ok := reqParams["post_data"].(string)
    if(ok){
        reqData.PostData = PostData
    }else{
        return errors.New("Lack of post_data")
    }
    fmt.Println(reqParams["headers"])

    Headers, ok := reqParams["headers"].(map[string]interface{})
    if(ok){
        reqData.Headers = libs.MapInterfaceToMapString(Headers)
    }else{
        return errors.New("Lack of headers")

    }
    return nil
}

func display(wsRespData *ResponseData, wsConn *wsConnection){
    result , err := json.Marshal(wsRespData)
    if err != nil {
        result = []byte("返回值解析失败")
    }
    wsConn.wsWrite(1, result)
}

func doBroadcast(reqParams map[string]interface{}, wsRespData *ResponseData, wsConn *wsConnection)(error) {
    wssid, ok := reqParams["wssid"].(string)
    if(!ok){
        return errors.New("Lack of wssid")
    }

    message, ok := reqParams["message"].(string)
    if(!ok){
        return errors.New("Lack of message")
    }
    WsManager.doBroadcast(wssid, []byte(message), wssid)
    wsRespData.ResponseData = "信息：" + message+" 发送给 " + wssid
    display(wsRespData, wsConn)
    return nil
}