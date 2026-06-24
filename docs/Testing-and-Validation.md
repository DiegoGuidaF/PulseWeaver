<!--
PLACEHOLDER — to be written in a dedicated session that reads the actual
testbench, pprof, and benchmark outputs. Do not pad this with prose until the
real results exist; an empty-but-honest page beats an impressive-sounding one.

WHAT THIS PAGE IS FOR
  Signal to a prospective user that PulseWeaver has been validated like a real
  project, not a weekend hobby — security-tested, performance-profiled, and
  benchmarked — and point them at where that work lives. The README carries only
  a one-line mention and links here for the substance.

WHAT TO COVER (fill from the workspace testbench/ + profiling outputs)
  - Security validation: the pentest checks in `testbench/security/` — CSRF,
    XSS, trivy image scan, and the hey-based load/timing driver. Say what was
    run and the posture it confirms.
  - Frontend quality: `testbench/quality/` — axe accessibility, focus handling,
    Lighthouse, viewport checks.
  - Backend performance: `testbench/profiling/` and the Go benchmarks — what the
    verify-ip hot path and access-log write path were profiled/benchmarked for,
    and the headline result (e.g. per-decision latency from the in-memory cache,
    access-log write throughput). Reference the benchmarks pattern doc.
  - If concrete numbers are published, stamp them with an "as of <date>" so a
    stale figure is obviously stale rather than silently wrong.

TONE (match the rest of the rewritten docs)
  - Accurate over impressive. State what was tested and what it does and does NOT
    prove. No marketing superlatives.
  - The README review surfaced this exact failure mode (the access-log "drops
    entries" wording): never trade an overstatement for a falsehood. If a result
    is partial or experimental (e.g. IPv6), say so plainly.
  - Short, scannable, links out to the testbench rather than reproducing it.
-->

# Testing & Validation

> This page is a stub. The security, accessibility, and performance validation
> results will be written up here from the project's testbench and profiling
> outputs.
