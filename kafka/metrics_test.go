package kafka

import (
	"testing"

	"github.com/rcrowley/go-metrics"
)

func TestPromRegistry_GetOrRegister_Factory(t *testing.T) {
	registry := NewPromRegistry()

	// Giả lập cách go-metrics gọi: truyền vào một factory function
	name := "test_meter"
	factory := func() metrics.Meter {
		return metrics.NewMeter()
	}

	result := registry.GetOrRegister(name, factory)

	// Kiểm tra xem kết quả có thể ép kiểu sang metrics.Meter không
	if _, ok := result.(metrics.Meter); !ok {
		t.Errorf("GetOrRegister returned %T, expected metrics.Meter", result)
	}

	// Kiểm tra xem nó có đúng là adapter của chúng ta không
	if _, ok := result.(*PromMeterAdapter); !ok {
		t.Errorf("Expected *PromMeterAdapter, got %T", result)
	}
}

func TestPromRegistry_GetOrRegister_HistogramFactory(t *testing.T) {
	registry := NewPromRegistry()

	name := "test_histogram"
	factory := func() metrics.Histogram {
		return metrics.NewHistogram(metrics.NewSampleSnapshot(0, nil))
	}

	result := registry.GetOrRegister(name, factory)

	if _, ok := result.(metrics.Histogram); !ok {
		t.Errorf("GetOrRegister returned %T, expected metrics.Histogram", result)
	}
}
