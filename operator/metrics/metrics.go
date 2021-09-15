package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	githubRateLimitCollector = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "kubeactions",
			Subsystem: "github",
			Name:      "rate_limit",
			Help:      "Current GitHub Rate Limit.",
		},
		[]string{
			"client",
		},
	)

	githubRateRemainingCollector = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "kubeactions",
			Subsystem: "github",
			Name:      "rate_remaining",
			Help:      "Current GitHub Rate Remaining.",
		},
		[]string{
			"client",
		},
	)

	githubAPICallsCollector = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kubeactions",
			Subsystem: "github",
			Name:      "api_calls",
			Help:      "Number of GitHub API Calls.",
		},
		[]string{
			"client",
			"request",
			"response",
		},
	)

	githubCacheHitCollector = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kubeactions",
			Subsystem: "github",
			Name:      "cache_hit",
			Help:      "GitHub Cache Hits.",
		},
		[]string{
			"cache",
			"hit",
		},
	)

	githubActionsEventCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kubeactions",
			Subsystem: "actions",
			Name:      "events",
		},
		[]string{
			"repository",
			"runner",
			"event",
		},
	)
)

func init() {
	metrics.Registry.MustRegister(
		githubRateLimitCollector,
		githubRateRemainingCollector,
		githubAPICallsCollector,
		githubCacheHitCollector,
		githubActionsEventCounter,
	)
}

type observer struct {
	start   time.Time
	observe func(time.Duration)
}

func newObserver(f func(time.Duration)) *observer {
	return &observer{
		start:   time.Now(),
		observe: f,
	}
}

func (o *observer) ObserveDeffered() {
	o.observe(time.Since(o.start))
}

func SetGitHubRateLimitCollector(clientName string, limit int) {
	githubRateLimitCollector.WithLabelValues(clientName).Set(float64(limit))
}

func SetGitHubRateRemainingCollector(clientName string, remaining int) {
	githubRateRemainingCollector.WithLabelValues(clientName).Set(float64(remaining))
}

func IncGitHubAPICallsCollector(clientName string, request string, response string) {
	githubAPICallsCollector.WithLabelValues(clientName, request, response).Inc()
}

func IncGitHubCacheHitCollector(cacheName string, hit bool) {
	githubCacheHitCollector.WithLabelValues(cacheName, strconv.FormatBool(hit)).Inc()
}

func IncGitHubActionsEventCounter(repository, runner, event string) {
	githubActionsEventCounter.WithLabelValues(repository, runner, event).Inc()
}
