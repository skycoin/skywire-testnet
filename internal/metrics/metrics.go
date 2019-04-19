package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Recorder records request metrics.
type Recorder interface {
	Record(resTime time.Duration, hasErr bool)
}

type dummy struct{}

// NewDummy constructs a new dummy metrics recorder.
func NewDummy() Recorder {
	return &dummy{}
}

func (m *dummy) Record(resTime time.Duration, hasErr bool) {}

type prom struct {
	reqCount prometheus.Counter
	errCount prometheus.Counter
	resTime  prometheus.Summary
}

// NewPrometheus constructs a new Prometheus metrics recorder.
func NewPrometheus(service string) Recorder {
	return &prom{
		reqCount: promauto.NewCounter(prometheus.CounterOpts{
			Name: service + "_request_total",
			Help: "The total number of processed requests",
		}),
		errCount: promauto.NewCounter(prometheus.CounterOpts{
			Name: service + "_errors_total",
			Help: "The total number of 500 responses",
		}),
		resTime: promauto.NewSummary(prometheus.SummaryOpts{
			Name: service + "_response_time",
			Help: "Response times",
		}),
	}
}

func (m *prom) Record(resTime time.Duration, hasErr bool) {
	m.reqCount.Inc()
	m.resTime.Observe(resTime.Seconds())
	if hasErr {
		m.errCount.Inc()
	}
}

// Handler provides metrics middleware.
func Handler(m Recorder, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if m == nil {
			next.ServeHTTP(w, req)
			return
		}

		wrapW := &wrapResponseWriter{ResponseWriter: w}
		startTime := time.Now()
		next.ServeHTTP(wrapW, req)
		m.Record(time.Since(startTime), wrapW.statusCode == http.StatusInternalServerError)
	})
}

type wrapResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *wrapResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
