package curl

import (
	"net"
	"net/http"
	"strings"
	"time"
)

// 全局共享 Transport，复用 TCP 连接
var sharedTransport = &http.Transport{
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	MaxIdleConns:          100,
	MaxIdleConnsPerHost:   10,
	IdleConnTimeout:       90 * time.Second,
	ResponseHeaderTimeout: 30 * time.Second,
}

// Request 构造类
type Request struct {
	Method          string
	Url             string
	dialTimeout     time.Duration
	responseTimeOut time.Duration
	Headers         map[string]string
	PostData        string
}

// 创建一个 Request 实例
func NewRequest() *Request {
	r := &Request{}
	r.dialTimeout = 5 * time.Second
	r.responseTimeOut = 5 * time.Second
	return r
}

// SetDialTimeOut 单位：秒
func (this *Request) SetDialTimeOut(TimeOutSecond int) *Request {
	this.dialTimeout = time.Duration(TimeOutSecond) * time.Second
	return this
}

// SetResponseTimeOut 单位：秒
func (this *Request) SetResponseTimeOut(TimeOutSecond int) *Request {
	this.responseTimeOut = time.Duration(TimeOutSecond) * time.Second
	return this
}

// SetMethod 设置请求方法
func (this *Request) SetMethod(method string) *Request {
	this.Method = method
	return this
}

// SetUrl 设置请求地址
func (this *Request) SetUrl(url string) *Request {
	this.Url = url
	return this
}

// SetHeaders 设置请求头
func (this *Request) SetHeaders(headers map[string]string) *Request {
	this.Headers = headers
	return this
}

// SetPostData 设置 POST 数据
func (this *Request) SetPostData(postData string) *Request {
	this.PostData = postData
	return this
}

func (this *Request) Send() (*Response, error) {
	// 使用全局共享 Transport，每次请求只调整超时
	transport := sharedTransport.Clone()
	transport.ResponseHeaderTimeout = this.responseTimeOut

	client := &http.Client{
		Transport: transport,
		Timeout:   this.dialTimeout + this.responseTimeOut,
	}

	req, err := http.NewRequest(this.Method, this.Url, strings.NewReader(this.PostData))
	if err != nil {
		return nil, err
	}

	if this.Method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	for k, v := range this.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	response := NewResponse()
	response.Raw = resp
	defer response.Raw.Body.Close()

	response.parseHeaders()
	response.parseBody()

	return response, nil
}
