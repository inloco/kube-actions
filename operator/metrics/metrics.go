package metrics

import (
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

	githubActionsEventCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kube_actions_operator",
			Name:      "github_actions_event_counter",
		},
		[]string{
			"runner",
			"event",
		},
	)

	githubActionsEventConsumeDurationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "kube_actions_operator",
			Name:      "github_actions_event_consume_duration_histogram",
			Buckets:   []float64{.1, .5, 1, 2.5, 5, 7.5, 10, 15, 20},
		},
		[]string{
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
		githubActionsEventCounter,
		githubActionsEventConsumeDurationHistogram,
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

func SetGithubRateLimitCollector(clientName string, limit int) {
	githubRateLimitCollector.WithLabelValues(clientName).Set(float64(limit))
}

func SetGithubRateRemainingCollector(clientName string, remaining int) {
	githubRateRemainingCollector.WithLabelValues(clientName).Set(float64(remaining))
}

func IncGithubAPICallsCollector(clientName, request, response string) {
	githubAPICallsCollector.WithLabelValues(clientName, request, response).Inc()
}

func IncGithubActionsEventCounter(runner, event string) {
	githubActionsEventCounter.WithLabelValues(runner, event).Inc()
}

func ObserveGithubActionsEventConsumeDuration(runner, event string) *observer {
	promObserver := githubActionsEventConsumeDurationHistogram.WithLabelValues(runner, event)
	return newObserver(func(duration time.Duration) {
		promObserver.Observe(duration.Seconds())
	})
}
