package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newChannel3Request(serverBaseURL string) *Request {
	return &Request{
		Method: http.MethodPost,
		URL:    serverBaseURL + "/v1/chat/completions",
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: []byte(`{"model":"qwen3-max","messages":[{"role":"user","content":"hello"}]}`),
		Auth: &AuthConfig{
			Type:   "bearer",
			APIKey: "sk_123456789",
		},
	}
}

func TestHttpClient_Do(t *testing.T) {
	// 测试server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("unexpected content type: %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"ok","mode":"non-stream"}`))
	}))
	defer server.Close()

	client := NewHttpClient()
	resp, err := client.Do(context.Background(), newChannel3Request(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("resp: %+v\n", resp)
	fmt.Printf("resp.body: %+v\n", string(resp.Body))
}

func TestHttpClient_DoStream(t *testing.T) {
	// 测试server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("unexpected content type: %s", got)
		}

		if flusher, ok := w.(http.Flusher); ok {
			_, _ = w.Write([]byte("data: part-1\n\n"))
			flusher.Flush()
			_, _ = w.Write([]byte("data: part-2\n\n"))
			flusher.Flush()
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			flusher.Flush()
			return
		}

	}))
	defer server.Close()

	client := NewHttpClient()
	resp, err := client.DoStream(context.Background(), newChannel3Request(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	err = consumeStream(resp)
	fmt.Printf("err: %+v\n", err)
}

func TestHttpClient_DoStream2(t *testing.T) {
	// 测试server

	client := NewHttpClient()
	body := `{"model":"qwen3-max","messages":[{"role":"user","content":"hello"}],"stream":true}`
	resp, err := client.DoStream(context.Background(), &Request{
		Method: http.MethodPost,
		URL:    "https://apis.iflow.cn/v1/chat/completions",
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: []byte(body),
		Auth: &AuthConfig{
			Type:   "bearer",
			APIKey: "sk-a87ab03fc02b704c336b7ce1cb572588",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = consumeStream(resp)
	fmt.Printf("err: %+v\n", err)
}

func consumeStream(stream Stream[*StreamEvent]) error {
	if stream == nil {
		return fmt.Errorf("stream is nil")
	}
	defer func() { _ = stream.Close() }()

	for stream.Next() {
		ev := stream.Current()
		if ev == nil {
			continue
		}

		// ev.Type: 事件类型
		// ev.Data: 事件内容（通常是 JSON 字节）
		fmt.Printf("event=%s data=%s\n", ev.Type, string(ev.Data))
	}

	// Next() 结束后一定要检查 Err()
	if err := stream.Err(); err != nil {
		return err
	}
	return nil
}
