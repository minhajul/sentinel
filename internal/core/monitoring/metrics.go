package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status"},
	)

	ConsumerProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "consumer_processing_duration_seconds",
			Help:    "Duration of Kafka consumer message processing in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"topic", "status"},
	)
)
