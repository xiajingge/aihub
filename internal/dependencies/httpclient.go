package dependencies

import (
	"net/http"
	"time"

	"github.com/xiajignge/aihub/pkg/httpclient"
)

func NewHttpClient() *httpclient.HttpClient {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   20,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	return httpclient.NewHttpClientWithClient(client)
}
