// Package serve runs the demo service: a small HTTP API instrumented
// with telemetry, structured health endpoints, and graceful shutdown.
package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/wNohejl/service-pulse/internal/telemetry"
)

// Options configures the demo service.
type Options struct {
	Addr      string
	Service   string
	SLO       float64
	Window    time.Duration
	ChaosRate float64 // 0..1 probability an /api/brew request fails (demo lever)
}

// Run starts the service and blocks until SIGINT or ctx cancellation.
func Run(ctx context.Context, o Options) error {
	if o.Addr == "" {
		o.Addr = ":8080"
	}
	if o.Service == "" {
		o.Service = "demo"
	}
	if o.SLO == 0 {
		o.SLO = 0.995
	}
	if o.Window == 0 {
		o.Window = 5 * time.Minute
	}
	reg := telemetry.New(o.SLO, o.Window)
	ready := false

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": o.Service})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		if !ready {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "starting"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ready"})
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprint(w, reg.PrometheusText(o.Service))
	})
	mux.HandleFunc("GET /api/brew", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// simulated work with a small tail
		d := time.Duration(2+rand.Intn(8)) * time.Millisecond
		if rand.Float64() < 0.05 {
			d += time.Duration(rand.Intn(200)) * time.Millisecond
		}
		time.Sleep(d)
		if o.ChaosRate > 0 && rand.Float64() < o.ChaosRate {
			reg.Observe(time.Since(start), false)
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "chaos"})
			return
		}
		reg.Observe(time.Since(start), true)
		writeJSON(w, http.StatusOK, map[string]any{"brew": "espresso", "took_ms": d.Milliseconds()})
	})
	mux.HandleFunc("GET /api/stats", func(w http.ResponseWriter, _ *http.Request) {
		rate, remaining, n := reg.Budget()
		writeJSON(w, http.StatusOK, map[string]any{
			"p50_ms": reg.Percentile(50), "p95_ms": reg.Percentile(95), "p99_ms": reg.Percentile(99),
			"success_rate": rate, "error_budget_remaining": remaining, "window_n": n,
		})
	})

	srv := &http.Server{Addr: o.Addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	ready = true
	log.Printf("pulse serve: %s listening on %s (SLO %.3f, chaos %.2f)", o.Service, o.Addr, o.SLO, o.ChaosRate)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	select {
	case err := <-errc:
		return err
	case <-stop:
	case <-ctx.Done():
	}
	log.Print("pulse serve: shutting down")
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutCtx)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
