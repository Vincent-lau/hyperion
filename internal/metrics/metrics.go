package metrics

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	/* scheduling metrics */
	ConsensusLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "hyperion_consensus_latency",
		Help: "Latency of consensus in microseconds",
		// Buckets: prometheus.LinearBuckets(20000, 2000, 10),
		NativeHistogramBucketFactor: 1.1,
	})

	XchgLatencyPerIter = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "hyperion_xchg_latency_per_iter",
		Help: "Latency of exchange in microseconds per iteration",
		// Buckets: prometheus.LinearBuckets(20000, 2000, 10),
		NativeHistogramBucketFactor: 1.1,
	})
	CompLatencyPerIter = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "hyperion_comp_latency_per_iter",
		Help: "Latency of computation in microseconds per iteration",
		// Buckets: prometheus.LinearBuckets(600, 200, 10),
		NativeHistogramBucketFactor: 1.1,
	})
	TotTimePerIter = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "hyperion_tot_time_per_iter",
		Help: "Total time in microseconds per iteration, i.e. sum of xchg and comp",
		// Buckets: prometheus.LinearBuckets(20000, 2000, 10),
		NativeHistogramBucketFactor: 1.1,
	})

	/* placement metrics */
	PlacementLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:                        "hyperion_placement_latency",
		Help:                        "Latency of placement in microseconds",
		NativeHistogramBucketFactor: 1.1,
	})

	// Create a new Prometheus gauge metric
	Gauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "my_metric",
		Help: "My custom metric",
	})

	metricsList = []prometheus.Collector{
		ConsensusLatency,
		XchgLatencyPerIter,
		CompLatencyPerIter,
		TotTimePerIter,

		PlacementLatency,
		Gauge,
	}
)

var registerMetrics sync.Once

func Register() {
	registerMetrics.Do(func() {
		prometheus.MustRegister(metricsList...)
	})
}

func Start() {
	Register()

	// Set the value of the metric
	Gauge.Set(42)

	// Create an HTTP handler to expose the metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	log.WithFields(log.Fields{
		"port":     8080,
		"endpoint": "/metrics",
	}).Info("Starting metrics server")

	// Start the HTTP server
	log.Fatalln(http.ListenAndServe(":8080", nil))
}
