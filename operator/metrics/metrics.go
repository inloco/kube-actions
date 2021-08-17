package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	githubActionsEventCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "kube_actions_operator",
		Name:      "github_actions_event_counter",
	}, []string{"runner", "event"})

	githubActionsEventConsumeDurationHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "kube_actions_operator",
		Name:      "github_actions_event_consume_duration_histogram",
		Buckets:   []float64{.1, .5, 1, 2.5, 5, 7.5, 10, 15, 20},
	}, []string{"runner", "event"})
)

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

func Init() {
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":9102", nil)
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
