// Package telemetry provides a dependency-free metrics registry:
// request counting, latency histograms with fixed buckets, and a
// rolling error budget against a configured SLO.
package telemetry

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Bucket upper bounds in milliseconds, Prometheus-style cumulative.
var defaultBuckets = []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500}

// Registry collects request outcomes for one service.
type Registry struct {
	mu       sync.Mutex
	requests int64
	errors   int64
	buckets  []float64
	counts   []int64   // cumulative-style on export; stored per-bucket here
	lats     []float64 // recent latencies (ms) for percentile queries
	maxLats  int

	slo    float64 // success target, e.g. 0.995
	window []outcome
	span   time.Duration
}

type outcome struct {
	at time.Time
	ok bool
}

// New creates a Registry with the given SLO success target (0 < slo < 1)
// and rolling window for the error budget.
func New(slo float64, window time.Duration) *Registry {
	return &Registry{
		buckets: defaultBuckets,
		counts:  make([]int64, len(defaultBuckets)+1), // +1 = +Inf
		maxLats: 4096,
		slo:     slo,
		span:    window,
	}
}

// Observe records one request outcome and its latency.
func (r *Registry) Observe(latency time.Duration, ok bool) {
	ms := float64(latency.Microseconds()) / 1000.0
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests++
	if !ok {
		r.errors++
	}
	idx := len(r.buckets) // +Inf slot
	for i, ub := range r.buckets {
		if ms <= ub {
			idx = i
			break
		}
	}
	r.counts[idx]++
	r.lats = append(r.lats, ms)
	if len(r.lats) > r.maxLats {
		r.lats = r.lats[len(r.lats)-r.maxLats:]
	}
	r.window = append(r.window, outcome{time.Now(), ok})
	r.trimLocked()
}

func (r *Registry) trimLocked() {
	cut := time.Now().Add(-r.span)
	i := 0
	for ; i < len(r.window) && r.window[i].at.Before(cut); i++ {
	}
	r.window = r.window[i:]
}

// Percentile returns the pth percentile (0-100) of recent latencies in ms.
func (r *Registry) Percentile(p float64) float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.lats) == 0 {
		return 0
	}
	s := append([]float64(nil), r.lats...)
	sort.Float64s(s)
	idx := int(p / 100 * float64(len(s)-1))
	return s[idx]
}

// Budget reports the error budget over the rolling window: the success
// rate, the budget fraction remaining (1 = untouched, <=0 = exhausted),
// and the number of observations it is based on.
func (r *Registry) Budget() (successRate, remaining float64, n int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.trimLocked()
	n = len(r.window)
	if n == 0 {
		return 1, 1, 0
	}
	okCount := 0
	for _, o := range r.window {
		if o.ok {
			okCount++
		}
	}
	successRate = float64(okCount) / float64(n)
	allowed := 1 - r.slo // error allowance
	used := 1 - successRate
	if allowed <= 0 {
		remaining = 0
	} else {
		remaining = 1 - used/allowed
	}
	return successRate, remaining, n
}

// PrometheusText renders the registry in Prometheus exposition format.
func (r *Registry) PrometheusText(service string) string {
	r.mu.Lock()
	requests, errors := r.requests, r.errors
	counts := append([]int64(nil), r.counts...)
	r.mu.Unlock()
	successRate, remaining, _ := r.Budget()

	var b strings.Builder
	l := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }
	l("# TYPE pulse_requests_total counter")
	l(`pulse_requests_total{service=%q} %d`, service, requests)
	l("# TYPE pulse_errors_total counter")
	l(`pulse_errors_total{service=%q} %d`, service, errors)
	l("# TYPE pulse_request_duration_ms histogram")
	cum := int64(0)
	for i, ub := range r.buckets {
		cum += counts[i]
		l(`pulse_request_duration_ms_bucket{service=%q,le="%g"} %d`, service, ub, cum)
	}
	cum += counts[len(counts)-1]
	l(`pulse_request_duration_ms_bucket{service=%q,le="+Inf"} %d`, service, cum)
	l("# TYPE pulse_success_rate gauge")
	l(`pulse_success_rate{service=%q} %.6f`, service, successRate)
	l("# TYPE pulse_error_budget_remaining gauge")
	l(`pulse_error_budget_remaining{service=%q} %.6f`, service, remaining)
	return b.String()
}
