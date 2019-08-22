package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// DmsgMetrics record dmsg metrics
type DmsgMetrics struct {
	ClientConns prometheus.Gauge
	Bandwidth   prometheus.Summary
}

// NewDmsgMetrics construct new DmsgMetrics.
func NewDmsgMetrics(service string) *DmsgMetrics {
	return &DmsgMetrics{
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
