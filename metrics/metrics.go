package metrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/minhgiang16983/Minh-Kit-Hehe/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var (
	// Http client metrics
	HttpClientMetricsRequest = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "hsk_http_client_requests_total",
		Help: "Total number of HTTP client requests",
	}, []string{"method", "host", "path", "status"})

	HttpClientMetricsLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "hsk_http_client_request_duration_seconds",
		Help:    "Duration of HTTP client requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "host", "path"},
	)
)

func init() {
}

func RecordHttpClientMetricsRequest(method, host, path string, status string) {
	HttpClientMetricsRequest.WithLabelValues(method, host, path, status).Inc()
}

func RecordHttpClientMetricsLatency(method, host, path string, d time.Duration) {
	HttpClientMetricsLatency.WithLabelValues(method, host, path).Observe(d.Seconds())
}

// Metrics holds all the Prometheus metrics
type Metrics struct {
	// HTTP metrics
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	httpRequestsInFlight *prometheus.GaugeVec

	// Service metrics
	serviceUp *prometheus.GaugeVec

	// Custom metrics
	customMetrics map[string]prometheus.Collector
	config        *MetricsConfig
}

// MetricsConfig holds configuration for metrics
type MetricsConfig struct {
	Namespace   string `mapstructure:"namespace" yaml:"namespace"`       //optinal
	Subsystem   string `mapstructure:"subsystem" yaml:"subsystem"`       //optional
	ServiceName string `mapstructure:"service_name" yaml:"service_name"` //optional
	Port        int    `mapstructure:"port" yaml:"port"`
	Path        string `mapstructure:"path" yaml:"path"`
}

// DefaultConfig returns default metrics configuration
func DefaultConfig() *MetricsConfig {
	return &MetricsConfig{
		Namespace:   "",
		Subsystem:   "",
		ServiceName: "",
		Port:        9090,
		Path:        "/metrics",
	}
}

// New creates a new Metrics instance
func New(config *MetricsConfig) *Metrics {
	if config == nil {
		config = DefaultConfig()
	}

	m := &Metrics{
		customMetrics: make(map[string]prometheus.Collector),
		config:        config,
	}

	// HTTP metrics
	m.httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code", "base_service"},
	)

	m.httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "base_service"},
	)

	m.httpRequestsInFlight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "http_requests_in_flight",
			Help:      "Current number of HTTP requests being processed",
		},
		[]string{"base_service"},
	)

	// Service metrics
	m.serviceUp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "service_up",
			Help:      "Indicates if the base_service is running",
		},
		[]string{"base_service"},
	)

	// Register all metrics
	prometheus.MustRegister(
		m.httpRequestsTotal,
		m.httpRequestDuration,
		m.httpRequestsInFlight,
		m.serviceUp,
	)

	// Initialize base_service up metric to 0 (down) by default
	m.SetServiceUp(false)

	return m
}

// HTTPMiddleware creates middleware for HTTP metrics
func (m *Metrics) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Increment in-flight requests
			m.httpRequestsInFlight.WithLabelValues(m.config.ServiceName).Inc()
			defer m.httpRequestsInFlight.WithLabelValues(m.config.ServiceName).Dec()

			// Create a response writer wrapper to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			// Record metrics
			duration := time.Since(start).Seconds()
			statusCode := strconv.Itoa(wrapped.statusCode)

			url := r.Pattern
			if url == "" {
				url = r.URL.Path
			}
			m.httpRequestsTotal.WithLabelValues(r.Method, url, statusCode, m.config.ServiceName).Inc()
			m.httpRequestDuration.WithLabelValues(r.Method, url, m.config.ServiceName).Observe(duration)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// AddCustomMetric adds a custom metric
func (m *Metrics) AddCustomMetric(name string, metric prometheus.Collector) error {
	if _, exists := m.customMetrics[name]; exists {
		return fmt.Errorf("metric with name %s already exists", name)
	}

	m.customMetrics[name] = metric
	prometheus.MustRegister(metric)
	return nil
}

// RemoveCustomMetric removes a custom metric
func (m *Metrics) RemoveCustomMetric(name string) error {
	metric, exists := m.customMetrics[name]
	if !exists {
		return fmt.Errorf("metric with name %s does not exist", name)
	}

	prometheus.Unregister(metric)
	delete(m.customMetrics, name)
	return nil
}

// SetServiceUp sets the base_service up status
func (m *Metrics) SetServiceUp(up bool) {
	value := 0.0
	if up {
		value = 1.0
	}
	m.serviceUp.WithLabelValues(m.config.ServiceName).Set(value)
}

// StartMetricsServer starts the metrics HTTP server
func (m *Metrics) StartMetricsServer(l logger.LoggerInterface) error {

	http.Handle(m.config.Path, promhttp.Handler())

	addr := fmt.Sprintf(":%d", m.config.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: http.DefaultServeMux,
	}

	// Set base_service as up only once when server starts
	m.SetServiceUp(true)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.Error("Metrics server error", l.ErrorField(err))
		}
	}()

	l.Info("Metrics server started", l.String("address", addr), l.String("path", m.config.Path))
	return nil
}

// StartMetricsServerWithContext starts the metrics HTTP server with context for graceful shutdown
func (m *Metrics) StartMetricsServerWithContext(ctx context.Context, config *MetricsConfig, l logger.LoggerInterface) error {
	if config == nil {
		config = DefaultConfig()
	}

	http.Handle(config.Path, promhttp.Handler())

	addr := fmt.Sprintf(":%d", config.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: http.DefaultServeMux,
	}

	// Set base_service as up only once when server starts
	m.SetServiceUp(true)

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			l.Error("Metrics server error", zap.Error(err))
		}
	}()

	l.Info("Metrics server started", zap.String("address", addr), zap.String("path", config.Path))

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("metrics server shutdown error: %v", err)
	}

	l.Info("Metrics server stopped gracefully")
	return nil
}

func (m *Metrics) GetConfig() *MetricsConfig {
	return m.config
}
