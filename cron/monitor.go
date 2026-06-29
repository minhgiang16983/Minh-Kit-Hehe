package cron

import (
	"context"
	"time"

	"github.com/minhgiang16983/Minh-Kit-Hehe/logger"
	"github.com/minhgiang16983/Minh-Kit-Hehe/metrics"
	gocron "github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

type Monitor struct {
	logger  logger.LoggerInterface
	metrics *metrics.Metrics

	// Prometheus metrics for cron jobs
	jobExecutionsTotal   *prometheus.CounterVec
	jobDurationHistogram *prometheus.HistogramVec
}

// IncrementJob implements gocron.MonitorStatus.
func (m *Monitor) IncrementJob(id uuid.UUID, name string, tags []string, status gocron.JobStatus) {
	// Record Prometheus metrics
	if m.jobExecutionsTotal != nil {
		// Increment job execution counter by status
		m.jobExecutionsTotal.WithLabelValues(name, string(status)).Inc()
	}
}

// RecordJobTiming implements gocron.MonitorStatus.
func (m *Monitor) RecordJobTiming(startTime time.Time, endTime time.Time, id uuid.UUID, name string, tags []string) {
	// do nothing, we will use RecordJobTimingWithStatus instead
}

// RecordJobTimingWithStatus implements gocron.MonitorStatus.
func (m *Monitor) RecordJobTimingWithStatus(startTime time.Time, endTime time.Time, id uuid.UUID, name string, tags []string, status gocron.JobStatus, err error) {
	duration := endTime.Sub(startTime)

	// Log only errors
	if err != nil {
		m.logger.Error("Job failed with error",
			m.logger.String("job_id", id.String()),
			m.logger.String("job_name", name),
			m.logger.String("status", string(status)),
			m.logger.Duration("duration", duration),
			m.logger.String("error", err.Error()),
		)
	}

	// Record Prometheus metrics
	if m.jobDurationHistogram != nil {
		// Record job duration histogram with status
		m.jobDurationHistogram.WithLabelValues(name, string(status)).Observe(duration.Seconds())
	}

}

func NewMonitor() gocron.MonitorStatus {
	return &Monitor{
		logger:  logger.GetLogger(context.Background()),
		metrics: nil,
	}
}

// NewMonitorWithMetrics creates a new monitor with metrics support
func NewMonitorWithMetrics(metrics *metrics.Metrics) gocron.MonitorStatus {
	monitor := &Monitor{
		logger:  logger.GetLogger(context.Background()),
		metrics: metrics,
	}

	// Initialize Prometheus metrics
	monitor.initializePrometheusMetrics()

	return monitor
}

// initializePrometheusMetrics creates and registers Prometheus metrics for cron jobs
func (m *Monitor) initializePrometheusMetrics() {

	config := m.metrics.GetConfig()
	// Job executions counter
	m.jobExecutionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "cron",
			Name:      "job_executions_total",
			Help:      "Total number of cron job executions by status",
		},
		[]string{"job_name", "status"},
	)

	// Job duration histogram
	m.jobDurationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: "cron",
			Name:      "job_duration_seconds",
			Help:      "Duration of cron job executions in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"job_name", "status"},
	)

	// Register metrics with Prometheus
	prometheus.MustRegister(
		m.jobExecutionsTotal,
		m.jobDurationHistogram,
	)
}
