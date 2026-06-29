package http

import (
	"net/http"
	"time"

	httpRoundTripper "github.com/minhgiang16983/Minh-Kit-Hehe/client/http/transport"

	"github.com/minhgiang16983/Minh-Kit-Hehe/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type HttpClientConfig struct {
	Transport http.RoundTripper
	Timeout   time.Duration
}

func NewHttpClient(config *HttpClientConfig) (*http.Client, error) {
	client := &http.Client{
		Timeout: config.Timeout,
	}

	if config.Transport == nil {
		config.Transport = http.DefaultTransport
	}

	if tracing.GetGlobalTracingConfig().IsEnableHTTPTracing() {
		client.Transport = otelhttp.NewTransport(&httpRoundTripper.HttpRoundTripper{
			Base: config.Transport,
		})
	}

	return client, nil
}
