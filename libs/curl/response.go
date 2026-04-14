package curl

import (
	"io"
	"net/http"
)

type Response struct {
	Raw     *http.Response
	Headers map[string]string
	Body    string
}

func NewResponse() *Response {
	return &Response{}
}

func (this *Response) IsOk() bool {
	return this.Raw.StatusCode == 200
}

func (this *Response) parseHeaders() {
	headers := map[string]string{}
	for k, v := range this.Raw.Header {
		headers[k] = v[0]
	}
	this.Headers = headers
}

func (this *Response) parseBody() error {
	body, err := io.ReadAll(this.Raw.Body)
	if err != nil {
		return err
	}
	this.Body = string(body)
	return nil
}
