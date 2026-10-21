---
"cachestore-go-resilient": minor
---

Add the `resilient` module: `Bounded` (per-operation timeout), `Logged` (rate-limited error logging), `New` (logged primary + in-memory Fallback with transition hooks) and `Prober` (recovery detection for callers that skip the cache while degraded). No third-party dependencies.
