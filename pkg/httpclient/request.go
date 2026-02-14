package httpclient

import (
	"fmt"
	"io"
	"net/http"
)

type Request struct {
	// HTTP basics
	Method  string      `json:"method"`
	URL     string      `json:"url"`
	Headers http.Header `json:"headers"`
	Body    []byte      `json:"body,omitempty"`

	// Authentication
	Auth *AuthConfig `json:"auth,omitempty"`

	// Request tracking
	RequestID string `json:"request_id"`

	// Raw HTTP request for advanced use cases
	RawRequest *http.Request `json:"-"`
}

func ReadHTTPRequest(rawReq *http.Request) (*Request, error) {
	req := &Request{
		Method:     rawReq.Method,
		URL:        rawReq.URL.String(),
		Headers:    rawReq.Header,
		Body:       []byte{},
		Auth:       &AuthConfig{},
		RequestID:  "",
		RawRequest: rawReq,
	}

	body, err := io.ReadAll(rawReq.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	req.Body = body

	return req, nil
}

type AuthConfig struct {
	// Type represents the type of authentication.
	// "bearer", "api_key"
	Type string `json:"type"`

	// APIKey is the API key for the request.
	APIKey string `json:"api_key,omitempty"`

	// HeaderKey is the header key for the request if the type is "api_key".
	HeaderKey string `json:"header_key,omitempty"`
}
