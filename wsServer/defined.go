package wsServer

//ws 输入参数类型
type WsRequestData struct {
    Wssid string  `json:"wssid"` //websocket id 服务端生成下发
    RequestId string  `json:"request_id"` //客户端请求的唯一的ID， 每次请求生成一个uuid
    RequestData map[string]interface{} `json:"request_data` //请求给业务的具体数据， 根据request_type类型有所变化
    RequestType string `json:"request_type"` //请求的类型， 目前支持req&resp
    Action string `json:"action"`  //请求动作
}

type RequestData struct {
    HttpMethod string `json:"http_method"`  //请求方式目前支持GET， POST
    RequestUrl string  `json:"request_url"`  //请求地址，GET方法的话参数拼后面， 如：http://example.mafengwo.cn/test.php?id=666
    PostData string `json:"post_data"`  //post数据的json串， key值约定为：params
    Headers map[string]string `json:"headers`
}

type ResponseData struct {
    Wssid string  `json:"wssid"` //websocket id 服务端生成下发
    RequestId string  `json:"request_id"` //客户端请求的唯一的ID， 每次请求生成一个uuid
    ResponseData string `json:"response_data"` //返回值
    Action string `json:"action"`  //请求动作
    ErrorCode int `json:"errcode"`
}