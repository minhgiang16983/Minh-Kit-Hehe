package transport

import (
	"github.com/minhgiang16983/Minh-Kit-Hehe/metrics"
	"net/http"
	"strconv"
	"time"
)

type HttpRoundTripper struct {
	Base http.RoundTripper
}

func (h *HttpRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	res, err := h.Base.RoundTrip(req)

	duration := time.Since(start)

	status := 500
	if res != nil {
		status = res.StatusCode
	}

	host := req.Host
	if host == "" {
		host = req.URL.Host
	}
	metrics.RecordHttpClientMetricsRequest(req.Method, host, req.URL.Path, strconv.Itoa(status))
	metrics.RecordHttpClientMetricsLatency(req.Method, host, req.URL.Path, duration)

	return res, err
}
