package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// DMSGMetrics record dmsg metrics
type DMSGMetrics struct {
	ClientConns prometheus.Gauge
	Bandwidth   prometheus.Summary
}

// NewDMSGMetrics construct new DMSGMetrics.
func NewDMSGMetrics(service string) *DMSGMetrics {
	return &DMSGMetrics{
		ClientConns: promauto.NewGauge(prometheus.GaugeOpts{
			Name: service + "_clients_total",
			Help: "The total number of connected clients",
		}),
		Bandwidth: promauto.NewSummary(prometheus.SummaryOpts{
			Name: service + "_bandwidth",
			Help: "Amount of bytes proxied between clients",
		}),
	}
}
