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

func request(reqParams *RequestData)(string, error) {
    var requestUrl string

    u, err := url.Parse(reqParams.RequestUrl)
    if err == nil {
        requestUrl = u.Scheme+"://"+u.Host+u.Path
    }
    if _, ok := UrlList[requestUrl]; !ok {
        fmt.Println(requestUrl)
        return "", errors.New("Illegal request, url is not allowed")
    }
    req := curl.NewRequest()
    resp, err := req.SetUrl(reqParams.RequestUrl).
        SetMethod(reqParams.HttpMethod).
        SetPostData(reqParams.PostData).
        SetHeaders(reqParams.Headers).
        Send()
    if err != nil {
        return "", err
    } else {
        if resp.IsOk() {
            return resp.Body, nil
        } else {
            return "", errors.New("httpStatus:"+ strconv.Itoa(resp.Raw.StatusCode))
        }
    }

}

func process(message []byte, wsConn *wsConnection)(error) {
    var wsReqData WsRequestData
    var wsReqParams map[string]interface{}
    var wsRespData ResponseData

    err := json.Unmarshal(message, &wsReqParams)
    if err != nil {
        resp := `{"errcode":4004, "wssid":"`+wsConn.wssid+`","request_id":"","response_data":" incoming parameter parsing failed， need json","action":"error"}`
        wsConn.wsWrite(1, []byte(resp))
        return nil
    }

    cwerr := checkWsReqData(wsReqParams, &wsReqData, &wsRespData)
    if cwerr!=nil  {
        wsRespData.ResponseData = fmt.Sprintf("%s", cwerr)
        wsRespData.Action="error"
        display(&wsRespData, wsConn)
        return nil
    }

    switch wsReqData.RequestType {
        case "req&resp":
            doRequestBusiness(wsReqData.RequestData, &wsRespData, wsConn)
        case "broadcast":
            doSendMsgToWssid(wsReqData.RequestData, &wsRespData, wsConn)
        default:
            wsConn.wsWrite(1, []byte("request_type: "+wsReqData.RequestType+" 非法"))
    }
    return nil
}

func checkWsReqData(wsReqParams map[string]interface{},  wsReqData *WsRequestData, wsRespData *ResponseData)(error){


    RequestId, ok := wsReqParams["request_id"].(string)
    if(ok){
        wsReqData.RequestId  = RequestId
        wsRespData.RequestId = RequestId
    }else{
        wsRespData.ErrorCode = 4201
        return errors.New("Lack of request_id")
    }
    Wssid, ok := wsReqParams["wssid"].(string)
    if(ok){
        wsReqData.Wssid = Wssid
        wsRespData.Wssid = Wssid
    }else{
        wsRespData.ErrorCode = 4202
        return errors.New("Lack of wssid")
    }

    RequestType, ok := wsReqParams["request_type"].(string)
    if(ok){
        wsReqData.RequestType = RequestType
    }else{
        wsRespData.ErrorCode = 4203
        return errors.New("Lack of request_type")
    }

    RequestData, ok := wsReqParams["request_data"].(map[string]interface{})
    if(ok){
        wsReqData.RequestData = RequestData
    }else{
        wsRespData.ErrorCode = 4204
        return errors.New("Lack of request_data")
    }
    Action, ok := wsReqParams["action"].(string)
    if(ok){
        wsReqData.Action = Action
        wsRespData.Action = Action
    }else{
        wsRespData.ErrorCode = 4205
        return errors.New("Lack of action")
    }
    return nil
}
func doRequestBusiness(reqParams map[string]interface{}, wsRespData *ResponseData, wsConn *wsConnection)(error) {
    var reqData RequestData
    errorCode, crerr := checkReqParams(reqParams, &reqData)
    if crerr!=nil  {
        wsRespData.ErrorCode=errorCode
        wsRespData.ResponseData = fmt.Sprintf("%s", crerr)
    }else{
        respRs, error:=request(&reqData)
        if error==nil{
            wsRespData.ResponseData = respRs
            wsRespData.ErrorCode=errorCode
        }else{
            wsRespData.ResponseData = fmt.Sprintf("%s", error)
            wsRespData.ErrorCode=5001
        }
    }
    display(wsRespData, wsConn)
    return  nil
}

func checkReqParams(reqParams map[string]interface{},  reqData *RequestData)(int, error){
    HttpMethod, ok := reqParams["http_method"].(string)
    if(ok){
        reqData.HttpMethod = HttpMethod
    }else{
        return 4101, errors.New("Lack of http_method")
    }

    RequestUrl, ok := reqParams["request_url"].(string)
    if(ok){
        reqData.RequestUrl = RequestUrl
    }else{
        return 4102, errors.New("Lack of request_url")
    }
    PostData, ok := reqParams["post_data"].(string)
    if(ok){
        reqData.PostData = PostData
    }else{
        return 4103, errors.New("Lack of post_data")
    }
    Headers, ok := reqParams["headers"].(map[string]interface{})
    if(ok){
        reqData.Headers = libs.MapInterfaceToMapString(Headers)
    }else{
        return 4104, errors.New("Lack of headers")

    }
    return 200, nil
}

func display(wsRespData *ResponseData, wsConn *wsConnection){
    result , err := json.Marshal(wsRespData)
    if err != nil {
        result = []byte("返回值解析失败")
    }
    wsConn.wsWrite(1, result)
}

func doSendMsgToWssid(reqParams map[string]interface{}, wsRespData *ResponseData, wsConn *wsConnection)(error) {
    wssid, ok := reqParams["wssid"].(string)
    if(!ok){
        return errors.New("Lack of wssid")
    }

    message, ok := reqParams["message"].(string)
    if(!ok){
        return errors.New("Lack of message")
    }
    WsManager.doSendMsgToWssid(wssid, []byte(message))
    wsRespData.ResponseData = "信息：" + message+" 发送给 " + wssid
    display(wsRespData, wsConn)
    return nil
}