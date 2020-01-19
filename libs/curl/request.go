package curl

import (
    "net"
    "net/http"
    "time"
    "strings"
    "fmt"
)

// Request构造类
type Request struct {
    Method          string
    Url             string
    dialTimeout     time.Duration
    responseTimeOut time.Duration
    Headers         map[string]string
    PostData        string
}

// 创建一个Request实例
func  NewRequest() *Request {
    r := &Request{}
    r.dialTimeout = 5
    r.responseTimeOut = 5
    return r
}

//SetDialTimeOut
func (this *Request) SetDialTimeOut(TimeOutSecond int) *Request{
    this.dialTimeout = time.Duration(TimeOutSecond)
    return this
}

//SetResponseTimeOut
func (this *Request) SetResponseTimeOut(TimeOutSecond int) *Request{
    this.responseTimeOut = time.Duration(TimeOutSecond)
    return this
}

// 设置请求方法
func (this *Request) SetMethod(method string) *Request {
    this.Method = method
    return this
}

// 设置请求地址
func (this *Request) SetUrl(url string) *Request {
    this.Url = url
    return this
}

// 设置请求头
func (this *Request) SetHeaders(headers map[string]string) *Request {
    this.Headers = headers
    return this
}

// 设置请求头
func (this *Request) SetPostData(postData string) *Request {
    this.PostData = postData
    return this
}

func (this *Request) Send() (*Response, error){

    // 初始化Response对象
    response := NewResponse()

    client := &http.Client{
        Transport: &http.Transport{
            Dial: func(netw, addr string) (net.Conn, error) {
                conn, err := net.DialTimeout(netw, addr, time.Second*this.dialTimeout)
                if err != nil {
                    return nil, err
                }
                conn.SetDeadline(time.Now().Add(time.Second * this.dialTimeout))
                return conn, nil
            },
            ResponseHeaderTimeout: time.Second * this.responseTimeOut,
        },
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


    if resp, err := client.Do(req); err != nil {
        fmt.Println(err)

        return nil, err
    } else {
        response.Raw = resp
    }

    defer response.Raw.Body.Close()

    response.parseHeaders()
    response.parseBody()

    return response, nil
}

