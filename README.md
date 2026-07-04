# ServicePulse

A small service-reliability toolkit in **Go**: run services with deep telemetry, watch their health, and gate releases on automated checks — the tooling layer that lets teams build, release, and operate software with confidence.

Three parts, one binary:

- **`pulse serve`** — wraps a demo service with structured health (`/healthz`, `/readyz`), metrics, and deep telemetry (latency histograms, error budgets)
- **`pulse watch`** — a monitor that scrapes health/telemetry from N services, detects degradation, and restarts/fences failing instances (high-availability patterns: circuit breaker, self-healing supervision)
- **`pulse gate`** — a release gate CLI: refuses to ship unless tests pass, config matches a **hardened baseline** (no debug flags, TLS on, no default credentials), and the previous version's error budget is intact

Why it exists: purpose-built to demonstrate reliability engineering end-to-end — telemetry, HA operation, and build/release discipline — in a codebase small enough to read in one sitting.

## Quickstart

```bash
go run ./cmd/pulse serve   # demo service with telemetry on :8080
go run ./cmd/pulse watch   # supervise it
go run ./cmd/pulse gate    # run the release checks
```

## Architecture

```
cmd/pulse/        CLI entry (serve | watch | gate)
internal/telemetry/   metrics registry, latency histograms, error budget
internal/watch/       scraper, circuit breaker, supervisor (restart policy)
internal/gate/        check runners: tests, hardened-config lint, budget check
baseline.yaml         the hardened configuration baseline gate enforces
```

Progress and per-phase exit criteria: [PHASES.md](PHASES.md)

## License

MIT
