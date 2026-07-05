package telemetry

import (
	"strings"
	"testing"
	"time"
)

func TestBudgetFullWhenAllOk(t *testing.T) {
	r := New(0.995, time.Minute)
	for i := 0; i < 100; i++ {
		r.Observe(10*time.Millisecond, true)
	}
	rate, remaining, n := r.Budget()
	if n != 100 || rate != 1 || remaining != 1 {
		t.Fatalf("want full budget, got rate=%v remaining=%v n=%d", rate, remaining, n)
	}
}

func TestBudgetExhausted(t *testing.T) {
	r := New(0.995, time.Minute) // 0.5% error allowance
	for i := 0; i < 99; i++ {
		r.Observe(time.Millisecond, true)
	}
	r.Observe(time.Millisecond, false) // 1% errors = 2x the allowance
	_, remaining, _ := r.Budget()
	if remaining > 0 {
		t.Fatalf("budget should be exhausted, remaining=%v", remaining)
	}
}

func TestPercentileOrdering(t *testing.T) {
	r := New(0.99, time.Minute)
	for _, ms := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 100} {
		r.Observe(time.Duration(ms)*time.Millisecond, true)
	}
	if p50, p99 := r.Percentile(50), r.Percentile(99); p50 >= p99 {
		t.Fatalf("p50 (%v) should be < p99 (%v)", p50, p99)
	}
}

func TestPrometheusTextShape(t *testing.T) {
	r := New(0.995, time.Minute)
	r.Observe(3*time.Millisecond, true)
	r.Observe(300*time.Millisecond, false)
	out := r.PrometheusText("demo")
	for _, want := range []string{
		`pulse_requests_total{service="demo"} 2`,
		`pulse_errors_total{service="demo"} 1`,
		`le="+Inf"} 2`,
		"pulse_error_budget_remaining",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}
