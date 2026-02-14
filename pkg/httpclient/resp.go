package httpclient

import (
	"io"
	"net/http"
)

type Response struct {
	// HTTP response basics
	StatusCode int `json:"status_code"`

	// Response headers
	Headers http.Header `json:"headers"`

	// Response body, for the non-streaming response.
	Body []byte `json:"body,omitempty"`

	// Streaming support
	Stream io.ReadCloser `json:"-"`

	// Request information
	Request *Request `json:"-"`

	// Raw HTTP response for advanced use cases
	RawResponse *http.Response `json:"-"`

	// Raw HTTP request for advanced use cases
	RawRequest *http.Request `json:"-"`
}

type Resp[T any] struct {
	Code    int    `json:"code"`           // 业务状态码
	Message string `json:"message"`        // 提示信息
	Data    T      `json:"data,omitempty"` // 返回数据
}

func Success[T any](data T) Resp[T] {
	return Resp[T]{
		Code:    0,
		Message: "success",
		Data:    data,
	}
}

func Fail[T any](msg string, data T) Resp[T] {
	return Resp[T]{
		Code:    1,
		Message: msg,
		Data:    data,
	}
}
