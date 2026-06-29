package kafka

import (
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rcrowley/go-metrics"
)

var (
	globalMu          sync.Mutex
	promMetrics       = make(map[string]any)
	defaultRegisterer = prometheus.DefaultRegisterer
)

type PromRegistry struct {
	mu         sync.RWMutex
	registered map[string]any
}

func NewPromRegistry() *PromRegistry {
	return &PromRegistry{
		registered: make(map[string]any),
	}
}

func (r *PromRegistry) GetOrRegister(name string, i any) any {
	r.mu.RLock()
	if metric, exists := r.registered[name]; exists {
		r.mu.RUnlock()
		return metric
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double check
	if metric, exists := r.registered[name]; exists {
		return metric
	}

	actual := i
	if v, ok := i.(func() metrics.Counter); ok {
		actual = v()
	} else if v, ok := i.(func() metrics.Gauge); ok {
		actual = v()
	} else if v, ok := i.(func() metrics.GaugeFloat64); ok {
		actual = v()
	} else if v, ok := i.(func() metrics.Meter); ok {
		actual = v()
	} else if v, ok := i.(func() metrics.Histogram); ok {
		actual = v()
	} else if v, ok := i.(func() metrics.Timer); ok {
		actual = v()
	}

	promName := sanitizeMetricName(name)

	globalMu.Lock()
	if metric, exists := promMetrics[promName]; exists {
		globalMu.Unlock()
		r.registered[name] = metric
		return metric
	}

	var adapter any
	switch actual.(type) {
	case metrics.Counter:
		promGauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: promName})
		if err := defaultRegisterer.Register(promGauge); err != nil {
			if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
				promGauge = are.ExistingCollector.(prometheus.Gauge)
			}
		}
		adapter = &PromCounterAdapter{promGauge}

	case metrics.Gauge:
		promGauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: promName})
		if err := defaultRegisterer.Register(promGauge); err != nil {
			if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
				promGauge = are.ExistingCollector.(prometheus.Gauge)
			}
		}
		adapter = &PromGaugeAdapter{promGauge}

	case metrics.Meter:
		promGauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: promName})
		if err := defaultRegisterer.Register(promGauge); err != nil {
			if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
				promGauge = are.ExistingCollector.(prometheus.Gauge)
			}
		}
		adapter = &PromMeterAdapter{promGauge}

	case metrics.Histogram:
		promSummary := prometheus.NewSummary(prometheus.SummaryOpts{
			Name:       promName,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		})
		if err := defaultRegisterer.Register(promSummary); err != nil {
			if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
				promSummary = are.ExistingCollector.(prometheus.Summary)
			}
		}
		adapter = &PromHistogramAdapter{promSummary}
	}

	if adapter != nil {
		promMetrics[promName] = adapter
		r.registered[name] = adapter
		globalMu.Unlock()
		return adapter
	}
	globalMu.Unlock()

	return i
}

func (r *PromRegistry) Register(name string, i any) error {
	r.GetOrRegister(name, i)
	return nil
}

func (r *PromRegistry) Get(name string) any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.registered[name]
}

func (r *PromRegistry) Each(func(string, any))            {}
func (r *PromRegistry) GetAll() map[string]map[string]any { return nil }
func (r *PromRegistry) RunHealthchecks()                  {}
func (r *PromRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.registered, name)
}
func (r *PromRegistry) UnregisterAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registered = make(map[string]any)
}

func sanitizeMetricName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

// --- Adapters ---

type PromCounterAdapter struct {
	prometheus.Gauge
}

func (a *PromCounterAdapter) Clear()                    { a.Set(0) }
func (a *PromCounterAdapter) Count() int64              { return 0 }
func (a *PromCounterAdapter) Snapshot() metrics.Counter { return a }
func (a *PromCounterAdapter) Dec(v int64)               { a.Sub(float64(v)) }
func (a *PromCounterAdapter) Inc(v int64)               { a.Add(float64(v)) }

type PromGaugeAdapter struct {
	prometheus.Gauge
}

func (a *PromGaugeAdapter) Snapshot() metrics.Gauge { return a }
func (a *PromGaugeAdapter) Update(v int64)          { a.Set(float64(v)) }
func (a *PromGaugeAdapter) Value() int64            { return 0 }

type PromMeterAdapter struct {
	prometheus.Gauge
}

func (a *PromMeterAdapter) Count() int64            { return 0 }
func (a *PromMeterAdapter) Mark(v int64)            { a.Add(float64(v)) }
func (a *PromMeterAdapter) Rate1() float64          { return 0 }
func (a *PromMeterAdapter) Rate5() float64          { return 0 }
func (a *PromMeterAdapter) Rate15() float64         { return 0 }
func (a *PromMeterAdapter) RateMean() float64       { return 0 }
func (a *PromMeterAdapter) Stop()                   {}
func (a *PromMeterAdapter) Snapshot() metrics.Meter { return a }

type PromHistogramAdapter struct {
	prometheus.Summary
}

func (a *PromHistogramAdapter) Clear()                          {}
func (a *PromHistogramAdapter) Count() int64                    { return 0 }
func (a *PromHistogramAdapter) Max() int64                      { return 0 }
func (a *PromHistogramAdapter) Mean() float64                   { return 0 }
func (a *PromHistogramAdapter) Min() int64                      { return 0 }
func (a *PromHistogramAdapter) Percentile(float64) float64      { return 0 }
func (a *PromHistogramAdapter) Percentiles([]float64) []float64 { return nil }
func (a *PromHistogramAdapter) Sample() metrics.Sample          { return nil }
func (a *PromHistogramAdapter) Snapshot() metrics.Histogram     { return a }
func (a *PromHistogramAdapter) StdDev() float64                 { return 0 }
func (a *PromHistogramAdapter) Sum() int64                      { return 0 }
func (a *PromHistogramAdapter) Update(v int64)                  { a.Observe(float64(v)) }
func (a *PromHistogramAdapter) Variance() float64               { return 0 }
