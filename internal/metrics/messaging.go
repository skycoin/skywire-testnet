package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MessagingMetrics record messaging metrics
type MessagingMetrics struct {
	ClientConns prometheus.Gauge
	Bandwidth   prometheus.Summary
}

// NewMessagingMetrics construct new MessagingMetrics.
func NewMessagingMetrics(service string) *MessagingMetrics {
	return &MessagingMetrics{
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
