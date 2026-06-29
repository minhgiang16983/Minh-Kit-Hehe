package cron

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/minhgiang16983/Minh-Kit-Hehe/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

func TestCron(t *testing.T) {
	metrics := metrics.New(metrics.DefaultConfig())
	cron, err := NewCron(metrics)
	if err != nil {
		t.Fatalf("Failed to create cron: %v", err)
	}

	err = cron.AddCronJob("test", "0/2 * * * * *", func() {
		now := time.Now()

		fmt.Printf("hello world %s\n", now.Format(time.RFC3339))
		time.Sleep(1 * time.Second)
	})
	if err != nil {
		t.Fatalf("Failed to add cron job: %v", err)
	}

	time.Sleep(10 * time.Second)

	err = cron.Shutdown()
	if err != nil {
		t.Fatalf("Failed to shutdown cron: %v", err)
	}

	// print all metrics
	m, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// print all metrics
	for _, metric := range m {
		_, err := expfmt.MetricFamilyToText(os.Stdout, metric)
		if err != nil {
			t.Fatalf("Failed to print metric: %v", err)
		}
	}
}
