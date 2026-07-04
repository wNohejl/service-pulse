# ServicePulse — Phases

Target theme: reliability/platform engineering (telemetry, HA, build/release, security baselines) in Go.
Cap: 6–8 h/week (learning-roadmap). Each phase exits with something demonstrable.

## Phase 0 — Walking skeleton (~1 week)
- [ ] Install Go toolchain; `go version` works (Go itself is a learning objective — first Go project)
- [ ] `pulse serve`: HTTP service with `/healthz` + one real endpoint, graceful shutdown
- [ ] GitHub Actions CI: build + `go vet` + tests on every push
- *Exit:* clone → `go run` → curl healthz → green CI badge. **Learning:** Go modules, HTTP, project layout.

## Phase 1 — Deep telemetry (~2 weeks)
- [ ] Metrics registry: request count, latency histogram (p50/p95/p99), error rate
- [ ] `/metrics` endpoint (Prometheus text format — industry standard, no dependency on running Prometheus)
- [ ] Error-budget tracker: rolling window SLO (e.g. 99.5% success) with remaining-budget calculation
- *Exit:* load the service with a script, watch p99 and error budget move. **Learning:** SLO/error-budget vocabulary, histogram buckets.

## Phase 2 — Watch: HA patterns (~2 weeks)
- [ ] `pulse watch`: scrape health/metrics from N configured services
- [ ] Circuit breaker: open on consecutive failures, half-open probes, close on recovery
- [ ] Supervisor: restart a failing local process with backoff; fence (stop routing) when restart loops
- *Exit:* kill the demo service mid-load; watch detects, restarts, and the breaker rides through. **Learning:** HA patterns as code, not slides.

## Phase 3 — Gate: build/release + hardened baseline (~2 weeks)
- [ ] `pulse gate`: runs `go test`, then lints runtime config against `baseline.yaml` (TLS required, no debug mode, no default credentials, ports allowlisted)
- [ ] Gate consults the error budget: ship blocked if budget exhausted (release engineering meets SLOs)
- [ ] CI wires `gate` in as the release job
- *Exit:* a deliberately bad config is refused with a clear report; a good one ships. **Learning:** security/compliance-by-design, release gating.

## Phase 4 — Polish + story (optional, ~1 week)
- [ ] Chaos flag: inject latency/errors into serve to demo watch+gate reacting (mirrors the BMS fault-injection story in SRE vocabulary)
- [ ] Technical write-up (bms-review pattern) → publishable
- *Exit:* 5-minute demo runs from one terminal.

## Postings this evidences
- Starbucks Engineer Senior+ (telemetry, HA, build/release, hardened baselines, Go) — vault `Career/jobs/starbucks-engineer-senior.md`
- Any SRE/platform/reliability posting — the theme generalizes.
