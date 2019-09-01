package meduhttp

import (
	"net"
	"net/http"
	"time"
)

type HTTPClient struct {
	*http.Client
}

type Options struct {
	Timeout time.Duration
}

type Response struct {
	URL        string
	StatusCode int
	Body       []byte
}

func New(options *Options) *HTTPClient {

	transport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   options.Timeout,
			KeepAlive: 1 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 60 * time.Second,
	}

	return &HTTPClient{
		&http.Client{Transport: transport},
	}
}
